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
	Content            *content.Store
	MongoClient        *mongo.Client
	MongoDB            *mongo.Database
	Broker             *Broker
	RecoverOpenMatches bool
}

type Handler struct {
	content  *content.Store
	client   *mongo.Client
	db       *mongo.Database
	broker   *Broker
	sessions *MatchSessionManager
}

func New(dep Dependencies) *Handler {
	broker := dep.Broker
	if broker == nil {
		broker = NewBroker()
	}
	h := &Handler{
		content: dep.Content,
		client:  dep.MongoClient,
		db:      dep.MongoDB,
		broker:  broker,
	}
	h.sessions = NewMatchSessionManager(h)
	if dep.RecoverOpenMatches && h.db != nil {
		h.sessions.RecoverOpenMatches()
	}
	return h
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/matches/computer/settings", h.ComputerSettings)
	api.Post("/matches/computer", h.CreateComputer)
	api.Get("/matches/open", h.Open)
	api.Post("/matches/multiplayer/pairings", h.CreateMultiplayerPairing)
	api.Post("/matches/pairings", h.CreatePairing)
	api.Post("/matches/pairings/scan", h.ScanPairing)
	api.Post("/matches/{matchID}/pairing-token", h.CreatePairingToken)
	api.Get("/matches/{matchID}", h.Get)
	api.Post("/matches/{matchID}/leave", h.Leave)
	api.Put("/matches/{matchID}/loadout", h.UpdateLoadout)
	api.Post("/matches/{matchID}/computer-players", h.AddComputerPlayer)
	api.Post("/matches/{matchID}/rematch", h.Rematch)
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
