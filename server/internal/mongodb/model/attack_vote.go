package model

import "time"

const AttackVotesCollection = "attack_votes"

type AttackVote struct {
	ID        string    `bson:"_id"`
	MissionID string    `bson:"mission_id"`
	PlayerID  string    `bson:"player_id"`
	Vote      bool      `bson:"vote"`
	CreatedAt time.Time `bson:"created_at"`
}
