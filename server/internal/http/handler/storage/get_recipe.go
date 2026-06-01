package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetRecipe godoc
// @Summary Get crafting recipe
// @Description Returns one crafting recipe visible to the current player.
// @Tags Crafting
// @Produce json
// @Security AuthCookieAuth
// @Param recipeID path string true "Recipe ID"
// @Success 200 {object} apimodel.RecipeSummary
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /crafting/recipes/{recipeID} [get]
func (h *Handler) GetRecipe(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, exampleRecipe(true))
}

func exampleRecipe(craftable bool) apimodel.RecipeSummary {
	return apimodel.RecipeSummary{
		RecipeID:             "recipe_engineering_skin",
		Name:                 "Engineering Skin",
		RequiredSitoneType:   "engineering",
		RequiredItemID:       "item-camp-sticker",
		RequiredItemQuantity: 1,
		OutputDefinitionID:   "sitone-engineering-skin",
		Unlocked:             true,
		Craftable:            craftable,
		AcquisitionHint:      "Complete engineering activities.",
	}
}
