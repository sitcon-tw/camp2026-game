package admin

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/gifthistory"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GiftHistory godoc
// @Summary Get admin gift history
// @Description Returns recent staff gift history across all players.
// @Tags admin
// @Produce json
// @Success 200 {object} gifthistory.Response
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/gift-history [get]
func (h *Handler) GiftHistory(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	entries, err := gifthistory.List(r.Context(), h.db, h.content, gifthistory.Filter{})
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("gift history unavailable", "admin_gift_history_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, gifthistory.Response{Entries: entries})
}
