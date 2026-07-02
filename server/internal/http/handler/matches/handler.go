package matches

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	matchQuestionCount = 10
	roundDuration      = 15
	revealDuration     = 4
)

type Dependencies struct {
	Content     *content.Store
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	Broker      *Broker
}

type Handler struct {
	content *content.Store
	client  *mongo.Client
	db      *mongo.Database
	broker  *Broker
}

func New(dep Dependencies) *Handler {
	broker := dep.Broker
	if broker == nil {
		broker = NewBroker()
	}
	return &Handler{
		content: dep.Content,
		client:  dep.MongoClient,
		db:      dep.MongoDB,
		broker:  broker,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Post("/matches", h.Create)
	api.Get("/matches/computer/settings", h.ComputerSettings)
	api.Post("/matches/computer", h.CreateComputer)
	api.Post("/matches/join", h.Join)
	api.Get("/matches/open", h.Open)
	api.Get("/matches/{matchID}", h.Get)
	api.Put("/matches/{matchID}/loadout", h.UpdateLoadout)
	api.Post("/matches/{matchID}/ready", h.Ready)
	api.Post("/matches/{matchID}/answers", h.Answer)
	api.Get("/matches/{matchID}/events", h.Events)
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
