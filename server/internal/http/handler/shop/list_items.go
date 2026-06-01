package shop

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListItems godoc
// @Summary List shop items
// @Description Lists items that can be purchased with open power.
// @Tags Shop
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.ShopItemListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /shop/items [get]
func (h *Handler) ListItems(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.ShopItemListResponse{
		Items: []apimodel.ShopItemSummary{
			exampleShopItem(),
		},
	})
}

func exampleShopItem() apimodel.ShopItemSummary {
	return apimodel.ShopItemSummary{
		ItemID:         "item-upgrade-stone",
		Name:           "Upgrade Stone",
		ItemType:       "craft_material",
		PriceOpenPower: 300,
		Description:    "Used to craft advanced sitones.",
	}
}
