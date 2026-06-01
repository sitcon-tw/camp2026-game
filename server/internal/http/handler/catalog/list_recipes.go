package catalog

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListRecipes godoc
// @Summary List recipe catalog
// @Description Lists public crafting recipes and acquisition hints.
// @Tags Catalog
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.RecipeListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /catalog/crafting-recipes [get]
func (h *Handler) ListRecipes(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.RecipeListResponse{
		Recipes: []apimodel.RecipeSummary{
			{
				RecipeID:             "recipe_engineering_skin",
				Name:                 "Engineering Skin",
				RequiredSitoneType:   "engineering",
				RequiredItemID:       "item-camp-sticker",
				RequiredItemQuantity: 1,
				OutputDefinitionID:   "sitone-engineering-skin",
				Unlocked:             true,
				Craftable:            false,
				AcquisitionHint:      "Complete engineering missions.",
			},
		},
	})
}
