package fusions

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

const fillPurchaseQuantity = 1

type fillPlanEntry struct {
	component content.FusionComponent
	name      string
	missing   int
	reason    string
	item      content.Item
	fillable  bool
}

// FillMissingMaterials godoc
// @Summary Fill missing fusion materials
// @Description Automatically purchases missing shop materials for a fusion recipe when possible, then returns refreshed recipe availability and failure details for anything that could not be filled.
// @Tags fusions
// @Produce json
// @Security AuthCookieAuth
// @Param recipeID path string true "Recipe ID"
// @Success 200 {object} FillMissingMaterialsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fusions/{recipeID}/fill-missing-materials [post]
func (h *Handler) FillMissingMaterials(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	recipeID := chi.URLParam(r, "recipeID")
	recipe, ok := h.content.GetFusionRecipe(recipeID)
	if !ok || !recipe.Enabled {
		httpx.WriteProblem(w, r, httpx.NotFound("fusion recipe not found"))
		return
	}

	inventory, err := h.playerInventory(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_inventory_lookup_failed", err))
		return
	}

	plan, err := h.buildFillPlan(r.Context(), player.ID, recipe, inventory)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_plan_failed", err))
		return
	}
	if len(plan) == 0 {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "recipe materials are already complete"))
		return
	}

	filled := make([]FillMaterialResult, 0, len(plan))
	failed := make([]FillMaterialFailure, 0, len(plan))
	spent := 0
	currentOpenPower, err := openpower.TotalForPlayer(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_open_power_lookup_failed", err))
		return
	}

	for _, entry := range plan {
		if !entry.fillable {
			failed = append(failed, FillMaterialFailure{
				Kind:     entry.component.Kind,
				ID:       entry.component.ID,
				Name:     entry.name,
				Quantity: entry.missing,
				Reason:   entry.reason,
			})
			continue
		}
		if currentOpenPower < entry.item.PriceOpenPower {
			failed = append(failed, FillMaterialFailure{
				Kind:     entry.component.Kind,
				ID:       entry.component.ID,
				Name:     entry.name,
				Quantity: entry.missing,
				Reason:   "insufficient open power",
			})
			continue
		}
		if err := h.purchaseFillItem(r.Context(), player.ID, entry.item); err != nil {
			failed = append(failed, FillMaterialFailure{
				Kind:     entry.component.Kind,
				ID:       entry.component.ID,
				Name:     entry.name,
				Quantity: entry.missing,
				Reason:   fillPurchaseFailureReason(err),
			})
			continue
		}
		currentOpenPower -= entry.item.PriceOpenPower
		spent += entry.item.PriceOpenPower
		filled = append(filled, FillMaterialResult{
			Kind:     entry.component.Kind,
			ID:       entry.component.ID,
			Name:     entry.name,
			Quantity: fillPurchaseQuantity,
		})
	}

	inventory, err = h.playerInventory(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_inventory_refresh_failed", err))
		return
	}
	response, err := h.recipeResponse(recipe, inventory)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_recipe_response_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, FillMissingMaterialsResponse{
		Recipe:              response,
		FilledMaterials:     filled,
		FailedMaterials:     failed,
		PriceOpenPowerSpent: spent,
	})
}

func (h *Handler) buildFillPlan(ctx context.Context, playerID string, recipe content.FusionRecipe, inventory inventoryCounts) ([]fillPlanEntry, error) {
	redeemedItemIDs := map[string]bool{}
	if h.db != nil {
		var err error
		redeemedItemIDs, err = h.redeemedItemIDs(ctx, playerID)
		if err != nil {
			return nil, err
		}
	}

	plan := make([]fillPlanEntry, 0, len(recipe.Inputs))
	for _, component := range recipe.Inputs {
		missing := max(component.Quantity-ownedQuantityForComponent(component, inventory), 0)
		if missing == 0 {
			continue
		}
		entry := fillPlanEntry{
			component: component,
			missing:   missing,
		}
		switch component.Kind {
		case content.FusionKindSitone:
			sitone, ok := h.content.GetSitone(component.ID)
			if !ok {
				entry.name = component.ID
				entry.reason = "material definition not found"
				break
			}
			entry.name = sitone.Name
			entry.reason = "sitones cannot be filled automatically"
		case content.FusionKindItem:
			item, ok := h.content.GetItem(component.ID)
			if !ok {
				entry.name = component.ID
				entry.reason = "material definition not found"
				break
			}
			entry.name = item.Name
			entry.item = item
			switch {
			case missing > fillPurchaseQuantity:
				entry.reason = "missing quantity exceeds automatic purchase limit"
			case !item.Enabled:
				entry.reason = "material is not available"
			case !item.Purchasable:
				entry.reason = "material is not purchasable automatically"
			case item.Locked:
				entry.reason = "material is locked"
			case redeemedItemIDs[item.ID]:
				entry.reason = "material has already been purchased"
			default:
				entry.fillable = true
			}
		default:
			entry.name = component.ID
			entry.reason = "unsupported material kind"
		}
		plan = append(plan, entry)
	}

	return plan, nil
}

func (h *Handler) redeemedItemIDs(ctx context.Context, playerID string) (map[string]bool, error) {
	cursor, err := h.db.Collection(mongomodel.ShopPurchasesCollection).Find(
		ctx,
		bson.M{"player_id": playerID},
		options.Find().SetSort(bson.D{{Key: "item_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var purchases []mongomodel.ShopPurchase
	if err := cursor.All(ctx, &purchases); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(purchases))
	for _, purchase := range purchases {
		out[purchase.ItemID] = true
	}
	return out, nil
}

func (h *Handler) purchaseFillItem(ctx context.Context, playerID string, item content.Item) error {
	now := time.Now().UTC()
	purchaseID := newID("purchase")

	if err := h.insertFillPurchase(ctx, purchaseID, playerID, item, now); err != nil {
		return err
	}
	if err := h.insertFillOpenPowerDeduction(ctx, purchaseID, playerID, item.PriceOpenPower, now); err != nil {
		return err
	}
	return h.incrementFillPlayerItem(ctx, playerID, item.ID)
}

func (h *Handler) insertFillPurchase(ctx context.Context, purchaseID string, playerID string, item content.Item, createdAt time.Time) error {
	_, err := h.db.Collection(mongomodel.ShopPurchasesCollection).InsertOne(ctx, mongomodel.ShopPurchase{
		ID:             purchaseID,
		PlayerID:       playerID,
		ItemID:         item.ID,
		Quantity:       fillPurchaseQuantity,
		PriceOpenPower: item.PriceOpenPower,
		CreatedAt:      createdAt,
	})
	return err
}

func (h *Handler) insertFillOpenPowerDeduction(ctx context.Context, purchaseID string, playerID string, price int, createdAt time.Time) error {
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

func (h *Handler) incrementFillPlayerItem(ctx context.Context, playerID string, itemID string) error {
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
			"$inc": bson.M{"quantity": fillPurchaseQuantity},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func fillPurchaseFailureReason(err error) string {
	if mongo.IsDuplicateKeyError(err) {
		return "material has already been purchased"
	}
	return "automatic purchase failed"
}
