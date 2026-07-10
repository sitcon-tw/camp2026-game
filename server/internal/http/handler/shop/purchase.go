package shop

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

const purchaseQuantity = 1

var errInsufficientOpenPower = errors.New("insufficient open power")

// Purchase godoc
// @Summary Purchase shop item
// @Description Purchases one enabled shop item, deducts open power, and adds it to the current player's item bag.
// @Tags shop
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body PurchaseRequest true "Purchase request"
// @Success 201 {object} PurchaseResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /shop/purchases [post]
func (h *Handler) Purchase(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	var body PurchaseRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.ItemID = strings.TrimSpace(body.ItemID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	item, found := shopItemByID(h.content, body.ItemID)
	if !found {
		httpx.WriteProblem(w, r, httpx.NotFound("shop item not found"))
		return
	}
	if item.Locked {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusLocked, "shop item is locked"))
		return
	}

	result, err := h.purchaseItem(r.Context(), player.ID, item)
	if err != nil {
		if errors.Is(err, errInsufficientOpenPower) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient open power"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("purchase failed", "shop_purchase_failed", err))
		return
	}
	h.publishPurchaseEvent(player.ID, item)

	httpx.WriteJSON(w, http.StatusCreated, PurchaseResponse{
		PurchaseID:     result.purchaseID,
		ItemID:         item.ID,
		Quantity:       purchaseQuantity,
		PriceOpenPower: item.PriceOpenPower,
		OpenPower:      result.openPower,
		Item:           shopItemResponse(item, true),
	})
}

func (h *Handler) publishPurchaseEvent(playerID string, item content.Item) {
	if h.broker == nil {
		return
	}
	h.broker.Publish(playerID, playerevents.Event{
		Name: "reward_granted",
		Reward: &playerevents.RewardGrantedEvent{
			Kind:       "item",
			RefID:      item.ID,
			Name:       item.Name,
			Quantity:   purchaseQuantity,
			ItemType:   item.Type,
			IconPath:   item.IconPath,
			Source:     "shop_purchase",
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
	})
}

type purchaseResult struct {
	purchaseID string
	openPower  int
}

func (h *Handler) purchaseItem(ctx context.Context, playerID string, item content.Item) (purchaseResult, error) {
	releaseLock, err := openpower.AcquirePlayerLock(ctx, h.db, playerID)
	if err != nil {
		return purchaseResult{}, err
	}
	defer releaseLock()

	if h.client == nil {
		return h.purchaseItemWithoutTransaction(ctx, playerID, item)
	}

	result, err := h.purchaseItemWithTransaction(ctx, playerID, item)
	if err != nil && transactionUnsupported(err) {
		return h.purchaseItemWithoutTransaction(ctx, playerID, item)
	}
	return result, err
}

func (h *Handler) purchaseItemWithTransaction(ctx context.Context, playerID string, item content.Item) (purchaseResult, error) {
	session, err := h.client.StartSession()
	if err != nil {
		return purchaseResult{}, err
	}
	defer session.EndSession(ctx)

	var result purchaseResult
	err = mongo.WithSession(ctx, session, func(ctx context.Context) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = session.AbortTransaction(context.Background())
			}
		}()

		var err error
		result, err = h.purchaseItemWithoutTransaction(ctx, playerID, item)
		if err != nil {
			return err
		}

		if err := session.CommitTransaction(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	})
	if err != nil {
		return purchaseResult{}, err
	}
	return result, nil
}

func (h *Handler) purchaseItemWithoutTransaction(ctx context.Context, playerID string, item content.Item) (purchaseResult, error) {
	openPower, err := h.sumOpenPower(ctx, playerID)
	if err != nil {
		return purchaseResult{}, err
	}
	if openPower < item.PriceOpenPower {
		return purchaseResult{}, errInsufficientOpenPower
	}

	now := time.Now()
	purchaseID := newID("purchase")
	if err := h.insertPurchase(ctx, purchaseID, playerID, item, now); err != nil {
		return purchaseResult{}, err
	}
	if err := h.insertOpenPowerDeduction(ctx, purchaseID, playerID, item.PriceOpenPower, now); err != nil {
		return purchaseResult{}, err
	}
	if err := h.incrementPlayerItem(ctx, playerID, item.ID); err != nil {
		return purchaseResult{}, err
	}

	return purchaseResult{
		purchaseID: purchaseID,
		openPower:  openPower - item.PriceOpenPower,
	}, nil
}

func transactionUnsupported(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) &&
		commandError.HasErrorCodeWithMessage(20, "Transaction numbers")
}

func (h *Handler) insertPurchase(ctx context.Context, purchaseID string, playerID string, item content.Item, createdAt time.Time) error {
	_, err := h.db.Collection(mongomodel.ShopPurchasesCollection).InsertOne(ctx, mongomodel.ShopPurchase{
		ID:             purchaseID,
		PlayerID:       playerID,
		ItemID:         item.ID,
		Quantity:       purchaseQuantity,
		PriceOpenPower: item.PriceOpenPower,
		Repeatable:     item.Repeatable,
		CreatedAt:      createdAt,
	})
	return err
}

func (h *Handler) insertOpenPowerDeduction(ctx context.Context, purchaseID string, playerID string, price int, createdAt time.Time) error {
	_, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        newID("open_power"),
		PlayerID:  playerID,
		Amount:    -price,
		Reason:    "shop_purchase",
		Source:    purchaseID,
		CreatedAt: createdAt,
	})
	return err
}

func (h *Handler) incrementPlayerItem(ctx context.Context, playerID string, itemID string) error {
	_, err := h.db.Collection(mongomodel.PlayerItemsCollection).UpdateOne(
		ctx,
		bson.M{
			"player_id": playerID,
			"item_id":   itemID,
		},
		bson.M{
			"$setOnInsert": bson.M{
				"_id":       newID("player_item"),
				"player_id": playerID,
				"item_id":   itemID,
			},
			"$inc": bson.M{"quantity": purchaseQuantity},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (h *Handler) sumOpenPower(ctx context.Context, playerID string) (int, error) {
	return openpower.TotalForPlayer(ctx, h.db, playerID)
}

func openPowerTotalFromCursor(ctx context.Context, cursor *mongo.Cursor) (int, error) {
	return openpower.TotalFromCursor(ctx, cursor)
}

func openPowerTotalPipeline(playerID string) mongo.Pipeline {
	return openpower.TotalPipeline(playerID)
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
