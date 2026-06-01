package model

const TeamsCollection = "teams"

type Team struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
}
