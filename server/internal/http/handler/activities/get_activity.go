package activities

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetActivity godoc
// @Summary Get activity
// @Description Returns one activity and its sitone reward information.
// @Tags Activities
// @Produce json
// @Security AuthCookieAuth
// @Param activityID path string true "Activity ID"
// @Success 200 {object} apimodel.ActivityDetailResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /activities/{activityID} [get]
func (h *Handler) GetActivity(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.ActivityDetailResponse{
		Activity: exampleActivity(),
	})
}
