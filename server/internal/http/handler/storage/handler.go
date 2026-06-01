package storage

import "github.com/go-chi/chi/v5"

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/me/sitones", h.ListSitones)
	api.Get("/me/sitones/{sitoneID}", h.GetSitone)
	api.Get("/me/items", h.ListItems)
	api.Get("/me/items/{itemInstanceID}", h.GetItem)
	api.Get("/crafting/recipes", h.ListRecipes)
	api.Get("/crafting/recipes/{recipeID}", h.GetRecipe)
	api.Post("/crafting", h.CraftSitone)
}
