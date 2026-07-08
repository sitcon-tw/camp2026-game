package model

import "time"

const MatchPairingsCollection = "match_pairings"

type MatchPairing struct {
	ID                 string    `bson:"_id"`
	Token              string    `bson:"token"`
	MatchID            string    `bson:"match_id"`
	HostPlayerID       string    `bson:"host_player_id"`
	ConsumedByPlayerID string    `bson:"consumed_by_player_id,omitempty"`
	CreatedAt          time.Time `bson:"created_at"`
	ExpiresAt          time.Time `bson:"expires_at"`
	ConsumedAt         time.Time `bson:"consumed_at,omitempty"`
}
