package shop

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ListItems godoc
// @Summary List shop items
// @Description Lists purchasable and enabled item definitions.
// @Tags shop
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} ItemListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /shop/items [get]
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	redeemed, err := h.redeemedItemIDs(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("shop items unavailable", "shop_items_redeemed_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ItemListResponse{
		Items: shopItemResponses(shopItems(h.content), redeemed),
	})
}

// GetItem godoc
// @Summary Get shop item
// @Description Returns one purchasable and enabled item definition.
// @Tags shop
// @Produce json
// @Security AuthCookieAuth
// @Param itemID path string true "Item ID"
// @Success 200 {object} ItemDetailResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /shop/items/{itemID} [get]
func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	itemID := chi.URLParam(r, "itemID")
	item, ok := shopItemByID(h.content, itemID)
	if !ok {
		httpx.WriteProblem(w, r, httpx.NotFound("shop item not found"))
		return
	}

	redeemed, err := h.itemRedeemed(r.Context(), player.ID, item.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("shop item unavailable", "shop_item_redeemed_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ItemDetailResponse{
		Item: shopItemResponse(item, redeemed),
	})
}

func shopItems(store *content.Store) []content.Item {
	items := store.ListItems()
	out := make([]content.Item, 0, len(items))
	for _, item := range items {
		if item.Purchasable && item.Enabled {
			out = append(out, item)
		}
	}
	return out
}

func shopItemByID(store *content.Store, itemID string) (content.Item, bool) {
	item, ok := store.GetItem(itemID)
	if !ok || !item.Purchasable || !item.Enabled {
		return content.Item{}, false
	}
	return item, true
}

func shopItemResponses(items []content.Item, redeemed map[string]struct{}) []ShopItemResponse {
	if len(items) == 0 {
		return nil
	}

	out := make([]ShopItemResponse, 0, len(items))
	for _, item := range items {
		_, isRedeemed := redeemed[item.ID]
		out = append(out, shopItemResponse(item, isRedeemed))
	}
	return out
}

func shopItemResponse(item content.Item, redeemed bool) ShopItemResponse {
	return ShopItemResponse{
		ID:             item.ID,
		Name:           item.Name,
		Type:           item.Type,
		Rarity:         item.Rarity,
		Description:    item.Description,
		IconPath:       item.IconPath,
		Source:         item.Source,
		PriceOpenPower: item.PriceOpenPower,
		Locked:         item.Locked,
		Redeemed:       redeemed && !item.Repeatable,
		Repeatable:     item.Repeatable,
	}
}

func (h *Handler) redeemedItemIDs(ctx context.Context, playerID string) (map[string]struct{}, error) {
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
	out := make(map[string]struct{}, len(purchases))
	for _, purchase := range purchases {
		out[purchase.ItemID] = struct{}{}
	}
	return out, nil
}

func (h *Handler) itemRedeemed(ctx context.Context, playerID string, itemID string) (bool, error) {
	count, err := h.db.Collection(mongomodel.ShopPurchasesCollection).CountDocuments(ctx, bson.M{
		"player_id": playerID,
		"item_id":   itemID,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
