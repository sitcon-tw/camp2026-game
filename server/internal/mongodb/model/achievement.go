package model

import "time"

const AchievementsCollection = "achievements"

type Achievement struct {
	ID                  string    `bson:"_id"`
	PlayerID            string    `bson:"player_id"`
	Key                 string    `bson:"key"`
	Name                string    `bson:"name"`
	Tier                int       `bson:"tier,omitempty"`
	SortOrder           int       `bson:"sort_order"`
	RequiredSitoneCount int       `bson:"required_sitone_count"`
	SitoneCount         int       `bson:"sitone_count"`
	OpenPowerReward     int       `bson:"open_power_reward,omitempty"`
	CreatedAt           time.Time `bson:"created_at"`
	NotificationPending bool      `bson:"notification_pending,omitempty"`
	NotifiedAt          time.Time `bson:"notified_at,omitempty"`
}
