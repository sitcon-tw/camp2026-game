package system

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

type Dependencies struct {
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
}

type Handler struct {
	mongoClient *mongo.Client
	db          *mongo.Database
}

func New(dep Dependencies) *Handler {
	return &Handler{
		mongoClient: dep.MongoClient,
		db:          dep.MongoDB,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/healthz", h.Health)
	api.Get("/maintenance", h.Maintenance)
}

type MaintenanceResponse struct {
	Enabled          bool       `json:"enabled"`
	Mode             string     `json:"mode"`
	Message          string     `json:"message"`
	BlocksNewMatches bool       `json:"blocksNewMatches"`
	BlocksWrites     bool       `json:"blocksWrites"`
	ActiveMatchCount int64      `json:"activeMatchCount"`
	OpenMatchCount   int64      `json:"openMatchCount"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	UpdatedAt        *time.Time `json:"updatedAt,omitempty"`
}

func (h *Handler) Maintenance(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		httpx.WriteJSON(w, http.StatusOK, maintenanceResponse(gamecontrol.DefaultSettings(), 0, 0))
		return
	}

	settings, err := gamecontrol.ReadSettings(r.Context(), h.db)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("maintenance state unavailable", "maintenance_settings_lookup_failed", err))
		return
	}
	activeMatchCount, openMatchCount, err := h.matchCounts(r)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("maintenance match counts unavailable", "maintenance_match_count_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, maintenanceResponse(settings, activeMatchCount, openMatchCount))
}

func (h *Handler) matchCounts(r *http.Request) (int64, int64, error) {
	collection := h.db.Collection(mongomodel.MatchesCollection)
	activeCount, err := collection.CountDocuments(r.Context(), bson.M{
		"status": mongomodel.MatchStatusActive,
	})
	if err != nil {
		return 0, 0, err
	}
	openCount, err := collection.CountDocuments(r.Context(), bson.M{
		"status": bson.M{"$in": []string{
			mongomodel.MatchStatusWaiting,
			mongomodel.MatchStatusActive,
		}},
	})
	if err != nil {
		return 0, 0, err
	}
	return activeCount, openCount, nil
}

func maintenanceResponse(settings gamecontrol.Settings, activeMatchCount int64, openMatchCount int64) MaintenanceResponse {
	settings.Normalize()
	var startedAt *time.Time
	if !settings.MaintenanceStartedAt.IsZero() {
		startedAt = &settings.MaintenanceStartedAt
	}
	var updatedAt *time.Time
	if !settings.UpdatedAt.IsZero() {
		updatedAt = &settings.UpdatedAt
	}

	return MaintenanceResponse{
		Enabled:          settings.MaintenanceActive(),
		Mode:             settings.MaintenanceMode,
		Message:          settings.MaintenanceMessage,
		BlocksNewMatches: settings.MaintenanceBlocksNewMatches(),
		BlocksWrites:     settings.MaintenanceBlocksWrites(),
		ActiveMatchCount: activeMatchCount,
		OpenMatchCount:   openMatchCount,
		StartedAt:        startedAt,
		UpdatedAt:        updatedAt,
	}
}
