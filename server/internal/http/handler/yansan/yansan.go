package yansan

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

type StatusResponse struct {
	BossStatus        string                   `json:"bossStatus"`
	ServicesAvailable bool                     `json:"servicesAvailable"`
	ProtectedStones   []ProtectedStoneResponse `json:"protectedStones"`
	TeamRedeemCount   int                      `json:"teamRedeemCount"`
}

type ProtectedStoneResponse struct {
	InstanceID string `json:"instanceId"`
	SitoneID   string `json:"sitoneId"`
	SitoneName string `json:"sitoneName,omitempty"`
}

type RedeemRequest struct {
	SitoneID string `json:"sitoneId" validate:"required"`
}

type ConvertRequest struct {
	CaptiveID      string `json:"captiveId" validate:"required"`
	ExtraOpenPower int    `json:"extraOpenPower" validate:"min=0"`
}

// ProcessResponse mirrors the territory process view: outcome labels only.
type ProcessResponse struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	SitoneID  string     `json:"sitoneId"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"startedAt"`
	ResolveAt *time.Time `json:"resolveAt,omitempty"`
}

type BossStatusResponse struct {
	Status           string   `json:"status"`
	RegisteredTeams  []string `json:"registeredTeams"`
	RequiredTeams    int      `json:"requiredTeams"`
	MyTeamRegistered bool     `json:"myTeamRegistered"`
}

// Status godoc
// @Summary Yansan status
// @Description Returns the boss state, whether yansan services are available, my team's protected unique stones and the team redeem count.
// @Tags yansan
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} StatusResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /yansan/status [get]
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !requireTeam(w, r, player) {
		return
	}

	ctx := r.Context()
	bossState, err := h.service.BossState(ctx)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("yansan status unavailable", "yansan_boss_state_failed", err))
		return
	}

	cursor, err := h.db.Collection(mongomodel.SitoneInstancesCollection).Find(ctx, bson.M{
		"origin_team_id": player.TeamID,
		"status":         mongomodel.SitoneInstanceStatusProtectedAtYansan,
	})
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("yansan status unavailable", "yansan_protected_lookup_failed", err))
		return
	}
	var instances []mongomodel.SitoneInstance
	if err := cursor.All(ctx, &instances); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("yansan status unavailable", "yansan_protected_lookup_failed", err))
		return
	}

	redeemCount, err := h.service.TeamRedeemCount(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("yansan status unavailable", "yansan_redeem_count_failed", err))
		return
	}

	response := StatusResponse{
		BossStatus:        bossState.Status,
		ServicesAvailable: bossState.Status != mongomodel.BossStatusUnderAttack,
		ProtectedStones:   make([]ProtectedStoneResponse, 0, len(instances)),
		TeamRedeemCount:   redeemCount,
	}
	for _, instance := range instances {
		entry := ProtectedStoneResponse{
			InstanceID: instance.ID,
			SitoneID:   instance.SitoneID,
		}
		if h.content != nil {
			if sitone, ok := h.content.GetSitone(instance.SitoneID); ok {
				entry.SitoneName = sitone.Name
			}
		}
		response.ProtectedStones = append(response.ProtectedStones, entry)
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// Redeem godoc
// @Summary Redeem a unique stone from yansan
// @Description Starts a redeem process for a unique stone of my team held at yansan. Cost and success rate scale with the team's redeem count. Blocked while the boss is under attack.
// @Tags yansan
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body RedeemRequest true "Redeem request"
// @Success 201 {object} ProcessResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /yansan/redeem [post]
func (h *Handler) Redeem(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body RedeemRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.SitoneID = strings.TrimSpace(body.SitoneID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	ctx := r.Context()
	var instance mongomodel.SitoneInstance
	err := h.db.Collection(mongomodel.SitoneInstancesCollection).FindOne(ctx, bson.M{
		"origin_team_id": player.TeamID,
		"sitone_id":      body.SitoneID,
		"status":         mongomodel.SitoneInstanceStatusProtectedAtYansan,
	}).Decode(&instance)
	if err == mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.NotFound("no protected stone of that kind for your team"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("redeem failed", "yansan_instance_lookup_failed", err))
		return
	}

	// Cost keyed on the original holder's rank (design §9).
	rank, err := h.service.PersonalRank(ctx, instance.OriginPlayerID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("redeem failed", "yansan_rank_failed", err))
		return
	}
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("redeem failed", "yansan_members_failed", err))
		return
	}

	process, err := h.service.StartRedeem(ctx, player, instance, rank, len(members))
	if err != nil {
		writeYansanError(w, r, err, "redeem")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProcessResponse(process))
}

// Convert godoc
// @Summary Convert a captive (yansan alias)
// @Description Alias of POST /territory/captives/{captiveID}/convert with the captive id in the body.
// @Tags yansan
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body ConvertRequest true "Convert request"
// @Success 201 {object} ProcessResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /yansan/convert [post]
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body ConvertRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.CaptiveID = strings.TrimSpace(body.CaptiveID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	ctx := r.Context()
	var captive mongomodel.CaptiveRecord
	err := h.db.Collection(mongomodel.CaptiveRecordsCollection).FindOne(ctx, bson.M{"_id": body.CaptiveID}).Decode(&captive)
	if err == mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.NotFound("captive not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("convert failed", "yansan_captive_lookup_failed", err))
		return
	}

	rank, err := h.service.PersonalRank(ctx, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("convert failed", "yansan_rank_failed", err))
		return
	}
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("convert failed", "yansan_members_failed", err))
		return
	}

	process, err := h.service.StartConvert(ctx, player, captive, rank, len(members), body.ExtraOpenPower)
	if err != nil {
		writeYansanError(w, r, err, "convert")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProcessResponse(process))
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
// @Description Registers the caller's team for the boss fight. When all player teams have registered, the boss transitions to under_attack: every pending yansan service immediately fails without refund.
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
		httpx.WriteProblem(w, r, httpx.InternalServerError("boss attack failed", "yansan_boss_attack_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.bossStatusResponse(r, state, player))
}

func (h *Handler) bossStatusResponse(r *http.Request, state mongomodel.BossState, player mongomodel.Player) BossStatusResponse {
	required := 0
	count, err := h.db.Collection(mongomodel.TeamsCollection).CountDocuments(r.Context(), bson.M{
		"_id":       bson.M{"$ne": territory.TeamYansanID},
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

func toProcessResponse(process mongomodel.YansanProcess) ProcessResponse {
	response := ProcessResponse{
		ID:        process.ID,
		Kind:      process.Kind,
		SitoneID:  process.SitoneID,
		Status:    process.Status,
		StartedAt: process.StartedAt,
	}
	if !process.ResolveAt.IsZero() {
		resolveAt := process.ResolveAt
		response.ResolveAt = &resolveAt
	}
	return response
}

func writeYansanError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, territory.ErrBossUnderAttack):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "yansan services are suspended while the boss is under attack"))
	case errors.Is(err, territory.ErrCaptiveNotConvertible):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "captive is still in cooldown or already processed"))
	case errors.Is(err, territory.ErrProcessAlreadyPending):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "a pending process already exists for this target"))
	case errors.Is(err, territory.ErrInsufficientOpenPower):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient open power"))
	case errors.Is(err, territory.ErrNotYourTeam):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "target does not belong to your team"))
	case errors.Is(err, territory.ErrWrongState):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "target is not in the required state"))
	default:
		httpx.WriteProblem(w, r, httpx.InternalServerError(action+" failed", "yansan_"+action+"_failed", err))
	}
}
