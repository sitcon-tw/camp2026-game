package territory

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/eventlog"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

// CreateAttack godoc
// @Summary Initiate a team attack
// @Description Starts an attack mission in voting state. The initiator auto-votes agree. Tier permissions are locked into the mission at creation.
// @Tags territory
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body CreateAttackRequest true "Attack request"
// @Success 201 {object} MissionResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/attacks [post]
func (h *Handler) CreateAttack(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body CreateAttackRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.DefenderTeamID = strings.TrimSpace(body.DefenderTeamID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if body.DefenderTeamID == player.TeamID {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("cannot attack your own team"))
		return
	}
	if body.DefenderTeamID == territory.TeamYansanID {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("use the boss attack endpoint to attack yansan"))
		return
	}

	ctx := r.Context()

	// Tier gating (§4): compute standings now and lock the tiers into the
	// mission (decision 6).
	standings, err := territory.ComputeStandings(ctx, h.db, h.content)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_standings_failed", err))
		return
	}
	var attackerTier, defenderTier territory.Tier
	attackerFound, defenderFound := false, false
	for _, standing := range standings {
		if standing.TeamID == player.TeamID {
			attackerTier, attackerFound = standing.Tier, true
		}
		if standing.TeamID == body.DefenderTeamID {
			defenderTier, defenderFound = standing.Tier, true
		}
	}
	if !attackerFound || !defenderFound {
		httpx.WriteProblem(w, r, httpx.NotFound("team not found in standings"))
		return
	}
	if !territory.CanAttack(attackerTier) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "your team cannot attack in its current tier"))
		return
	}
	if !territory.CanBeAttacked(defenderTier) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "the target team cannot be attacked in its current tier"))
		return
	}

	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_members_failed", err))
		return
	}
	teamSize := len(members)
	if len(body.SitoneIDs) > territory.DeployCap(teamSize) {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("too many sitones for your team size"))
		return
	}

	// Validate ownership with free quantity: quantity already excludes
	// instanced stones (hybrid model §2).
	quantities, err := h.service.PlayerQuantities(ctx, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_quantities_failed", err))
		return
	}
	needed := make(map[string]int, len(body.SitoneIDs))
	for _, sitoneID := range body.SitoneIDs {
		sitone, ok := h.content.GetSitone(sitoneID)
		if !ok {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity("unknown sitone "+sitoneID))
			return
		}
		if sitone.Attack <= 0 && sitone.Defense <= 0 {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity("sitone "+sitoneID+" cannot be deployed"))
			return
		}
		needed[sitoneID]++
	}
	for sitoneID, n := range needed {
		if quantities[sitoneID] < n {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient free quantity of sitone "+sitoneID))
			return
		}
	}

	// No other active mission for the team (§19 anti-abuse).
	active, err := h.service.ActiveMissionForTeam(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_active_mission_failed", err))
		return
	}
	if active != nil {
		refreshed, err := h.service.ExpireMissionIfDue(ctx, *active)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_expire_failed", err))
			return
		}
		if refreshed.Status == mongomodel.AttackMissionStatusVoting || refreshed.Status == mongomodel.AttackMissionStatusDeployed {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "your team already has an active mission"))
			return
		}
	}

	now := time.Now().UTC()
	mission := mongomodel.AttackMission{
		ID:                  territory.NewID("attack_mission"),
		AttackerTeamID:      player.TeamID,
		DefenderTeamID:      body.DefenderTeamID,
		InitiatorPlayerID:   player.ID,
		BeneficiaryPlayerID: player.ID,
		Status:              mongomodel.AttackMissionStatusVoting,
		SelectedSitoneIDs:   body.SitoneIDs,
		VoteDeadline:        now.Add(territory.VoteDeadlineDuration),
		AttackerTier:        int(attackerTier),
		DefenderTier:        int(defenderTier),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if _, err := h.db.Collection(mongomodel.AttackMissionsCollection).InsertOne(ctx, mission); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_mission_insert_failed", err))
		return
	}

	// Initiator auto-votes agree (decision 9).
	_, err = h.db.Collection(mongomodel.AttackVotesCollection).InsertOne(ctx, mongomodel.AttackVote{
		ID:        territory.NewID("attack_vote"),
		MissionID: mission.ID,
		PlayerID:  player.ID,
		Vote:      true,
		CreatedAt: now,
	})
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_vote_insert_failed", err))
		return
	}

	_, _ = eventlog.Insert(ctx, h.db, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeAttackInitiated,
		ActorPlayerID:    player.ID,
		TeamID:           player.TeamID,
		RelatedMissionID: mission.ID,
		Payload: bson.M{
			"defender_team_id": body.DefenderTeamID,
			"sitone_ids":       body.SitoneIDs,
		},
	})
	_, _ = eventlog.Insert(ctx, h.db, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeAttackVoteCast,
		ActorPlayerID:    player.ID,
		TeamID:           player.TeamID,
		RelatedMissionID: mission.ID,
		Payload:          bson.M{"agree": true, "auto": true},
	})

	response, err := h.missionResponse(ctx, mission, true)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("attack failed", "territory_mission_view_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response)
}

