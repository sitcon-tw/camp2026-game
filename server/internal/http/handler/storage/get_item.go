package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetItem godoc
// @Summary Get owned item
// @Description Returns one item stack owned by the current player.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Param itemInstanceID path string true "Item instance ID"
// @Success 200 {object} apimodel.ItemSummary
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/items/{itemInstanceID} [get]
func (h *Handler) GetItem(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, exampleItem())
}

func exampleItem() apimodel.ItemSummary {
	return apimodel.ItemSummary{
		ID:           "I8M4RX",
		DefinitionID: "item-camp-sticker",
		Name:         "Camp Sticker",
		Quantity:     3,
	}
}
