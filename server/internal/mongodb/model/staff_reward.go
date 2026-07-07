package model

import "time"

const StaffRewardsCollection = "staff_rewards"

type StaffReward struct {
	ID                  string    `bson:"_id"`
	StaffPlayerID       string    `bson:"staff_player_id"`
	RecipientPlayerID   string    `bson:"recipient_player_id"`
	Kind                string    `bson:"kind"`
	RefID               string    `bson:"ref_id"`
	Quantity            int       `bson:"quantity"`
	CreatedAt           time.Time `bson:"created_at"`
	NotificationPending bool      `bson:"notification_pending,omitempty"`
	NotifiedAt          time.Time `bson:"notified_at,omitempty"`
}
