package matches

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

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
	if matchIsOpen(match) {
		state, err := h.openMatchState(r.Context(), match.ID, player.ID)
		if err != nil {
			if h.writeRecoveredClosedMatchState(w, r, match.ID, player.ID, err) {
				return
			}
			httpx.WriteProblem(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, state)
		return
	}

	state, err := h.buildMatchState(r.Context(), match, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}

func (h *Handler) writeRecoveredClosedMatchState(
	w http.ResponseWriter,
	r *http.Request,
	matchID string,
	playerID string,
	err error,
) bool {
	if !openMatchUnavailable(err) {
		return false
	}

	match, findErr := h.findMatchByID(r.Context(), matchID)
	if errors.Is(findErr, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return true
	}
	if findErr != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match lookup failed", "match_lookup_failed", findErr))
		return true
	}
	if !isParticipant(match, playerID) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return true
	}
	if matchIsOpen(match) {
		return false
	}

	state, buildErr := h.buildMatchState(r.Context(), match, playerID)
	if buildErr != nil {
		httpx.WriteProblem(w, r, buildErr)
		return true
	}
	httpx.WriteJSON(w, http.StatusOK, state)
	return true
}
