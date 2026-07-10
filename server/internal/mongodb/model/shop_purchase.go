package model

import "time"

const ShopPurchasesCollection = "shop_purchases"

type ShopPurchase struct {
	ID             string    `bson:"_id"`
	PlayerID       string    `bson:"player_id"`
	ItemID         string    `bson:"item_id"`
	Quantity       int       `bson:"quantity"`
	PriceOpenPower int       `bson:"price_open_power"`
	Repeatable     bool      `bson:"repeatable"`
	CreatedAt      time.Time `bson:"created_at"`
}
