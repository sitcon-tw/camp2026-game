package apimodel

type ShopItemListResponse struct {
	Items []ShopItemSummary `json:"items"`
}

type ShopItemDetailResponse struct {
	Item ShopItemSummary `json:"item"`
}

type ShopItemSummary struct {
	ItemID         string `json:"itemId" example:"item-upgrade-stone"`
	Name           string `json:"name" example:"Upgrade Stone"`
	ItemType       string `json:"itemType" example:"craft_material"`
	PriceOpenPower int    `json:"priceOpenPower" example:"300"`
	Description    string `json:"description,omitempty" example:"Used to craft advanced sitones."`
}

type ShopPurchaseRequest struct {
	ItemID   string `json:"itemId" validate:"required" example:"item-upgrade-stone"`
	Quantity int    `json:"quantity" validate:"required,min=1,max=99" example:"1"`
}

type ShopPurchaseResponse struct {
	PurchaseID     string      `json:"purchaseId" example:"P9K2QA"`
	Item           ItemSummary `json:"item"`
	OpenPowerSpent int         `json:"openPowerSpent" example:"300"`
	OpenPower      int         `json:"openPower" example:"980"`
}
