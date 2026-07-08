package model

import "time"

const StaffRewardsCollection = "staff_rewards"
const StaffRewardTokensCollection = "staff_reward_tokens"
const StaffRewardTokenClaimsCollection = "staff_reward_token_claims"

type StaffReward struct {
	ID                  string    `bson:"_id"`
	StaffPlayerID       string    `bson:"staff_player_id"`
	RecipientPlayerID   string    `bson:"recipient_player_id"`
	Kind                string    `bson:"kind"`
	RefID               string    `bson:"ref_id"`
	Quantity            int       `bson:"quantity"`
	CreatedAt           time.Time `bson:"created_at"`
	NotificationPending bool      `bson:"notification_pending,omitempty"`
	NotifiedAt          time.Time `bson:"notified_at,omitempty"`
}

type StaffRewardToken struct {
	ID            string    `bson:"_id"`
	Token         string    `bson:"token"`
	StaffPlayerID string    `bson:"staff_player_id"`
	Kind          string    `bson:"kind"`
	RefID         string    `bson:"ref_id,omitempty"`
	Quantity      int       `bson:"quantity,omitempty"`
	Amount        int       `bson:"amount,omitempty"`
	CreatedAt     time.Time `bson:"created_at"`
	ExpiresAt     time.Time `bson:"expires_at"`
}

type StaffRewardTokenClaim struct {
	ID            string    `bson:"_id"`
	TokenID       string    `bson:"token_id"`
	PlayerID      string    `bson:"player_id"`
	StaffRewardID string    `bson:"staff_reward_id"`
	ClaimedAt     time.Time `bson:"claimed_at"`
}
