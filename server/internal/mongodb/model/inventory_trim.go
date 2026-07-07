package model

import "time"

const InventoryTrimsCollection = "inventory_trims"

type InventoryTrim struct {
	ID                  string    `bson:"_id"`
	PlayerID            string    `bson:"player_id"`
	SitoneCount         int       `bson:"sitone_count"`
	OpenPower           int       `bson:"open_power"`
	Message             string    `bson:"message"`
	CreatedAt           time.Time `bson:"created_at"`
	NotificationPending bool      `bson:"notification_pending,omitempty"`
	NotifiedAt          time.Time `bson:"notified_at,omitempty"`
}
