package fronts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

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
	Context context.Context
	Log     *slog.Logger
	Broker  *FrontBroker
}

type Handler struct {
	content *content.Store
	db      *mongo.Database
	broker  *FrontBroker
	log     *slog.Logger
}

func New(dep Dependencies) *Handler {
	broker := dep.Broker
	if broker == nil {
		broker = NewFrontBroker()
	}
	h := &Handler{
		content: dep.Content,
		db:      dep.MongoDB,
		broker:  broker,
		log:     dep.Log,
	}
	if dep.Context != nil && h.db != nil {
		go h.runFrontTradeLoop(dep.Context)
	}
	return h
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Get("/fronts/current", h.Current)
	api.Get("/fronts/{frontID}", h.Get)
	api.Get("/fronts/{frontID}/leaderboard", h.Leaderboard)
	api.Get("/fronts/{frontID}/events", h.Events)
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
	httpx.WriteJSON(w, http.StatusOK, detailResponse(front, player.ID, player.TeamID, sitones, playerOpenPower))
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
	if h.content == nil {
		return mongomodel.Front{}, errFrontNotFound
	}
	template, ok := h.content.TerritoryMap()
	if !ok {
		return mongomodel.Front{}, errFrontNotFound
	}
	var front mongomodel.Front
	err := h.db.Collection(mongomodel.FrontsCollection).FindOne(ctx, bson.M{"_id": template.ID}).Decode(&front)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return frontFromTerritoryTemplate(template), nil
	}
	if err != nil {
		return mongomodel.Front{}, err
	}
	front = h.withFrontDefaults(front)
	front.Current = true
	return front, nil
}

func (h *Handler) frontByID(ctx context.Context, frontID string) (mongomodel.Front, error) {
	frontID = strings.TrimSpace(frontID)
	if frontID == "" {
		return mongomodel.Front{}, errFrontNotFound
	}
	if h.content == nil {
		return mongomodel.Front{}, errFrontNotFound
	}
	template, ok := h.content.GetTerritoryMap(frontID)
	if !ok {
		return mongomodel.Front{}, errFrontNotFound
	}

	var front mongomodel.Front
	err := h.db.Collection(mongomodel.FrontsCollection).
		FindOne(ctx, bson.M{"_id": frontID}).
		Decode(&front)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return frontFromTerritoryTemplate(template), nil
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
	front.MapMode = content.FrontMapModeTerritoryGrid
	if front.Territory == nil && h.content != nil {
		template, ok := h.content.GetTerritoryMap(front.MapID)
		if ok {
			fallback := frontFromTerritoryTemplate(template)
			front.Name = fallback.Name
			front.Teams = fallback.Teams
			front.Territory = fallback.Territory
			front.ActiveEvents = fallback.ActiveEvents
			front.Leaderboard = fallback.Leaderboard
		}
	}
	if h.content != nil {
		if template, ok := h.content.GetTerritoryMap(front.MapID); ok && template.ID == front.ID {
			front.Current = true
		}
	}
	if front.Territory != nil {
		for i := range front.Teams {
			if front.Teams[i].TradeHourlyLimit <= 0 {
				front.Teams[i].TradeHourlyLimit = frontTradeHourlyLimit
			}
		}
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
