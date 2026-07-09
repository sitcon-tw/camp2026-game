package territory

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

// Standings godoc
// @Summary Territory combat standings
// @Description Lists each team's hidden combat tier and allowed actions. Scores and formulas are never exposed. Includes the yansan boss team flagged isBoss.
// @Tags territory
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} StandingsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/standings [get]
func (h *Handler) Standings(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	standings, err := territory.ComputeStandings(r.Context(), h.db, h.content)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("standings unavailable", "territory_standings_failed", err))
		return
	}

	response := StandingsResponse{
		Teams:    make([]StandingEntryResponse, 0, len(standings)+1),
		MyTeamID: player.TeamID,
	}
	for _, standing := range standings {
		entry := StandingEntryResponse{
			TeamID:        standing.TeamID,
			Name:          standing.TeamName,
			Tier:          standing.Tier.String(),
			CanAttack:     territory.CanAttack(standing.Tier),
			CanBeAttacked: territory.CanBeAttacked(standing.Tier),
		}
		response.Teams = append(response.Teams, entry)
		if standing.TeamID == player.TeamID {
			response.MyTier = standing.Tier.String()
		}
	}

	bossState, err := h.service.BossState(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("standings unavailable", "territory_boss_state_failed", err))
		return
	}
	response.Teams = append(response.Teams, StandingEntryResponse{
		TeamID:        territory.TeamYansanID,
		Name:          territory.YansanTeamName,
		Tier:          "boss",
		CanAttack:     false,
		CanBeAttacked: bossState.Status == mongomodel.BossStatusOpen,
		IsBoss:        true,
		BossStatus:    bossState.Status,
	})

	httpx.WriteJSON(w, http.StatusOK, response)
}
