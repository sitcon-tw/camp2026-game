package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const EventLogsCollection = "event_logs"

const (
	EventTypeAttackInitiated       = "attack_initiated"
	EventTypeAttackVoteCast        = "attack_vote_cast"
	EventTypeAttackVoteExpired     = "attack_vote_expired"
	EventTypeAttackCancelled       = "attack_cancelled"
	EventTypeOpenPowerSpent        = "open_power_spent"
	EventTypeSitonesDeployed       = "sitones_deployed"
	EventTypeMissionResolved       = "mission_resolved"
	EventTypeStealSuccess          = "steal_success"
	EventTypeSitoneCaptured        = "sitone_captured"
	EventTypeSitoneMissing         = "sitone_missing"
	EventTypeRescueStarted         = "rescue_started"
	EventTypeRescueResolved        = "rescue_resolved"
	EventTypeSitoneDied            = "sitone_died"
	EventTypeSitoneSentYansan      = "sitone_sent_yansan"
	EventTypeRedeemStarted         = "redeem_started"
	EventTypeRedeemSucceeded       = "redeem_succeeded"
	EventTypeRedeemFailed          = "redeem_failed"
	EventTypeConvertStarted        = "convert_started"
	EventTypeConvertSucceeded      = "convert_succeeded"
	EventTypeConvertFailed         = "convert_failed"
	EventTypeBossOpened            = "boss_opened"
	EventTypeBossClosed            = "boss_closed"
	EventTypeBossAttacked          = "boss_attacked"
	EventTypeBossInterruptedYansan = "boss_interrupted_yansan"
	EventTypeStaffAdjustment       = "staff_adjustment"
	EventTypeDefenseUpdated        = "defense_updated"
	EventTypeCaptiveRecaptured     = "captive_recaptured"
)

type EventLog struct {
	ID               string    `bson:"_id"`
	EventType        string    `bson:"event_type"`
	ActorPlayerID    string    `bson:"actor_player_id,omitempty"`
	TargetPlayerID   string    `bson:"target_player_id,omitempty"`
	TeamID           string    `bson:"team_id,omitempty"`
	RelatedSitoneID  string    `bson:"related_sitone_id,omitempty"`
	RelatedMissionID string    `bson:"related_mission_id,omitempty"`
	Payload          bson.M    `bson:"payload,omitempty"`
	CreatedAt        time.Time `bson:"created_at"`
}
