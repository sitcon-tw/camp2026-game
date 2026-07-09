package model

import "time"

const TeamDefensesCollection = "team_defenses"

// TeamDefense stores the standing defense configuration for a team.
// The document id is the team id.
type TeamDefense struct {
	ID                string    `bson:"_id"`
	SitoneIDs         []string  `bson:"sitone_ids"`
	UpdatedByPlayerID string    `bson:"updated_by_player_id"`
	UpdatedAt         time.Time `bson:"updated_at"`
	LockedUntil       time.Time `bson:"locked_until,omitempty"`
}
