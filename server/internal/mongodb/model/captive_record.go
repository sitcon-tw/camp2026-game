package model

import "time"

const CaptiveRecordsCollection = "captive_records"

const (
	CaptiveRecordStatusCooldown    = "cooldown"
	CaptiveRecordStatusConvertible = "convertible"
	CaptiveRecordStatusConverted   = "converted"
	CaptiveRecordStatusRecaptured  = "recaptured"
)

type CaptiveRecord struct {
	ID             string    `bson:"_id"`
	SitoneID       string    `bson:"sitone_id"`
	OriginalTeamID string    `bson:"original_team_id"`
	CurrentTeamID  string    `bson:"current_team_id"`
	InstanceID     string    `bson:"instance_id"`
	CapturedAt     time.Time `bson:"captured_at"`
	CooldownUntil  time.Time `bson:"cooldown_until"`
	ConvertedAt    time.Time `bson:"converted_at,omitempty"`
	Status         string    `bson:"status"`
}
