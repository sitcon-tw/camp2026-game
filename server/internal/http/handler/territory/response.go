package territory

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

// StandingsResponse exposes only tiers and allowed actions — never scores
// (design §4).
type StandingsResponse struct {
	Teams           []StandingEntryResponse `json:"teams"`
	MyTeamID        string                  `json:"myTeamId"`
	MyTier          string                  `json:"myTier"`
	ActiveMissionID string                  `json:"activeMissionId,omitempty"`
}

type StandingEntryResponse struct {
	TeamID        string `json:"teamId"`
	Name          string `json:"name"`
	Tier          string `json:"tier"`
	CanAttack     bool   `json:"canAttack"`
	CanBeAttacked bool   `json:"canBeAttacked"`
	IsBoss        bool   `json:"isBoss"`
	BossStatus    string `json:"bossStatus,omitempty"`
}

type CreateAttackRequest struct {
	DefenderTeamID string   `json:"defenderTeamId" validate:"required"`
	SitoneIDs      []string `json:"sitoneIds" validate:"required,min=1"`
	ItemIDs        []string `json:"itemIds"`
}

type VoteRequest struct {
	Agree *bool `json:"agree" validate:"required"`
}

// CostEstimateResponse is the only cost information players ever see.
type CostEstimateResponse struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// MissionResponse is the player-facing mission view: labels only, no
// rolls, no formulas (design §14).
type MissionResponse struct {
	ID             string                `json:"id"`
	Status         string                `json:"status"`
	AttackerTeamID string                `json:"attackerTeamId"`
	DefenderTeamID string                `json:"defenderTeamId"`
	InitiatorID    string                `json:"initiatorId"`
	VoteDeadline   time.Time             `json:"voteDeadline"`
	EstimatedCost  *CostEstimateResponse `json:"estimatedCost,omitempty"`
	RiskLevel      string                `json:"riskLevel,omitempty"`
	MissingRisk    string                `json:"missingRisk,omitempty"`
	DeathRisk      string                `json:"deathRisk,omitempty"`
	AgreeCount     int                   `json:"agreeCount"`
	Threshold      int                   `json:"threshold"`
	SitoneIDs      []string              `json:"sitoneIds,omitempty"`
	ItemIDs        []string              `json:"itemIds,omitempty"`
	ResolveAt      *time.Time            `json:"resolveAt,omitempty"`
	ResultSummary  map[string]any        `json:"resultSummary,omitempty"`
}

