package me

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/gifthistory"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GiftHistory godoc
// @Summary Get current player gift history
// @Description Returns the authenticated player's staff gift history.
// @Tags me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} gifthistory.Response
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/gift-history [get]
func (h *Handler) GiftHistory(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}
	if h.content == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("content store is unavailable"))
		return
	}

	entries, err := gifthistory.List(r.Context(), h.db, h.content, gifthistory.Filter{
		RecipientPlayerID: player.ID,
	})
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("gift history unavailable", "me_gift_history_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, gifthistory.Response{Entries: entries})
}
