package model

import "time"

const PlayersCollection = "players"

type Player struct {
	ID               string    `bson:"_id"`
	AuthToken        string    `bson:"auth_token"`
	QRCodeToken      string    `bson:"qrcode_token"`
	Nickname         string    `bson:"nickname"`
	TeamID           string    `bson:"team_id,omitempty"`
	AvatarURL        string    `bson:"avatar_url,omitempty"`
	Role             string    `bson:"role,omitempty"`
	DefaultSitoneIDs []string  `bson:"default_sitone_ids,omitempty"`
	CreatedAt        time.Time `bson:"created_at,omitempty"`
	UpdatedAt        time.Time `bson:"updated_at,omitempty"`
}
