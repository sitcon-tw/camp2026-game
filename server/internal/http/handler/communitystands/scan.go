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

	standtokens "github.com/sitcon-tw/camp2026-game/internal/communitystand"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ScanGet godoc
// @Summary Get community stand by QR token
// @Description Returns the community stand shown after a player scans its QR code. The QR token is opaque and does not contain the stand ID.
// @Tags community-stands
// @Produce json
// @Security AuthCookieAuth
// @Param qrToken path string true "Opaque community stand QR token"
// @Success 200 {object} DetailResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /community/scans/{qrToken} [get]
func (h *Handler) ScanGet(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	stand, err := h.findEnabledStandByQRToken(r.Context(), chi.URLParam(r, "qrToken"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("community stand not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand unavailable", "community_stand_lookup_failed", err))
		return
	}
	h.writeStandDetail(w, r, player.ID, stand)
}

// ScanClaim godoc
// @Summary Claim community stand reward by QR token
// @Description Claims the scanned community stand reward for the authenticated player. Each player can claim each stand once.
// @Tags community-stands
// @Produce json
// @Security AuthCookieAuth
// @Param qrToken path string true "Opaque community stand QR token"
// @Success 201 {object} ClaimResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /community/scans/{qrToken}/claim [post]
func (h *Handler) ScanClaim(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	stand, err := h.findEnabledStandByQRToken(r.Context(), chi.URLParam(r, "qrToken"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("community stand not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand unavailable", "community_stand_lookup_failed", err))
		return
	}
	h.writeStandClaim(w, r, player.ID, stand)
}

func (h *Handler) writeStandDetail(w http.ResponseWriter, r *http.Request, playerID string, stand mongomodel.CommunityStand) {
	reward, ok := h.rewardResponse(stand.Reward)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand reward unavailable", "community_stand_reward_missing", errors.New("reward content missing")))
		return
	}
	if err := h.recordStandVisit(r.Context(), stand.ID, playerID); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand visit failed", "community_stand_visit_failed", err))
		return
	}
	claimed, err := h.playerClaimedStand(r.Context(), stand.ID, playerID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand unavailable", "community_stand_claim_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, DetailResponse{
		Stand:   standResponse(stand, reward),
		Claimed: claimed,
	})
}

func (h *Handler) writeStandClaim(w http.ResponseWriter, r *http.Request, playerID string, stand mongomodel.CommunityStand) {
	reward, ok := h.rewardResponse(stand.Reward)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand reward unavailable", "community_stand_reward_missing", errors.New("reward content missing")))
		return
	}
	if err := h.recordStandVisit(r.Context(), stand.ID, playerID); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand visit failed", "community_stand_visit_failed", err))
		return
	}

	claimID, err := h.claimStandReward(r.Context(), playerID, stand)
	if errors.Is(err, errCommunityStandAlreadyClaimed) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "community stand reward already claimed"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand claim failed", "community_stand_claim_failed", err))
		return
	}
	h.publishRewardGranted(playerID, reward)

	httpx.WriteJSON(w, http.StatusCreated, ClaimResponse{
		ClaimID: claimID,
		Stand:   standResponse(stand, reward),
		Reward:  reward,
		Claimed: true,
	})
}

func (h *Handler) findEnabledStandByQRToken(ctx context.Context, qrToken string) (mongomodel.CommunityStand, error) {
	var stand mongomodel.CommunityStand
	err := h.db.Collection(mongomodel.CommunityStandsCollection).
		FindOne(ctx, bson.M{"qr_token": qrToken, "enabled": true}).
		Decode(&stand)
	return stand, err
}

func (h *Handler) ensureStandQRToken(ctx context.Context, stand mongomodel.CommunityStand) (mongomodel.CommunityStand, error) {
	if stand.QRToken != "" {
		return stand, nil
	}

	qrToken, err := standtokens.NewQRToken()
	if err != nil {
		return mongomodel.CommunityStand{}, err
	}

	var updated mongomodel.CommunityStand
	err = h.db.Collection(mongomodel.CommunityStandsCollection).FindOneAndUpdate(
		ctx,
		bson.M{
			"_id": stand.ID,
			"$or": []bson.M{
				{"qr_token": bson.M{"$exists": false}},
				{"qr_token": ""},
			},
		},
		bson.M{"$set": bson.M{
			"qr_token":   qrToken,
			"updated_at": time.Now().UTC(),
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return h.findEnabledStandByID(ctx, stand.ID)
	}
	if err != nil {
		return mongomodel.CommunityStand{}, err
	}
	return updated, nil
}
