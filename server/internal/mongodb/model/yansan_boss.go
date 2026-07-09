package model

import "time"

const BossStatesCollection = "yansan_boss_states"

const BossStateID = "yansan_boss"

const (
	BossStatusClosed      = "closed"
	BossStatusOpen        = "open"
	BossStatusUnderAttack = "under_attack"
	BossStatusFinished    = "finished"
)

type BossState struct {
	ID                   string    `bson:"_id"`
	Status               string    `bson:"status"`
	OpenedBy             string    `bson:"opened_by,omitempty"`
	OpenedAt             time.Time `bson:"opened_at,omitempty"`
	AttackStartedAt      time.Time `bson:"attack_started_at,omitempty"`
	ClosedAt             time.Time `bson:"closed_at,omitempty"`
	ParticipatingTeamIDs []string  `bson:"participating_team_ids,omitempty"`
	UpdatedAt            time.Time `bson:"updated_at"`
}
