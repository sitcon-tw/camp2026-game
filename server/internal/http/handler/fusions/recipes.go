package fusions

import (
	"context"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ListRecipes godoc
// @Summary List fusion recipes
// @Description Lists enabled fusion recipes and whether the authenticated player has enough materials.
// @Tags fusions
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} RecipeListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fusions/recipes [get]
func (h *Handler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	inventory, err := h.playerInventory(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fusion recipes unavailable", "fusion_recipes_inventory_lookup_failed", err))
		return
	}

	recipes, err := h.recipeResponses(h.content.ListFusionRecipes(), inventory)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fusion recipes are inconsistent", "fusion_recipes_response_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, RecipeListResponse{Recipes: recipes})
}

type inventoryCounts struct {
	items   map[string]int
	sitones map[string]int
}

func (h *Handler) playerInventory(ctx context.Context, playerID string) (inventoryCounts, error) {
	items, err := h.playerItemCounts(ctx, playerID)
	if err != nil {
		return inventoryCounts{}, err
	}
	sitones, err := h.playerSitoneCounts(ctx, playerID)
	if err != nil {
		return inventoryCounts{}, err
	}
	return inventoryCounts{items: items, sitones: sitones}, nil
}

func (h *Handler) playerItemCounts(ctx context.Context, playerID string) (map[string]int, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerItemsCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "item_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.PlayerItem
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(records))
	for _, record := range records {
		out[record.ItemID] = record.Quantity
	}
	return out, nil
}

func (h *Handler) playerSitoneCounts(ctx context.Context, playerID string) (map[string]int, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "sitone_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.PlayerSitone
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(records))
	for _, record := range records {
		out[record.SitoneID] = record.Quantity
	}
	return out, nil
}

func (h *Handler) recipeResponses(recipes []content.FusionRecipe, inventory inventoryCounts) ([]FusionRecipeResponse, error) {
	out := make([]FusionRecipeResponse, 0, len(recipes))
	for _, recipe := range recipes {
		if !recipe.Enabled {
			continue
		}
		response, err := h.recipeResponse(recipe, inventory)
		if err != nil {
			return nil, err
		}
		out = append(out, response)
	}
	return out, nil
}

func (h *Handler) recipeResponse(recipe content.FusionRecipe, inventory inventoryCounts) (FusionRecipeResponse, error) {
	inputs, err := h.componentResponses(recipe.Inputs, inventory)
	if err != nil {
		return FusionRecipeResponse{}, err
	}
	outputs, err := h.componentResponses(recipe.Outputs, inventoryCounts{})
	if err != nil {
		return FusionRecipeResponse{}, err
	}
	return FusionRecipeResponse{
		ID:          recipe.ID,
		BranchID:    recipe.BranchID,
		Type:        recipe.Type,
		StageFrom:   recipe.StageFrom,
		StageTo:     recipe.StageTo,
		Name:        recipe.Name,
		Description: recipe.Description,
		Story:       recipe.Story,
		ReviewTitle: recipe.ReviewTitle,
		ReviewURL:   recipe.ReviewURL,
		Enabled:     recipe.Enabled,
		Available:   recipeAvailable(recipe, inventory),
		Inputs:      inputs,
		Outputs:     outputs,
	}, nil
}

func (h *Handler) componentResponses(components []content.FusionComponent, inventory inventoryCounts) ([]FusionComponentResponse, error) {
	out := make([]FusionComponentResponse, 0, len(components))
	for _, component := range components {
		ownedQuantity := ownedQuantityForComponent(component, inventory)
		response := FusionComponentResponse{
			Kind:            component.Kind,
			ID:              component.ID,
			Quantity:        component.Quantity,
			OwnedQuantity:   ownedQuantity,
			MissingQuantity: max(component.Quantity-ownedQuantity, 0),
		}
		switch component.Kind {
		case content.FusionKindItem:
			item, ok := h.content.GetItem(component.ID)
			if !ok {
				return nil, fmt.Errorf("item %q not found", component.ID)
			}
			response.Name = item.Name
			response.Type = item.Type
			response.Rarity = item.Rarity
			response.IconPath = item.IconPath
			response.Source = item.Source
		case content.FusionKindSitone:
			sitone, ok := h.content.GetSitone(component.ID)
			if !ok {
				return nil, fmt.Errorf("sitone %q not found", component.ID)
			}
			response.Name = sitone.Name
			response.Type = sitone.Type
			response.Rarity = sitone.Rarity
			response.IconPath = sitone.IconPath
			response.AbilityName = sitone.AbilityName
			response.AbilityKind = sitone.AbilityKind
			response.AbilityValue = sitone.AbilityValue
			response.AbilityCount = sitone.AbilityCount
			response.AbilityDescription = sitone.AbilityDescription
		default:
			return nil, fmt.Errorf("unsupported component kind %q", component.Kind)
		}
		out = append(out, response)
	}
	return out, nil
}

func ownedQuantityForComponent(component content.FusionComponent, inventory inventoryCounts) int {
	switch component.Kind {
	case content.FusionKindItem:
		return inventory.items[component.ID]
	case content.FusionKindSitone:
		return inventory.sitones[component.ID]
	default:
		return 0
	}
}

func recipeAvailable(recipe content.FusionRecipe, inventory inventoryCounts) bool {
	for _, input := range recipe.Inputs {
		switch input.Kind {
		case content.FusionKindItem:
			if inventory.items[input.ID] < input.Quantity {
				return false
			}
		case content.FusionKindSitone:
			if inventory.sitones[input.ID] < input.Quantity {
				return false
			}
		default:
			return false
		}
	}
	return true
}
