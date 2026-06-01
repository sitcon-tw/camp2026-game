package model

const PlayersCollection = "players"

type Player struct {
	ID          string `bson:"_id"`
	AuthToken   string `bson:"auth_token"`
	QRCodeToken string `bson:"qrcode_token"`
	Nickname    string `bson:"nickname"`
	TeamID      string `bson:"team_id"`
	AvatarURL   string `bson:"avatar_url,omitempty"`
}