type DefenseResponse struct {
	SitoneIDs   []string   `json:"sitoneIds"`
	Cap         int        `json:"cap"`
	LockedUntil *time.Time `json:"lockedUntil,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

type UpdateDefenseRequest struct {
	SitoneIDs []string `json:"sitoneIds" validate:"required"`
}

type CaptiveResponse struct {
	ID             string     `json:"id"`
	SitoneID       string     `json:"sitoneId"`
	SitoneName     string     `json:"sitoneName,omitempty"`
	Status         string     `json:"status"`
	OriginalTeamID string     `json:"originalTeamId"`
	CurrentTeamID  string     `json:"currentTeamId"`
	CapturedAt     time.Time  `json:"capturedAt"`
	CooldownUntil  time.Time  `json:"cooldownUntil"`
	ConvertedAt    *time.Time `json:"convertedAt,omitempty"`
	HeldByMyTeam   bool       `json:"heldByMyTeam"`
}

type ListCaptivesResponse struct {
	Captives []CaptiveResponse `json:"captives"`
}

type ConvertCaptiveRequest struct {
	ExtraOpenPower int `json:"extraOpenPower" validate:"min=0"`
}

// ProcessResponse is the player-facing view of a yansan process (convert /
// redeem / rescue): outcome labels only.
type ProcessResponse struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	SitoneID  string     `json:"sitoneId"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"startedAt"`
	ResolveAt *time.Time `json:"resolveAt,omitempty"`
	Outcome   string     `json:"outcome,omitempty"`
}

type CreateRescueRequest struct {
	SitoneID string `json:"sitoneId" validate:"required"`
}

func processResponse(process mongomodel.YansanProcess) ProcessResponse {
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
	if process.Status != mongomodel.YansanProcessStatusPending {
		response.Outcome = processOutcomeLabel(process)
	}
	return response
}

func processOutcomeLabel(process mongomodel.YansanProcess) string {
	switch process.Status {
	case mongomodel.YansanProcessStatusSucceeded:
		return "success"
	case mongomodel.YansanProcessStatusAbortedBoss:
		return "aborted_boss"
	case mongomodel.YansanProcessStatusFailed:
		if process.Kind == mongomodel.YansanProcessKindRescue {
			if outcome, ok := process.Result["outcome"].(string); ok {
				return outcome
			}
		}
		return "fail"
	default:
		return ""
	}
}

// missionResponse builds the safe mission view for a player of the
// attacker team.
func (h *Handler) missionResponse(ctx context.Context, mission mongomodel.AttackMission, includeSitones bool) (MissionResponse, error) {
	agreeCount, err := h.countAgreeVotes(ctx, mission.ID)
	if err != nil {
		return MissionResponse{}, err
	}
	members, err := h.service.TeamMembers(ctx, mission.AttackerTeamID)
	if err != nil {
		return MissionResponse{}, err
	}
	teamSize := len(members)

	response := MissionResponse{
		ID:             mission.ID,
		Status:         mission.Status,
		AttackerTeamID: mission.AttackerTeamID,
		DefenderTeamID: mission.DefenderTeamID,
		InitiatorID:    mission.InitiatorPlayerID,
		VoteDeadline:   mission.VoteDeadline,
		AgreeCount:     agreeCount,
		Threshold:      territory.VoteThreshold(teamSize),
	}
	if includeSitones {
		response.SitoneIDs = mission.SelectedSitoneIDs
		response.ItemIDs = mission.SelectedItemIDs
	}
	if !mission.ResolveAt.IsZero() {
		resolveAt := mission.ResolveAt
		response.ResolveAt = &resolveAt
	}
	if mission.Status == mongomodel.AttackMissionStatusResolved && mission.ResultSummary != nil {
		response.ResultSummary = map[string]any(mission.ResultSummary)
	}

	if mission.Status == mongomodel.AttackMissionStatusVoting {
		rank, err := h.service.PersonalRank(ctx, mission.InitiatorPlayerID)
		if err != nil {
			return MissionResponse{}, err
		}
		minCost, maxCost, err := h.service.Costs.EstimateCost(
			territory.CostKindAttack,
			rank,
			teamSize,
			territory.AttackCostExtras(territory.Tier(mission.AttackerTier), territory.Tier(mission.DefenderTier)),
		)
		if err == nil {
			response.EstimatedCost = &CostEstimateResponse{Min: minCost, Max: maxCost}
		}
		stones := h.deployedStonesFromSelection(mission.SelectedSitoneIDs)
		response.RiskLevel = territory.AttackRiskLevel(territory.ComputeEffectiveAttack(h.service.Engine, stones, teamSize))
		response.MissingRisk = territory.MissingRiskLevel(h.service.Engine, stones)
		response.DeathRisk = territory.DeathRiskLevel(h.service.Engine, stones)
	}
	return response, nil
}

func (h *Handler) deployedStonesFromSelection(sitoneIDs []string) []territory.DeployedStone {
	stones := make([]territory.DeployedStone, 0, len(sitoneIDs))
	for _, sitoneID := range sitoneIDs {
		if sitone, ok := h.content.GetSitone(sitoneID); ok {
			stones = append(stones, territory.DeployedStone{SitoneID: sitoneID, Sitone: sitone})
		}
	}
	return stones
}

func (h *Handler) countAgreeVotes(ctx context.Context, missionID string) (int, error) {
	count, err := h.db.Collection(mongomodel.AttackVotesCollection).CountDocuments(ctx, bson.M{
		"mission_id": missionID,
		"vote":       true,
	})
	return int(count), err
}
