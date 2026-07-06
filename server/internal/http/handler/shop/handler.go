package shop

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

type Dependencies struct {
	Content     *content.Store
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	Broker      *playerevents.Broker
}

type Handler struct {
	content *content.Store
	client  *mongo.Client
	db      *mongo.Database
	broker  *playerevents.Broker
}

func New(dep Dependencies) *Handler {
	return &Handler{
		content: dep.Content,
		client:  dep.MongoClient,
		db:      dep.MongoDB,
		broker:  dep.Broker,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/shop/items", h.ListItems)
	api.Get("/shop/items/{itemID}", h.GetItem)
	api.Post("/shop/purchases", h.Purchase)
}

func currentPlayer(w http.ResponseWriter, r *http.Request) (mongomodel.Player, bool) {
	player, ok := authctx.PlayerFromContext(r.Context())
	if !ok {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "authentication required"))
		return mongomodel.Player{}, false
	}
	return player, true
}

func (h *Handler) requireDatabase(w http.ResponseWriter, r *http.Request) bool {
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return false
	}
	return true
}

func (h *Handler) requireContent(w http.ResponseWriter, r *http.Request) bool {
	if h.content == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("content store is unavailable"))
		return false
	}
	return true
}
