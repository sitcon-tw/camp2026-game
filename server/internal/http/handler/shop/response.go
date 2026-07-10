package shop

type ItemListResponse struct {
	Items []ShopItemResponse `json:"items"`
}

type ItemDetailResponse struct {
	Item ShopItemResponse `json:"item"`
}

type ShopItemResponse struct {
	ID             string `json:"id" example:"item_adventure_backpack"`
	Name           string `json:"name" example:"冒險背包"`
	Type           string `json:"type" example:"material"`
	Rarity         string `json:"rarity" example:"common"`
	Description    string `json:"description" example:"冒險背包，可用於小石合成。"`
	IconPath       string `json:"iconPath,omitempty" example:"/game-icons/items/item_adventure_backpack.png"`
	Source         string `json:"source,omitempty" example:"shop"`
	PriceOpenPower int    `json:"priceOpenPower" example:"50"`
	Locked         bool   `json:"locked" example:"false"`
	Redeemed       bool   `json:"redeemed" example:"false"`
	Repeatable     bool   `json:"repeatable" example:"false"`
}

type PurchaseRequest struct {
	ItemID string `json:"itemId" validate:"required" example:"item_adventure_backpack"`
}

type PurchaseResponse struct {
	PurchaseID     string           `json:"purchaseId" example:"purchase_abc123"`
	ItemID         string           `json:"itemId" example:"item_adventure_backpack"`
	Quantity       int              `json:"quantity" example:"1"`
	PriceOpenPower int              `json:"priceOpenPower" example:"50"`
	OpenPower      int              `json:"openPower" example:"1230"`
	Item           ShopItemResponse `json:"item"`
}
