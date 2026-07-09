package yansan

import (
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

type BossStatusResponse struct {
	Status           string   `json:"status"`
	RegisteredTeams  []string `json:"registeredTeams"`
	RequiredTeams    int      `json:"requiredTeams"`
	MyTeamRegistered bool     `json:"myTeamRegistered"`
}

// BossStatus godoc
// @Summary Yansan boss status
// @Tags yansan
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} BossStatusResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /yansan/boss/status [get]
func (h *Handler) BossStatus(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}
	state, err := h.service.BossState(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("boss status unavailable", "yansan_boss_state_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.bossStatusResponse(r, state, player))
}

// BossAttack godoc
// @Summary Register my team for the boss attack
// @Description Registers the caller's team for the boss fight. When all player teams have registered, the boss transitions to under_attack.
// @Tags yansan
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} BossStatusResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /yansan/boss/attack [post]
func (h *Handler) BossAttack(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !requireTeam(w, r, player) {
		return
	}

	state, _, err := h.service.RegisterBossAttack(r.Context(), player)
	if err != nil {
		if errors.Is(err, territory.ErrBossNotOpen) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "boss is not open for attack"))
			return
		}
		if errors.Is(err, territory.ErrNotTerritoryParticipant) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "team does not participate in territory"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("boss attack failed", "yansan_boss_attack_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.bossStatusResponse(r, state, player))
}

func (h *Handler) bossStatusResponse(r *http.Request, state mongomodel.BossState, player mongomodel.Player) BossStatusResponse {
	required := 0
	count, err := h.db.Collection(mongomodel.TeamsCollection).CountDocuments(r.Context(), bson.M{
		"_id":       bson.M{"$nin": bson.A{territory.TeamYansanID, territory.TeamStaffID}},
		"is_system": bson.M{"$ne": true},
	})
	if err == nil {
		required = int(count)
	}

	registered := state.ParticipatingTeamIDs
	if registered == nil {
		registered = []string{}
	}
	myTeamRegistered := false
	for _, teamID := range registered {
		if teamID == player.TeamID {
			myTeamRegistered = true
		}
	}
	return BossStatusResponse{
		Status:           state.Status,
		RegisteredTeams:  registered,
		RequiredTeams:    required,
		MyTeamRegistered: myTeamRegistered,
	}
}
