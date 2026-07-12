package fusions

import (
	"context"
	"fmt"
	"net/http"
	"slices"
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

const (
	fillActionPurchase = "purchase"
	fillActionFusion   = "fusion"
)

type fillState struct {
	handler            *Handler
	playerID           string
	inventory          inventoryCounts
	openPower          int
	itemPurchaseCounts map[string]int
	producers          map[string]producerRecipe
	filled             map[string]FillMaterialResult
	failed             map[string]FillMaterialFailure
	spent              int
}

type producerRecipe struct {
	recipe content.FusionRecipe
}

// FillMissingMaterials godoc
// @Summary Fill missing fusion materials
// @Description Automatically purchases or crafts missing materials for a fusion recipe when possible, then returns refreshed recipe availability and failure details for anything that could not be filled.
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
	releaseLock, err := openpower.AcquirePlayerLock(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_lock_failed", err))
		return
	}
	defer releaseLock()

	inventory, err := h.playerInventory(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_inventory_lookup_failed", err))
		return
	}

	currentOpenPower, err := openpower.TotalForPlayer(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_open_power_lookup_failed", err))
		return
	}

	itemPurchaseCounts, err := h.itemPurchaseCounts(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fill missing materials failed", "fusion_fill_redeemed_item_lookup_failed", err))
		return
	}

	state := fillState{
		handler:            h,
		playerID:           player.ID,
		inventory:          inventory,
		openPower:          currentOpenPower,
		itemPurchaseCounts: itemPurchaseCounts,
		producers:          h.producerRecipes(),
		filled:             map[string]FillMaterialResult{},
		failed:             map[string]FillMaterialFailure{},
	}

	missingFound := false
	for _, component := range recipe.Inputs {
		missing := max(component.Quantity-ownedQuantityForComponent(component, state.inventory), 0)
		if missing == 0 {
			continue
		}
		missingFound = true
		state.ensureComponent(r.Context(), component, component.Quantity, nil)
	}
	if !missingFound {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "recipe materials are already complete"))
		return
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
		FilledMaterials:     sortedFillResults(state.filled),
		FailedMaterials:     sortedFillFailures(state.failed),
		PriceOpenPowerSpent: state.spent,
	})
}

func (s *fillState) ensureComponent(ctx context.Context, component content.FusionComponent, required int, path []string) bool {
	if ownedQuantityForComponent(component, s.inventory) >= required {
		return true
	}

	componentKey := fillComponentKey(component.Kind, component.ID)
	if slices.Contains(path, componentKey) {
		s.recordFailure(component, required-ownedQuantityForComponent(component, s.inventory), "合成路徑發生循環，無法自動補足")
		return false
	}
	path = append(path, componentKey)

	for ownedQuantityForComponent(component, s.inventory) < required {
		if component.Kind == content.FusionKindItem {
			if purchased, reason := s.tryPurchase(ctx, component); purchased {
				continue
			} else if reason != "" && !s.canCraft(component) {
				s.recordFailure(component, required-ownedQuantityForComponent(component, s.inventory), reason)
				return false
			}
		}

		if crafted, reason := s.tryCraft(ctx, component, path); crafted {
			continue
		} else if reason != "" {
			s.recordFailure(component, required-ownedQuantityForComponent(component, s.inventory), reason)
			return false
		}

		s.recordFailure(component, required-ownedQuantityForComponent(component, s.inventory), "玩家目前無法自行取得這個材料")
		return false
	}

	return true
}

func (s *fillState) tryPurchase(ctx context.Context, component content.FusionComponent) (bool, string) {
	item, ok := s.handler.content.GetItem(component.ID)
	if !ok {
		return false, "找不到材料資料"
	}
	switch {
	case component.Quantity-ownedQuantityForComponent(component, s.inventory) > fillPurchaseQuantity:
		return false, "缺少數量超過可自動購買上限"
	case !item.Enabled:
		return false, "材料目前未開放"
	case !item.Purchasable:
		return false, "材料無法直接購買"
	case item.Locked:
		return false, "材料尚未開放購買"
	case s.itemPurchaseCounts[item.ID] >= content.ShopPurchaseLimit:
		return false, "材料已達購買上限"
	case s.openPower < item.PriceOpenPower:
		return false, "開源力不足，無法購買"
	}
	if err := s.handler.purchaseFillItem(ctx, s.playerID, item); err != nil {
		return false, fillPurchaseFailureReason(err)
	}

	s.itemPurchaseCounts[item.ID]++
	s.openPower -= item.PriceOpenPower
	s.spent += item.PriceOpenPower
	s.inventory.items[item.ID]++
	s.recordFilled(content.FusionKindItem, item.ID, item.Name, fillPurchaseQuantity, fillActionPurchase, "")
	return true, ""
}

