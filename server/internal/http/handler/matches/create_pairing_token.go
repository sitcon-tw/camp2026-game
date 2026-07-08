package matches

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// CreatePairingToken godoc
// @Summary Refresh on-site match pairing token
// @Description Returns a new short-lived QR pairing token for the authenticated host of a waiting match.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Param matchID path string true "Match ID"
// @Success 200 {object} MatchPairingResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/pairing-token [post]
func (h *Handler) CreatePairingToken(w http.ResponseWriter, r *http.Request) {
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
	if !matchCanIssuePairingToken(match, player.ID) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "match pairing is not available"))
		return
	}
	if err := h.ensureMaintenanceAllowsNewMatch(r.Context(), "match pairing failed"); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	response, err := h.pairingResponse(r.Context(), match, player.ID)
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}
