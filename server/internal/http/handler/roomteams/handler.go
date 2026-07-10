package roomteams

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

type Dependencies struct {
	MongoDB *mongo.Database
}

type Handler struct {
	db *mongo.Database
}

func New(dep Dependencies) *Handler {
	return &Handler{db: dep.MongoDB}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Post("/room-teams/scans/{qrToken}/join", h.ScanJoin)
}

func (h *Handler) RegisterStaffRoutes(api chi.Router) {
	api.Get("/staff/room-teams", h.ListStaffRoomTeams)
	api.Post("/staff/room-teams/{roomNumber}/token", h.CreateStaffRoomTeamToken)
	api.Post("/staff/room-teams/{roomNumber}/members", h.AddStaffRoomTeamMember)
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
	if h.db != nil {
		return true
	}
	httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
	return false
}
