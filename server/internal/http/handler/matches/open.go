package matches

import (
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
		if errors.Is(err, mongo.ErrNoDocuments) {
			httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match lookup failed", "match_open_lookup_failed", err))
		return
	}

	h.writeAdvancedMatchState(w, r, match, player.ID)
}
