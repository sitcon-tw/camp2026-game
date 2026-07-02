package matches

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// Get godoc
// @Summary Get match state
// @Description Returns the current match state for a participant. Active matches reveal current-round answers only during the reveal phase.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} MatchStateResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	match, err := h.findMatchByID(r.Context(), chi.URLParam(r, "matchID"))
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	if !isParticipant(match, player.ID) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return
	}

	h.writeAdvancedMatchState(w, r, match, player.ID)
}
