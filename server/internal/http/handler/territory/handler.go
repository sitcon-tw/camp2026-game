package territory

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

type Dependencies struct {
	Log         *slog.Logger
	Content     *content.Store
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	// StartSweeper launches the central resolution sweeper goroutine
	// (design §0.1 decision 5) and seeds the yansan boss team.
	StartSweeper bool
}

type Handler struct {
	content *content.Store
	client  *mongo.Client
	db      *mongo.Database
	service *territory.Service
	sweeper *territory.Sweeper
}

func New(dep Dependencies) *Handler {
	h := &Handler{
		content: dep.Content,
		client:  dep.MongoClient,
		db:      dep.MongoDB,
	}
	if dep.MongoDB != nil {
		h.service = territory.NewService(dep.MongoClient, dep.MongoDB, dep.Content)
		if dep.StartSweeper {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = territory.EnsureYansanTeam(ctx, dep.MongoDB)
			}()
			h.sweeper = territory.StartSweeper(h.service, dep.Log, territory.DefaultSweepInterval)
		}
	}
	return h
}

// Service exposes the shared territory service (used by the yansan
// handler package so both share one implementation).
func (h *Handler) Service() *territory.Service {
	return h.service
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/territory/standings", h.Standings)
	api.Post("/territory/attacks", h.CreateAttack)
	api.Get("/territory/attacks/{missionID}", h.GetAttack)
	api.Post("/territory/attacks/{missionID}/votes", h.VoteAttack)
	api.Post("/territory/attacks/{missionID}/cancel", h.CancelAttack)
	api.Get("/territory/defense", h.GetDefense)
	api.Put("/territory/defense", h.UpdateDefense)
	api.Get("/territory/captives", h.ListCaptives)
	api.Post("/territory/captives/{captiveID}/convert", h.ConvertCaptive)
	api.Post("/territory/rescues", h.CreateRescue)
	api.Get("/territory/rescues/{rescueID}", h.GetRescue)
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
	if h.db == nil || h.service == nil {
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

func requireTeam(w http.ResponseWriter, r *http.Request, player mongomodel.Player) bool {
	if player.TeamID == "" {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "you are not in a team"))
		return false
	}
	return true
}
