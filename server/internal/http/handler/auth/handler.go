package auth

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Dependencies struct {
	MongoDB *mongo.Database
}

type Handler struct {
	db *mongo.Database
}

func New(dep Dependencies) *Handler {
	return &Handler{
		db: dep.MongoDB,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Post("/auth/login", h.Login)
	api.Post("/auth/logout", h.Logout)
}
