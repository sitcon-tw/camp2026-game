package model

const TeamsCollection = "teams"

type Team struct {
	ID        string `bson:"_id"`
	Name      string `bson:"name"`
	AvatarURL string `bson:"avatar_url,omitempty"`
}
