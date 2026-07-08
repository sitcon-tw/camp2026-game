package communitystands

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const communityStandQRTokenTTL = 2 * time.Minute

// Get godoc
// @Summary Get community stand by fixed stand ID
// @Description Returns the community stand shown after a player scans its QR code, including whether the current player already claimed the reward.
// @Tags community-stands
// @Produce json
// @Security AuthCookieAuth
// @Param standID path string true "Community stand ID from the scanned URL"
// @Success 200 {object} DetailResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /community/{standID} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	stand, err := h.findEnabledStandByID(r.Context(), chi.URLParam(r, "standID"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("community stand not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand unavailable", "community_stand_lookup_failed", err))
		return
	}
	reward, ok := h.rewardResponse(stand.Reward)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand reward unavailable", "community_stand_reward_missing", errors.New("reward content missing")))
		return
	}
	if err := h.recordStandVisit(r.Context(), stand.ID, player.ID); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand visit failed", "community_stand_visit_failed", err))
		return
	}
	claimed, err := h.playerClaimedStand(r.Context(), stand.ID, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand unavailable", "community_stand_claim_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, DetailResponse{
		Stand:   standResponse(stand, reward),
		Claimed: claimed,
	})
}

// Display godoc
// @Summary Get community stand display data
// @Description Returns public stand information and visit counters for community stand partners.
// @Tags community-stands
// @Produce json
// @Param standID path string true "Community stand ID"
// @Success 200 {object} DisplayResponse
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /community/{standID}/display [get]
func (h *Handler) Display(w http.ResponseWriter, r *http.Request) {
	if !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	stand, err := h.findEnabledStandByID(r.Context(), chi.URLParam(r, "standID"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("community stand not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand unavailable", "community_stand_lookup_failed", err))
		return
	}
	stand, err = h.ensureStandQRToken(r.Context(), stand, time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand QR token unavailable", "community_stand_qr_token_failed", err))
		return
	}
	reward, ok := h.rewardResponse(stand.Reward)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand reward unavailable", "community_stand_reward_missing", errors.New("reward content missing")))
		return
	}
	visitCount, claimCount, err := h.standCounters(r.Context(), stand.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand counters unavailable", "community_stand_counters_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, DisplayResponse{
		Stand:            standResponse(stand, reward),
		VisitCount:       visitCount,
		ClaimCount:       claimCount,
		QRToken:          stand.QRToken,
		QRTokenExpiresAt: stand.QRTokenExpiresAt,
	})
}

func (h *Handler) findEnabledStandByID(ctx context.Context, standID string) (mongomodel.CommunityStand, error) {
	var stand mongomodel.CommunityStand
	err := h.db.Collection(mongomodel.CommunityStandsCollection).
		FindOne(ctx, bson.M{"_id": standID, "enabled": true}).
		Decode(&stand)
	return stand, err
}

func (h *Handler) playerClaimedStand(ctx context.Context, standID string, playerID string) (bool, error) {
	count, err := h.db.Collection(mongomodel.CommunityStandClaimsCollection).CountDocuments(ctx, bson.M{
		"stand_id":  standID,
		"player_id": playerID,
	})
	return count > 0, err
}

func (h *Handler) recordStandVisit(ctx context.Context, standID string, playerID string) error {
	now := time.Now().UTC()
	_, err := h.db.Collection(mongomodel.CommunityStandVisitsCollection).UpdateOne(
		ctx,
		bson.M{"stand_id": standID, "player_id": playerID},
		bson.M{
			"$set": bson.M{"last_visited_at": now},
			"$setOnInsert": bson.M{
				"_id":              newID("community_visit"),
				"stand_id":         standID,
				"player_id":        playerID,
				"first_visited_at": now,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func (h *Handler) standCounters(ctx context.Context, standID string) (int64, int64, error) {
	visitCount, err := h.db.Collection(mongomodel.CommunityStandVisitsCollection).CountDocuments(ctx, bson.M{"stand_id": standID})
	if err != nil {
		return 0, 0, err
	}
	claimCount, err := h.db.Collection(mongomodel.CommunityStandClaimsCollection).CountDocuments(ctx, bson.M{"stand_id": standID})
	if err != nil {
		return 0, 0, err
	}
	return visitCount, claimCount, nil
}

func (h *Handler) rewardResponse(reward mongomodel.StandReward) (RewardResponse, bool) {
	response := RewardResponse{
		Kind:     reward.Kind,
		RefID:    reward.RefID,
		Quantity: reward.Quantity,
		Amount:   reward.Amount,
	}
	switch reward.Kind {
	case rewardKindOpenPower:
		response.Name = "開源力"
		return response, true
	case rewardKindItem:
		item, ok := h.content.GetItem(reward.RefID)
		if !ok || !item.Enabled {
			return RewardResponse{}, false
		}
		response.Name = item.Name
		response.IconPath = item.IconPath
		return response, true
	case rewardKindSitone:
		sitone, ok := h.content.GetSitone(reward.RefID)
		if !ok {
			return RewardResponse{}, false
		}
		response.Name = sitone.Name
		response.IconPath = sitone.IconPath
		return response, true
	default:
		return RewardResponse{}, false
	}
}
