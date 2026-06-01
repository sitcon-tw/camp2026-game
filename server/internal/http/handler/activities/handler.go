package activities

import "github.com/go-chi/chi/v5"

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/activities", h.ListActivities)
	api.Get("/activities/{activityID}", h.GetActivity)
	api.Post("/activities/{activityID}/claims", h.ClaimActivity)
}
