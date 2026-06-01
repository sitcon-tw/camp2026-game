package matches

import "github.com/go-chi/chi/v5"

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/matches", h.ListMatches)
	api.Post("/matches", h.CreateMatch)
	api.Get("/matches/{matchID}", h.GetMatch)
	api.Post("/matches/{matchID}/answers", h.SubmitAnswer)
	api.Post("/matches/{matchID}/finish", h.FinishMatch)
}
