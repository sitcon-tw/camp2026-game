package territory

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/eventlog"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

// GetDefense godoc
// @Summary Get team defense configuration
// @Tags territory
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} DefenseResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/defense [get]
func (h *Handler) GetDefense(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !requireTeam(w, r, player) {
		return
	}

	ctx := r.Context()
	var defense mongomodel.TeamDefense
	err := h.db.Collection(mongomodel.TeamDefensesCollection).FindOne(ctx, bson.M{"_id": player.TeamID}).Decode(&defense)
	if err != nil && err != mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.InternalServerError("defense unavailable", "territory_defense_lookup_failed", err))
		return
	}
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("defense unavailable", "territory_members_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, defenseResponse(defense, territory.DeployCap(len(members))))
}

// UpdateDefense godoc
// @Summary Update team defense configuration
// @Description Sets the standing defense sitones. Stones must be owned across team members with free quantity; they are NOT instance-locked — defense is read at resolve time. Each change starts a short modify cooldown.
// @Tags territory
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body UpdateDefenseRequest true "Defense configuration"
// @Success 200 {object} DefenseResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/defense [put]
func (h *Handler) UpdateDefense(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body UpdateDefenseRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	ctx := r.Context()
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("defense update failed", "territory_members_failed", err))
		return
	}
	cap := territory.DeployCap(len(members))
	if len(body.SitoneIDs) > cap {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("too many defense sitones for your team size"))
		return
	}

	// Ownership across team members with free quantity (design §0.1
	// decision 7: no instance lock — the config is read at resolve time).
	memberIDs := make([]string, 0, len(members))
	for _, member := range members {
		memberIDs = append(memberIDs, member.ID)
	}
	teamQuantities, err := h.service.TeamQuantities(ctx, memberIDs)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("defense update failed", "territory_quantities_failed", err))
		return
	}
	needed := make(map[string]int, len(body.SitoneIDs))
	for _, sitoneID := range body.SitoneIDs {
		sitone, ok := h.content.GetSitone(sitoneID)
		if !ok {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity("unknown sitone "+sitoneID))
			return
		}
		if !territory.CanDefendWithSitone(sitone) {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity("sitone "+sitoneID+" cannot be deployed"))
			return
		}
		needed[sitoneID]++
	}
	for sitoneID, n := range needed {
		if teamQuantities[sitoneID] < n {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity("your team does not own enough of sitone "+sitoneID))
			return
		}
	}

	now := time.Now().UTC()

	// Modify cooldown (decision 7): reject while locked, atomically.
	result, err := h.db.Collection(mongomodel.TeamDefensesCollection).UpdateOne(
		ctx,
		bson.M{
			"_id": player.TeamID,
			"$or": bson.A{
				bson.M{"locked_until": bson.M{"$exists": false}},
				bson.M{"locked_until": bson.M{"$lte": now}},
			},
		},
		bson.M{"$set": bson.M{
			"sitone_ids":           body.SitoneIDs,
			"updated_by_player_id": player.ID,
			"updated_at":           now,
			"locked_until":         now.Add(territory.DefenseModifyCooldown),
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "defense was modified recently, try again later"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("defense update failed", "territory_defense_update_failed", err))
		return
	}
	if result.MatchedCount == 0 && result.UpsertedCount == 0 {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "defense was modified recently, try again later"))
		return
	}

	_, _ = eventlog.Insert(ctx, h.db, mongomodel.EventLog{
		EventType:     mongomodel.EventTypeDefenseUpdated,
		ActorPlayerID: player.ID,
		TeamID:        player.TeamID,
		Payload:       bson.M{"sitone_ids": body.SitoneIDs},
	})

	httpx.WriteJSON(w, http.StatusOK, defenseResponse(mongomodel.TeamDefense{
		ID:                player.TeamID,
		SitoneIDs:         body.SitoneIDs,
		UpdatedByPlayerID: player.ID,
		UpdatedAt:         now,
		LockedUntil:       now.Add(territory.DefenseModifyCooldown),
	}, cap))
}

func defenseResponse(defense mongomodel.TeamDefense, cap int) DefenseResponse {
	response := DefenseResponse{
		SitoneIDs: defense.SitoneIDs,
		Cap:       cap,
	}
	if response.SitoneIDs == nil {
		response.SitoneIDs = []string{}
	}
	if !defense.LockedUntil.IsZero() {
		lockedUntil := defense.LockedUntil
		response.LockedUntil = &lockedUntil
	}
	if !defense.UpdatedAt.IsZero() {
		updatedAt := defense.UpdatedAt
		response.UpdatedAt = &updatedAt
	}
	return response
}
