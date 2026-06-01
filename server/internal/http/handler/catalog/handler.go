package catalog

import "github.com/go-chi/chi/v5"

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/catalog/sitones", h.ListSitones)
	api.Get("/catalog/items", h.ListItems)
	api.Get("/catalog/crafting-recipes", h.ListRecipes)
}
