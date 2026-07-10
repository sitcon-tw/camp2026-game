package model

import "time"

const QRScanCooldownsCollection = "qr_scan_cooldowns"

type QRScanCooldown struct {
	ID        string    `bson:"_id"`
	PlayerID  string    `bson:"player_id"`
	OwnerID   string    `bson:"owner_id,omitempty"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}
