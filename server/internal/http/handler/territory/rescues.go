package territory

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// CreateRescue godoc
// @Summary Start a rescue for a missing sitone
// @Description Starts a rescue process for a missing sitone instance of my team. The executor pays; the cost is keyed on the original holder. Resolved later by the central sweeper.
// @Tags territory
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body CreateRescueRequest true "Rescue request"
// @Success 201 {object} ProcessResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/rescues [post]
func (h *Handler) CreateRescue(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body CreateRescueRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.SitoneID = strings.TrimSpace(body.SitoneID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	ctx := r.Context()

	// Find a missing instance of my team for that sitone.
	var instance mongomodel.SitoneInstance
	err := h.db.Collection(mongomodel.SitoneInstancesCollection).FindOne(
		ctx,
		bson.M{
			"team_id":   player.TeamID,
			"sitone_id": body.SitoneID,
			"status":    mongomodel.SitoneInstanceStatusMissing,
		},
		options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: 1}}),
	).Decode(&instance)
	if err == mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.NotFound("no missing sitone of that kind in your team"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("rescue failed", "territory_instance_lookup_failed", err))
		return
	}

	// Cost keyed on the original holder's personal rank (design §5).
	rank, err := h.service.PersonalRank(ctx, instance.OriginPlayerID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("rescue failed", "territory_rank_failed", err))
		return
	}
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("rescue failed", "territory_members_failed", err))
		return
	}

	process, err := h.service.StartRescue(ctx, player, instance, rank, len(members))
	if err != nil {
		writeProcessStartError(w, r, err, "rescue")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, processResponse(process))
}

// GetRescue godoc
// @Summary Get a rescue process
// @Tags territory
// @Produce json
// @Security AuthCookieAuth
// @Param rescueID path string true "Rescue process id"
// @Success 200 {object} ProcessResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/rescues/{rescueID} [get]
func (h *Handler) GetRescue(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !requireTeam(w, r, player) {
		return
	}

	rescueID := strings.TrimSpace(chi.URLParam(r, "rescueID"))
	var process mongomodel.YansanProcess
	err := h.db.Collection(mongomodel.YansanProcessesCollection).FindOne(r.Context(), bson.M{
		"_id":     rescueID,
		"kind":    mongomodel.YansanProcessKindRescue,
		"team_id": player.TeamID,
	}).Decode(&process)
	if err == mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.NotFound("rescue not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("rescue unavailable", "territory_rescue_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, processResponse(process))
}
