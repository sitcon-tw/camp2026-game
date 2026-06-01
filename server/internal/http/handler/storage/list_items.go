package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListItems godoc
// @Summary List owned items
// @Description Lists items owned by the current player. Items are bought with open power and used for crafting.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.ItemListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/items [get]
func (h *Handler) ListItems(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.ItemListResponse{
		Items: []apimodel.ItemSummary{
			exampleItem(),
		},
	})
}
