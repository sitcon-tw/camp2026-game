package model

import "time"

const (
	RoomTeamsCollection           = "room_teams"
	RoomTeamMembershipsCollection = "room_team_memberships"
)

type RoomTeam struct {
	ID               string    `bson:"_id"`
	RoomNumber       string    `bson:"room_number"`
	QRToken          string    `bson:"qr_token,omitempty"`
	QRTokenExpiresAt time.Time `bson:"qr_token_expires_at,omitempty"`
	CreatedAt        time.Time `bson:"created_at"`
	UpdatedAt        time.Time `bson:"updated_at"`
}

type RoomTeamMembership struct {
	ID         string    `bson:"_id"`
	RoomTeamID string    `bson:"room_team_id"`
	PlayerID   string    `bson:"player_id"`
	JoinedBy   string    `bson:"joined_by,omitempty"`
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
}
