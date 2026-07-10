package fronts

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

var errFrontNotFound = errors.New("front not found")

type Dependencies struct {
	Content *content.Store
	MongoDB *mongo.Database
}

type Handler struct {
	content *content.Store
	db      *mongo.Database
}

func New(dep Dependencies) *Handler {
	return &Handler{
		content: dep.Content,
		db:      dep.MongoDB,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/fronts/current", h.Current)
	api.Get("/fronts/{frontID}", h.Get)
	api.Get("/fronts/{frontID}/leaderboard", h.Leaderboard)
	api.Post("/fronts/{frontID}/commands", h.CreateCommand)
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

// Current godoc
// @Summary Get current front
// @Description Returns the current front snapshot. Falls back to a deterministic bootstrap snapshot when MongoDB has no current front document.
// @Tags fronts
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} CurrentResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fronts/current [get]
func (h *Handler) Current(w http.ResponseWriter, r *http.Request) {
	_, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	front, err := h.currentFront(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front unavailable", "front_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, CurrentResponse{
		Front: frontSummaryResponse(front),
	})
}

// Get godoc
// @Summary Get front
// @Description Returns a front snapshot by ID. Falls back to deterministic bootstrap fronts when MongoDB has no matching document.
// @Tags fronts
// @Produce json
// @Security AuthCookieAuth
// @Param frontID path string true "Front ID"
// @Success 200 {object} DetailResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fronts/{frontID} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	front, err := h.frontByID(r.Context(), chi.URLParam(r, "frontID"))
	if errors.Is(err, errFrontNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("front not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front unavailable", "front_lookup_failed", err))
		return
	}
	front = withCurrentPlayerTeam(front, player)
	sitones, err := h.playerFrontSitones(r.Context(), player)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front sitones unavailable", "front_sitones_lookup_failed", err))
		return
	}
	playerOpenPower, err := openpower.TotalForPlayer(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front open power unavailable", "front_open_power_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detailResponse(front, player.TeamID, sitones, playerOpenPower))
}

// Leaderboard godoc
// @Summary Get front leaderboard
// @Description Returns the ranked front leaderboard from the snapshot.
// @Tags fronts
// @Produce json
// @Security AuthCookieAuth
// @Param frontID path string true "Front ID"
// @Success 200 {object} LeaderboardResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fronts/{frontID}/leaderboard [get]
func (h *Handler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	front, err := h.frontByID(r.Context(), chi.URLParam(r, "frontID"))
	if errors.Is(err, errFrontNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("front not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front leaderboard unavailable", "front_leaderboard_lookup_failed", err))
		return
	}
	front = withCurrentPlayerTeam(front, player)
	httpx.WriteJSON(w, http.StatusOK, LeaderboardResponse{
		FrontID: front.ID,
		Entries: leaderboardResponse(
			rankedLeaderboard(front),
			player.TeamID,
		),
	})
}

func (h *Handler) currentFront(ctx context.Context) (mongomodel.Front, error) {
	if h.content != nil {
		if territoryTemplate, ok := h.content.TerritoryMap(); ok {
			var territoryFront mongomodel.Front
			err := h.db.Collection(mongomodel.FrontsCollection).
				FindOne(ctx, bson.M{"_id": territoryTemplate.ID}).
				Decode(&territoryFront)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return frontFromTerritoryTemplate(territoryTemplate), nil
			}
			if err != nil {
				return mongomodel.Front{}, err
			}
			territoryFront = h.withFrontDefaults(territoryFront)
			territoryFront.Current = true
			return territoryFront, nil
		}
	}

	var front mongomodel.Front
	err := h.db.Collection(mongomodel.FrontsCollection).
		FindOne(
			ctx,
			bson.M{"current": true},
			options.FindOne().SetSort(bson.D{
				{Key: "updated_at", Value: -1},
				{Key: "_id", Value: 1},
			}),
		).
		Decode(&front)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return h.fallbackCurrentFront(), nil
	}
	if err != nil {
		return mongomodel.Front{}, err
	}
	return h.withFrontDefaults(front), nil
}

func (h *Handler) frontByID(ctx context.Context, frontID string) (mongomodel.Front, error) {
	frontID = strings.TrimSpace(frontID)
	if frontID == "" {
		return mongomodel.Front{}, errFrontNotFound
	}

	var front mongomodel.Front
	err := h.db.Collection(mongomodel.FrontsCollection).
		FindOne(ctx, bson.M{"_id": frontID}).
		Decode(&front)
	if errors.Is(err, mongo.ErrNoDocuments) {
		fallback, ok := h.fallbackFrontByID(frontID)
		if !ok {
			return mongomodel.Front{}, errFrontNotFound
		}
		return fallback, nil
	}
	if err != nil {
		return mongomodel.Front{}, err
	}
	if front.ID == "" {
		return mongomodel.Front{}, errFrontNotFound
	}
	return h.withFrontDefaults(front), nil
}

func (h *Handler) withFrontDefaults(front mongomodel.Front) mongomodel.Front {
	if front.MapID == "" {
		front.MapID = front.ID
	}
	if front.Name == "" {
		front.Name = front.MapID
	}
	if front.Status == "" {
		front.Status = mongomodel.FrontStatusOpenPlay
	}
	if front.Revision <= 0 {
		front.Revision = 1
	}
	if front.MapMode == "" && h.content != nil {
		if _, ok := h.content.GetTerritoryMap(front.MapID); ok {
			front.MapMode = content.FrontMapModeTerritoryGrid
		}
	}
	if front.MapMode == content.FrontMapModeTerritoryGrid && front.Territory == nil && h.content != nil {
		template, ok := h.content.GetTerritoryMap(front.MapID)
		if ok {
			fallback := frontFromTerritoryTemplate(template)
			front.Name = fallback.Name
			front.Teams = fallback.Teams
			front.Territory = fallback.Territory
			front.ActiveEvents = fallback.ActiveEvents
			front.Leaderboard = fallback.Leaderboard
		}
	} else if len(front.Cells) == 0 && h.content != nil {
		template, ok := h.content.GetFrontMap(front.MapID)
		if ok && template.Enabled {
			fallback := frontFromTemplate(template)
			front.Name = fallback.Name
			front.Cells = fallback.Cells
			front.Teams = fallback.Teams
			front.ActiveEvents = fallback.ActiveEvents
		}
	}
	for i := range front.Cells {
		front.Cells[i].Control = clampFrontInt(front.Cells[i].Control, 0, 100)
		front.Cells[i].Defense = clampFrontInt(front.Cells[i].Defense, 0, territoryMaxDefense)
	}
	if front.MapMode == content.FrontMapModeTerritoryGrid && h.content != nil {
		if template, ok := h.content.GetTerritoryMap(front.MapID); ok && template.ID == front.ID {
			front.Current = true
		}
	}
	if front.MapMode == content.FrontMapModeTerritoryGrid && front.Territory != nil {
		syncTerritoryTeamRanks(front.Teams, front.Territory)
		front.Leaderboard = deriveLeaderboard(front)
	}
	if front.CreatedAt.IsZero() {
		front.CreatedAt = seedFrontUpdatedAt
	}
	if front.UpdatedAt.IsZero() {
		front.UpdatedAt = front.CreatedAt
	}
	return front
}
