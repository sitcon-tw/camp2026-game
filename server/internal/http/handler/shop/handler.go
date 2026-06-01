package shop

import "github.com/go-chi/chi/v5"

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/shop/items", h.ListItems)
	api.Get("/shop/items/{itemID}", h.GetItem)
	api.Post("/shop/purchases", h.CreatePurchase)
}
