package home

import "github.com/go-chi/chi/v5"

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/me", h.GetMe)
	api.Get("/me/state", h.GetUserState)
	api.Get("/me/open-power", h.GetOpenPower)
	api.Get("/me/open-power/records", h.ListOpenPowerRecords)
}
