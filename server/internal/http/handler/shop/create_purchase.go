package shop

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// CreatePurchase godoc
// @Summary Purchase shop item
// @Description Purchases an item with open power and adds it to the current player's inventory.
// @Tags Shop
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body apimodel.ShopPurchaseRequest true "Shop purchase request"
// @Success 201 {object} apimodel.ShopPurchaseResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 501 {object} httpx.ProblemDetails
// @Router /shop/purchases [post]
func (h *Handler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	var body apimodel.ShopPurchaseRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	httpx.WriteProblem(w, r, httpx.NotImplemented("shop purchases are not implemented yet"))
}
