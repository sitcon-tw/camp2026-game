package home

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetMe godoc
// @Summary Get current player
// @Description Returns the current authenticated player's profile, team, and open power balance.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.MeResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me [get]
func (h *Handler) GetMe(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.MeResponse{
		Player: apimodel.AuthPlayerSummary{
			PlayerID:  "7H9K2Q",
			Nickname:  "Alice",
			OpenPower: 1280,
			AvatarURL: "https://example.test/avatar/alice.png",
			Team: apimodel.AuthTeamSummary{
				TeamID: "8M4RXP",
				Name:   "Blue Team",
			},
		},
	})
}
