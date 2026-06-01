package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListRecipes godoc
// @Summary List visible crafting recipes
// @Description Lists crafting recipes visible to the current player, including whether each recipe is currently craftable.
// @Tags Crafting
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.RecipeListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /crafting/recipes [get]
func (h *Handler) ListRecipes(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.RecipeListResponse{
		Recipes: []apimodel.RecipeSummary{
			exampleRecipe(true),
		},
	})
}