func (s *fillState) tryCraft(ctx context.Context, component content.FusionComponent, path []string) (bool, string) {
	producer, ok := s.producers[fillComponentKey(component.Kind, component.ID)]
	if !ok {
		if component.Kind == content.FusionKindSitone {
			return false, "這顆小石沒有可自動模仿的合成路徑"
		}
		return false, ""
	}

	if s.handler.client == nil {
		return false, "自動合成目前暫時不可用"
	}

	for _, input := range producer.recipe.Inputs {
		if !s.ensureComponent(ctx, input, input.Quantity, path) {
			return false, fmt.Sprintf("缺少 %s 的前置材料", componentDisplayName(s.handler.content, component))
		}
	}

	if !recipeAvailable(producer.recipe, s.inventory) {
		return false, fmt.Sprintf("仍無法補齊 %s 的合成材料", componentDisplayName(s.handler.content, component))
	}

	if _, err := s.handler.createFusion(ctx, s.playerID, producer.recipe); err != nil {
		if err == errInsufficientMaterials {
			return false, fmt.Sprintf("%s 的合成材料仍然不足", producer.recipe.Name)
		}
		if err == errFusionTransactionsUnavailable {
			return false, "自動合成目前暫時不可用"
		}
		return false, "自動合成失敗"
	}

	s.applyRecipeToInventory(producer.recipe)
	s.handler.publishFusionEvents(s.playerID, producer.recipe)
	for _, output := range producer.recipe.Outputs {
		s.recordFilled(output.Kind, output.ID, componentDisplayName(s.handler.content, output), output.Quantity, fillActionFusion, producer.recipe.Name)
	}
	return true, ""
}

func (s *fillState) canCraft(component content.FusionComponent) bool {
	_, ok := s.producers[fillComponentKey(component.Kind, component.ID)]
	return ok
}

func (s *fillState) applyRecipeToInventory(recipe content.FusionRecipe) {
	for _, input := range recipe.Inputs {
		s.addInventory(input, -input.Quantity)
	}
	for _, output := range recipe.Outputs {
		s.addInventory(output, output.Quantity)
	}
}

func (s *fillState) addInventory(component content.FusionComponent, delta int) {
	switch component.Kind {
	case content.FusionKindItem:
		s.inventory.items[component.ID] += delta
		if s.inventory.items[component.ID] <= 0 {
			delete(s.inventory.items, component.ID)
		}
	case content.FusionKindSitone:
		s.inventory.sitones[component.ID] += delta
		if s.inventory.sitones[component.ID] <= 0 {
			delete(s.inventory.sitones, component.ID)
		}
	}
}

func (s *fillState) recordFilled(kind string, id string, name string, quantity int, action string, source string) {
	key := fillResultKey(kind, id, action, source)
	record := s.filled[key]
	record.Kind = kind
	record.ID = id
	record.Name = name
	record.Quantity += quantity
	record.Action = action
	record.Source = source
	s.filled[key] = record
}

func (s *fillState) recordFailure(component content.FusionComponent, quantity int, reason string) {
	if quantity <= 0 {
		return
	}
	key := fillResultKey(component.Kind, component.ID, "", reason)
	record := s.failed[key]
	record.Kind = component.Kind
	record.ID = component.ID
	record.Name = componentDisplayName(s.handler.content, component)
	record.Quantity += quantity
	record.Reason = reason
	s.failed[key] = record
}

func (h *Handler) producerRecipes() map[string]producerRecipe {
	out := map[string]producerRecipe{}
	ambiguous := map[string]bool{}
	for _, recipe := range h.content.ListFusionRecipes() {
		if !recipe.Enabled {
			continue
		}
		for _, output := range recipe.Outputs {
			key := fillComponentKey(output.Kind, output.ID)
			if ambiguous[key] {
				continue
			}
			if _, exists := out[key]; exists {
				delete(out, key)
				ambiguous[key] = true
				continue
			}
			out[key] = producerRecipe{recipe: recipe}
		}
	}
	return out
}

func fillComponentKey(kind string, id string) string {
	return kind + ":" + id
}

func fillResultKey(kind string, id string, action string, extra string) string {
	return kind + ":" + id + ":" + action + ":" + extra
}

func sortedFillResults(values map[string]FillMaterialResult) []FillMaterialResult {
	if len(values) == 0 {
		return nil
	}
	out := make([]FillMaterialResult, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	slices.SortFunc(out, func(a, b FillMaterialResult) int {
		if a.Action != b.Action {
			return compareStrings(a.Action, b.Action)
		}
		if a.Name != b.Name {
			return compareStrings(a.Name, b.Name)
		}
		return compareStrings(a.Source, b.Source)
	})
	return out
}

func sortedFillFailures(values map[string]FillMaterialFailure) []FillMaterialFailure {
	if len(values) == 0 {
		return nil
	}
	out := make([]FillMaterialFailure, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	slices.SortFunc(out, func(a, b FillMaterialFailure) int {
		if a.Name != b.Name {
			return compareStrings(a.Name, b.Name)
		}
		return compareStrings(a.Reason, b.Reason)
	})
	return out
}

func compareStrings(a string, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func componentDisplayName(store *content.Store, component content.FusionComponent) string {
	switch component.Kind {
	case content.FusionKindItem:
		if item, ok := store.GetItem(component.ID); ok {
			return item.Name
		}
	case content.FusionKindSitone:
		if sitone, ok := store.GetSitone(component.ID); ok {
			return sitone.Name
		}
	}
	return component.ID
}

func (h *Handler) itemPurchaseCounts(ctx context.Context, playerID string) (map[string]int, error) {
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
	out := make(map[string]int, len(purchases))
	for _, purchase := range purchases {
		out[purchase.ItemID] += max(1, purchase.Quantity)
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
		return "材料已經購買過了"
	}
	return "自動購買失敗"
}
