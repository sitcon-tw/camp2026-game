package matches

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// CreateMatch godoc
// @Summary Create match
// @Description Starts a Knowledge King player match with selected sitones.
// @Tags Matches
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body apimodel.MatchCreateRequest true "Match creation request"
// @Success 201 {object} apimodel.MatchResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 501 {object} httpx.ProblemDetails
// @Router /matches [post]
func (h *Handler) CreateMatch(w http.ResponseWriter, r *http.Request) {
	var body apimodel.MatchCreateRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	httpx.WriteProblem(w, r, httpx.NotImplemented("match creation is not implemented yet"))
}
