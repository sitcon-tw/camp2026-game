package activities

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListActivities godoc
// @Summary List activities
// @Description Lists activities that can grant sitones to the current player.
// @Tags Activities
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.ActivityListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /activities [get]
func (h *Handler) ListActivities(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.ActivityListResponse{
		Activities: []apimodel.ActivitySummary{
			exampleActivity(),
		},
	})
}

func exampleActivity() apimodel.ActivitySummary {
	return apimodel.ActivitySummary{
		ActivityID:  "booth-linux-101",
		Name:        "Linux 101 Booth",
		Description: "Complete the booth challenge to receive a sitone.",
		Status:      "claimable",
		Reward: apimodel.Reward{
			Sitones: []apimodel.SitoneGrant{
				{DefinitionID: "sitone-engineering", Quantity: 1},
			},
		},
	}
}
