package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const AttackMissionsCollection = "attack_missions"

const (
	AttackMissionStatusVoting    = "voting"
	AttackMissionStatusCancelled = "cancelled"
	AttackMissionStatusExpired   = "expired"
	AttackMissionStatusDeployed  = "deployed"
	AttackMissionStatusResolved  = "resolved"
)

type AttackMission struct {
	ID                  string    `bson:"_id"`
	AttackerTeamID      string    `bson:"attacker_team_id"`
	DefenderTeamID      string    `bson:"defender_team_id"`
	InitiatorPlayerID   string    `bson:"initiator_player_id"`
	BeneficiaryPlayerID string    `bson:"beneficiary_player_id"`
	Status              string    `bson:"status"`
	SelectedSitoneIDs   []string  `bson:"selected_sitone_ids"`
	CostOpenPower       int       `bson:"cost_open_power"`
	VoteDeadline        time.Time `bson:"vote_deadline"`
	StartedAt           time.Time `bson:"started_at,omitempty"`
	ResolveAt           time.Time `bson:"resolve_at,omitempty"`
	ResolvedAt          time.Time `bson:"resolved_at,omitempty"`
	AttackerTier        int       `bson:"attacker_tier,omitempty"`
	DefenderTier        int       `bson:"defender_tier,omitempty"`
	RandomSeedRef       string    `bson:"random_seed_ref,omitempty"`
	ResultSummary       bson.M    `bson:"result_summary,omitempty"`
	CreatedAt           time.Time `bson:"created_at"`
	UpdatedAt           time.Time `bson:"updated_at"`
}
