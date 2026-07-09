package admin

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

type BossStateResponse struct {
	Status          string   `json:"status"`
	RegisteredTeams []string `json:"registeredTeams"`
}

// OpenYansanBoss godoc
// @Summary Open the yansan boss (admin)
// @Description Transitions the yansan boss to open so teams can register for the boss fight.
// @Tags admin
// @Produce json
// @Success 200 {object} BossStateResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/yansan/boss/open [post]
func (h *Handler) OpenYansanBoss(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}
	state, err := h.territoryService().OpenBoss(r.Context(), "admin")
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("boss open failed", "admin_boss_open_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bossStateResponse(state))
}

// CloseYansanBoss godoc
// @Summary Close the yansan boss (admin)
// @Description Ends the boss fight; yansan services resume.
// @Tags admin
// @Produce json
// @Success 200 {object} BossStateResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/yansan/boss/close [post]
func (h *Handler) CloseYansanBoss(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}
	state, err := h.territoryService().CloseBoss(r.Context(), "admin")
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("boss close failed", "admin_boss_close_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bossStateResponse(state))
}

func (h *Handler) territoryService() *territory.Service {
	return territory.NewService(nil, h.db, h.content)
}

func bossStateResponse(state mongomodel.BossState) BossStateResponse {
	registered := state.ParticipatingTeamIDs
	if registered == nil {
		registered = []string{}
	}
	return BossStateResponse{
		Status:          state.Status,
		RegisteredTeams: registered,
	}
}
