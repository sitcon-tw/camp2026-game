package system

import (
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Dependencies struct {
	MongoClient *mongo.Client
}

type Handler struct {
	mongoClient *mongo.Client
}

func New(dep Dependencies) *Handler {
	return &Handler{
		mongoClient: dep.MongoClient,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/healthz", h.Health)
}
