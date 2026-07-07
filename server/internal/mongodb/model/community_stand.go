package model

import "time"

const (
	CommunityStandsCollection      = "community_stands"
	CommunityStandVisitsCollection = "community_stand_visits"
	CommunityStandClaimsCollection = "community_stand_claims"
)

type CommunityStand struct {
	ID          string      `bson:"_id"`
	Name        string      `bson:"name"`
	Description string      `bson:"description"`
	LogoURL     string      `bson:"logo_url,omitempty"`
	WebsiteURL  string      `bson:"website_url,omitempty"`
	Enabled     bool        `bson:"enabled"`
	Reward      StandReward `bson:"reward"`
	CreatedAt   time.Time   `bson:"created_at"`
	UpdatedAt   time.Time   `bson:"updated_at"`
}

type StandReward struct {
	Kind     string `bson:"kind"`
	RefID    string `bson:"ref_id,omitempty"`
	Quantity int    `bson:"quantity,omitempty"`
	Amount   int    `bson:"amount,omitempty"`
}

type CommunityStandClaim struct {
	ID        string    `bson:"_id"`
	StandID   string    `bson:"stand_id"`
	PlayerID  string    `bson:"player_id"`
	RewardID  string    `bson:"reward_id"`
	CreatedAt time.Time `bson:"created_at"`
}

type CommunityStandVisit struct {
	ID             string    `bson:"_id"`
	StandID        string    `bson:"stand_id"`
	PlayerID       string    `bson:"player_id"`
	FirstVisitedAt time.Time `bson:"first_visited_at"`
	LastVisitedAt  time.Time `bson:"last_visited_at"`
}
