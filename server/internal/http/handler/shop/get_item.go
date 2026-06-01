package shop

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetItem godoc
// @Summary Get shop item
// @Description Returns one purchasable shop item.
// @Tags Shop
// @Produce json
// @Security AuthCookieAuth
// @Param itemID path string true "Item ID"
// @Success 200 {object} apimodel.ShopItemDetailResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /shop/items/{itemID} [get]
func (h *Handler) GetItem(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.ShopItemDetailResponse{
		Item: exampleShopItem(),
	})
}