// VoteAttack godoc
// @Summary Vote on an attack mission
// @Description Casts a teammate vote. Reaching the threshold within the deadline deploys the mission: the initiator pays the open power cost and the selected sitones are locked as deployed instances.
// @Tags territory
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param missionID path string true "Mission id"
// @Param request body VoteRequest true "Vote"
// @Success 200 {object} MissionResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/attacks/{missionID}/votes [post]
func (h *Handler) VoteAttack(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body VoteRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	ctx := r.Context()
	mission, ok := h.findMission(w, r, chi.URLParam(r, "missionID"))
	if !ok {
		return
	}
	if mission.AttackerTeamID != player.TeamID {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "only teammates can vote"))
		return
	}

	// Lazy expiry (decision 9).
	mission, err := h.service.ExpireMissionIfDue(ctx, mission)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_expire_failed", err))
		return
	}
	if mission.Status != mongomodel.AttackMissionStatusVoting {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "mission is not open for voting"))
		return
	}

	// One vote per player (unique index).
	_, err = h.db.Collection(mongomodel.AttackVotesCollection).InsertOne(ctx, mongomodel.AttackVote{
		ID:        territory.NewID("attack_vote"),
		MissionID: mission.ID,
		PlayerID:  player.ID,
		Vote:      *body.Agree,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "you have already voted"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_vote_insert_failed", err))
		return
	}
	_, _ = eventlog.Insert(ctx, h.db, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeAttackVoteCast,
		ActorPlayerID:    player.ID,
		TeamID:           player.TeamID,
		RelatedMissionID: mission.ID,
		Payload:          bson.M{"agree": *body.Agree},
	})

	// Threshold check → commit under team lock (decision 9: deploy at the
	// instant the threshold is reached).
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_members_failed", err))
		return
	}
	teamSize := len(members)
	agreeCount, err := h.countAgreeVotes(ctx, mission.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_vote_count_failed", err))
		return
	}

	if agreeCount >= territory.VoteThreshold(teamSize) {
		rank, err := h.service.PersonalRank(ctx, mission.InitiatorPlayerID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_rank_failed", err))
			return
		}
		var committed mongomodel.AttackMission
		err = h.service.WithLock(ctx, "attack:"+player.TeamID, func(ctx context.Context) error {
			var commitErr error
			committed, commitErr = h.service.CommitMission(ctx, mission, rank, teamSize)
			return commitErr
		})
		if err != nil {
			if errors.Is(err, territory.ErrInsufficientOpenPower) {
				h.cancelMissionInternal(r, mission, "insufficient_open_power")
				httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "initiator has insufficient open power"))
				return
			}
			if errors.Is(err, territory.ErrInsufficientSitones) {
				h.cancelMissionInternal(r, mission, "insufficient_sitones")
				httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "initiator no longer holds the selected sitones"))
				return
			}
			httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_commit_failed", err))
			return
		}
		mission = committed
	}

	response, err := h.missionResponse(ctx, mission, true)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("vote failed", "territory_mission_view_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// CancelAttack godoc
// @Summary Cancel an attack mission
// @Description Initiator or a teammate cancels the mission while it is still voting (before the threshold commits it).
// @Tags territory
// @Produce json
// @Security AuthCookieAuth
// @Param missionID path string true "Mission id"
// @Success 200 {object} MissionResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/attacks/{missionID}/cancel [post]
func (h *Handler) CancelAttack(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	ctx := r.Context()
	mission, ok := h.findMission(w, r, chi.URLParam(r, "missionID"))
	if !ok {
		return
	}
	if mission.AttackerTeamID != player.TeamID {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "only teammates can cancel"))
		return
	}
	mission, err := h.service.ExpireMissionIfDue(ctx, mission)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("cancel failed", "territory_expire_failed", err))
		return
	}
	if mission.Status != mongomodel.AttackMissionStatusVoting {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "mission can no longer be cancelled"))
		return
	}

	now := time.Now().UTC()
	result, err := h.db.Collection(mongomodel.AttackMissionsCollection).UpdateOne(
		ctx,
		bson.M{"_id": mission.ID, "status": mongomodel.AttackMissionStatusVoting},
		bson.M{"$set": bson.M{
			"status":     mongomodel.AttackMissionStatusCancelled,
			"updated_at": now,
		}},
	)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("cancel failed", "territory_cancel_failed", err))
		return
	}
	if result.ModifiedCount == 0 {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "mission can no longer be cancelled"))
		return
	}
	mission.Status = mongomodel.AttackMissionStatusCancelled

	_, _ = eventlog.Insert(ctx, h.db, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeAttackCancelled,
		ActorPlayerID:    player.ID,
		TeamID:           player.TeamID,
		RelatedMissionID: mission.ID,
	})

	response, err := h.missionResponse(ctx, mission, true)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("cancel failed", "territory_mission_view_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// GetAttack godoc
// @Summary Get an attack mission
// @Description Returns the mission view for the attacker team. The defender team can see resolved missions. Result summaries only contain public-safe labels.
// @Tags territory
// @Produce json
// @Security AuthCookieAuth
// @Param missionID path string true "Mission id"
// @Success 200 {object} MissionResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/attacks/{missionID} [get]
func (h *Handler) GetAttack(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	ctx := r.Context()
	mission, ok := h.findMission(w, r, chi.URLParam(r, "missionID"))
	if !ok {
		return
	}
	mission, err := h.service.ExpireMissionIfDue(ctx, mission)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("mission unavailable", "territory_expire_failed", err))
		return
	}

	isAttacker := mission.AttackerTeamID == player.TeamID
	isDefender := mission.DefenderTeamID == player.TeamID
	if !isAttacker && !(isDefender && mission.Status == mongomodel.AttackMissionStatusResolved) {
		httpx.WriteProblem(w, r, httpx.NotFound("mission not found"))
		return
	}

	response, err := h.missionResponse(ctx, mission, isAttacker)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("mission unavailable", "territory_mission_view_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) findMission(w http.ResponseWriter, r *http.Request, missionID string) (mongomodel.AttackMission, bool) {
	missionID = strings.TrimSpace(missionID)
	var mission mongomodel.AttackMission
	err := h.db.Collection(mongomodel.AttackMissionsCollection).FindOne(r.Context(), bson.M{"_id": missionID}).Decode(&mission)
	if err == mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.NotFound("mission not found"))
		return mongomodel.AttackMission{}, false
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("mission unavailable", "territory_mission_lookup_failed", err))
		return mongomodel.AttackMission{}, false
	}
	return mission, true
}

func (h *Handler) cancelMissionInternal(r *http.Request, mission mongomodel.AttackMission, reason string) {
	now := time.Now().UTC()
	_, _ = h.db.Collection(mongomodel.AttackMissionsCollection).UpdateOne(
		r.Context(),
		bson.M{"_id": mission.ID, "status": mongomodel.AttackMissionStatusVoting},
		bson.M{"$set": bson.M{
			"status":     mongomodel.AttackMissionStatusCancelled,
			"updated_at": now,
		}},
	)
	_, _ = eventlog.Insert(r.Context(), h.db, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeAttackCancelled,
		TeamID:           mission.AttackerTeamID,
		RelatedMissionID: mission.ID,
		Payload:          bson.M{"reason": reason},
	})
}
