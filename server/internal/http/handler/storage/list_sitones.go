package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListSitones godoc
// @Summary List owned sitones
// @Description Lists sitones owned by the current player. Sitones are gained from activities and used in matches or crafting.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.SitoneListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/sitones [get]
func (h *Handler) ListSitones(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.SitoneListResponse{
		Sitones: []apimodel.SitoneSummary{
			exampleSitone(),
		},
	})
}
