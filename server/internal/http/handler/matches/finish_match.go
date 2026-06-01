package matches

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// FinishMatch godoc
// @Summary Finish match
// @Description Finalizes a match and grants open power from the result.
// @Tags Matches
// @Produce json
// @Security AuthCookieAuth
// @Param matchID path string true "Match ID"
// @Success 200 {object} apimodel.MatchFinishResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 501 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/finish [post]
func (h *Handler) FinishMatch(w http.ResponseWriter, r *http.Request) {
	httpx.WriteProblem(w, r, httpx.NotImplemented("match finish is not implemented yet"))
}
