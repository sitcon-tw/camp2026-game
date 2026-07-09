package territory

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/territory"
)

// ListCaptives godoc
// @Summary List captive records involving my team
// @Description Lists captives currently held by my team or originally belonging to my team.
// @Tags territory
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} ListCaptivesResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/captives [get]
func (h *Handler) ListCaptives(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !requireTeam(w, r, player) {
		return
	}

	ctx := r.Context()
	cursor, err := h.db.Collection(mongomodel.CaptiveRecordsCollection).Find(
		ctx,
		bson.M{"$or": bson.A{
			bson.M{"current_team_id": player.TeamID},
			bson.M{"original_team_id": player.TeamID},
		}},
		options.Find().SetSort(bson.D{{Key: "captured_at", Value: -1}}),
	)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("captives unavailable", "territory_captives_failed", err))
		return
	}
	var records []mongomodel.CaptiveRecord
	if err := cursor.All(ctx, &records); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("captives unavailable", "territory_captives_failed", err))
		return
	}

	now := time.Now()
	response := ListCaptivesResponse{Captives: make([]CaptiveResponse, 0, len(records))}
	for _, record := range records {
		status := record.Status
		if status == mongomodel.CaptiveRecordStatusCooldown && now.After(record.CooldownUntil) {
			status = mongomodel.CaptiveRecordStatusConvertible
		}
		entry := CaptiveResponse{
			ID:             record.ID,
			SitoneID:       record.SitoneID,
			Status:         status,
			OriginalTeamID: record.OriginalTeamID,
			CurrentTeamID:  record.CurrentTeamID,
			CapturedAt:     record.CapturedAt,
			CooldownUntil:  record.CooldownUntil,
			HeldByMyTeam:   record.CurrentTeamID == player.TeamID,
		}
		if sitone, ok := h.content.GetSitone(record.SitoneID); ok {
			entry.SitoneName = sitone.Name
		}
		if !record.ConvertedAt.IsZero() {
			convertedAt := record.ConvertedAt
			entry.ConvertedAt = &convertedAt
		}
		response.Captives = append(response.Captives, entry)
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// ConvertCaptive godoc
// @Summary Convert a captive to my team
// @Description Starts a convert process for a captive held by my team once its cooldown has ended. The executor pays; extra open power can be invested to improve the odds. Blocked while the boss is under attack.
// @Tags territory
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param captiveID path string true "Captive record id"
// @Param request body ConvertCaptiveRequest false "Convert options"
// @Success 201 {object} ProcessResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /territory/captives/{captiveID}/convert [post]
func (h *Handler) ConvertCaptive(w http.ResponseWriter, r *http.Request) {
	h.convertCaptiveByID(w, r, chi.URLParam(r, "captiveID"))
}

// convertCaptiveByID is shared with the yansan handler alias route.
func (h *Handler) convertCaptiveByID(w http.ResponseWriter, r *http.Request, captiveID string) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) || !requireTeam(w, r, player) {
		return
	}

	var body ConvertCaptiveRequest
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(r, &body); err != nil {
			httpx.WriteProblem(w, r, err)
			return
		}
		if err := httpx.ValidateStruct(body); err != nil {
			httpx.WriteProblem(w, r, err)
			return
		}
	}

	ctx := r.Context()
	captiveID = strings.TrimSpace(captiveID)
	var captive mongomodel.CaptiveRecord
	err := h.db.Collection(mongomodel.CaptiveRecordsCollection).FindOne(ctx, bson.M{"_id": captiveID}).Decode(&captive)
	if err == mongo.ErrNoDocuments {
		httpx.WriteProblem(w, r, httpx.NotFound("captive not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("convert failed", "territory_captive_lookup_failed", err))
		return
	}

	rank, err := h.service.PersonalRank(ctx, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("convert failed", "territory_rank_failed", err))
		return
	}
	members, err := h.service.TeamMembers(ctx, player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("convert failed", "territory_members_failed", err))
		return
	}

	process, err := h.service.StartConvert(ctx, player, captive, rank, len(members), body.ExtraOpenPower)
	if err != nil {
		writeProcessStartError(w, r, err, "convert")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, processResponse(process))
}

func writeProcessStartError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, territory.ErrBossUnderAttack):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "yansan services are suspended while the boss is under attack"))
	case errors.Is(err, territory.ErrCaptiveNotConvertible):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "captive is still in cooldown or already processed"))
	case errors.Is(err, territory.ErrProcessAlreadyPending):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "a pending process already exists for this target"))
	case errors.Is(err, territory.ErrInsufficientOpenPower):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient open power"))
	case errors.Is(err, territory.ErrNotYourTeam):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "target does not belong to your team"))
	case errors.Is(err, territory.ErrWrongState):
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "target is not in the required state"))
	default:
		httpx.WriteProblem(w, r, httpx.InternalServerError(action+" failed", "territory_"+action+"_failed", err))
	}
}
