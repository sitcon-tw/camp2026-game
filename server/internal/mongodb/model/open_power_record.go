package model

import "time"

const (
	OpenPowerRecordsCollection   = "open_power_records"
	OpenPowerLocksCollection     = "open_power_locks"
	OpenPowerTransfersCollection = "open_power_transfers"
)

type OpenPowerRecord struct {
	ID        string    `bson:"_id"`
	PlayerID  string    `bson:"player_id"`
	Amount    int       `bson:"amount"`
	Reason    string    `bson:"reason"`
	Source    string    `bson:"source"`
	CreatedAt time.Time `bson:"created_at"`
}

type OpenPowerTransfer struct {
	ID                  string    `bson:"_id"`
	SenderPlayerID      string    `bson:"sender_player_id"`
	RecipientPlayerID   string    `bson:"recipient_player_id"`
	TeamID              string    `bson:"team_id"`
	Amount              int       `bson:"amount"`
	SenderRecordID      string    `bson:"sender_record_id"`
	RecipientRecordID   string    `bson:"recipient_record_id"`
	CreatedAt           time.Time `bson:"created_at"`
	NotificationPending bool      `bson:"notification_pending,omitempty"`
	NotifiedAt          time.Time `bson:"notified_at,omitempty"`
}
