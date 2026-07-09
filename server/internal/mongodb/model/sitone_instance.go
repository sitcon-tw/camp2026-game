package model

import "time"

const SitoneInstancesCollection = "sitone_instances"

const (
	SitoneInstanceStatusNormal            = "normal"
	SitoneInstanceStatusDeployed          = "deployed"
	SitoneInstanceStatusFatigued          = "fatigued"
	SitoneInstanceStatusCaptured          = "captured"
	SitoneInstanceStatusCaptiveCooldown   = "captive_cooldown"
	SitoneInstanceStatusConvertPending    = "convert_pending"
	SitoneInstanceStatusMissing           = "missing"
	SitoneInstanceStatusRescued           = "rescued"
	SitoneInstanceStatusDead              = "dead"
	SitoneInstanceStatusProtectedAtYansan = "protected_at_yansan"
)

type SitoneInstance struct {
	ID             string    `bson:"_id"`
	PlayerID       string    `bson:"player_id"`
	TeamID         string    `bson:"team_id"`
	OriginPlayerID string    `bson:"origin_player_id"`
	OriginTeamID   string    `bson:"origin_team_id"`
	SitoneID       string    `bson:"sitone_id"`
	Status         string    `bson:"status"`
	MissionID      string    `bson:"mission_id,omitempty"`
	Fatigue        int       `bson:"fatigue"`
	CooldownUntil  time.Time `bson:"cooldown_until,omitempty"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}
