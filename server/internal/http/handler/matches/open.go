package matches

import (
	"context"
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// Open godoc
// @Summary Get current open match
// @Description Returns the current waiting or active match for the authenticated participant.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} OpenMatchResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/open [get]
func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	match, err := h.findOpenParticipantMatch(r.Context(), player.ID)
	if err != nil {
		writeOpenMatchLookupProblem(w, r, err)
		return
	}

	state, err := h.openMatchState(r.Context(), match.ID, player.ID)
	if err != nil {
		writeOpenMatchLookupProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}

func (h *Handler) openMatchState(ctx context.Context, matchID string, playerID string) (MatchStateResponse, error) {
	session, err := h.sessions.GetOrLoad(ctx, matchID)
	if err != nil {
		return MatchStateResponse{}, err
	}
	return session.State(ctx, playerID)
}

func (h *Handler) writeExistingOpenMatchState(w http.ResponseWriter, r *http.Request, playerID string) bool {
	match, err := h.findOpenParticipantMatch(r.Context(), playerID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match lookup failed", "match_open_lookup_failed", err))
		return true
	}

	state, err := h.openMatchState(r.Context(), match.ID, playerID)
	if err != nil {
		if openMatchUnavailable(err) {
			return false
		}
		httpx.WriteProblem(w, r, err)
		return true
	}
	httpx.WriteJSON(w, http.StatusOK, state)
	return true
}

func writeOpenMatchLookupProblem(w http.ResponseWriter, r *http.Request, err error) {
	if openMatchUnavailable(err) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return
	}
	httpx.WriteProblem(w, r, httpx.InternalServerError("match lookup failed", "match_open_lookup_failed", err))
}

func openMatchUnavailable(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, errMatchNotOpen)
}
