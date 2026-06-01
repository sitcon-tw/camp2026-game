package activities

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ClaimActivity godoc
// @Summary Claim activity reward
// @Description Claims a sitone reward from an activity when the player is eligible.
// @Tags Activities
// @Produce json
// @Security AuthCookieAuth
// @Param activityID path string true "Activity ID"
// @Success 201 {object} apimodel.ActivityClaimResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 501 {object} httpx.ProblemDetails
// @Router /activities/{activityID}/claims [post]
func (h *Handler) ClaimActivity(w http.ResponseWriter, r *http.Request) {
	httpx.WriteProblem(w, r, httpx.NotImplemented("activity reward claims are not implemented yet"))
}
