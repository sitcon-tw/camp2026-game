package model

import "time"

const OpenPowerRecordsCollection = "open_power_records"

type OpenPowerRecord struct {
	ID        string    `bson:"_id"`
	PlayerID  string    `bson:"player_id"`
	Amount    int       `bson:"amount"`
	Reason    string    `bson:"reason"`
	Source    string    `bson:"source"`
	CreatedAt time.Time `bson:"created_at"`
}
