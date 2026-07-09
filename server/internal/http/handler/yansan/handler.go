package yansan

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

type Dependencies struct {
	Content *content.Store
	MongoDB *mongo.Database
	// Service is the shared territory service (same instance as the
	// territory handler so both share tunables and locks).
	Service *territory.Service
}

type Handler struct {
	content *content.Store
	db      *mongo.Database
	service *territory.Service
}

func New(dep Dependencies) *Handler {
	return &Handler{
		content: dep.Content,
		db:      dep.MongoDB,
		service: dep.Service,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/yansan/boss/status", h.BossStatus)
	api.Post("/yansan/boss/attack", h.BossAttack)
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
