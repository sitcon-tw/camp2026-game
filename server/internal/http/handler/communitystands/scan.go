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
	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/qrcooldown"
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

	stand, err := h.findEnabledStandByQRToken(r.Context(), chi.URLParam(r, "qrToken"), time.Now().UTC())
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
// @Failure 429 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /community/scans/{qrToken}/claim [post]
func (h *Handler) ScanClaim(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	stand, err := h.findEnabledStandByQRToken(r.Context(), chi.URLParam(r, "qrToken"), time.Now().UTC())
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
	settings, err := gamecontrol.ReadSettings(r.Context(), h.db)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("qr code scan cooldown unavailable", "qrcode_scan_cooldown_settings_failed", err))
		return
	}
	reservation, reserved, err := h.reserveQRCodeScanCooldown(r.Context(), playerID, stand.ID, settings, time.Now().UTC())
	if errors.Is(err, qrcooldown.ErrActive) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusTooManyRequests, "qr code scan cooldown active"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("qr code scan cooldown unavailable", "qrcode_scan_cooldown_reserve_failed", err))
		return
	}
	releaseCooldown := func() {
		if !reserved {
			return
		}
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = qrcooldown.Release(releaseCtx, h.db, reservation)
	}

	if err := h.recordStandVisit(r.Context(), stand.ID, playerID); err != nil {
		releaseCooldown()
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand visit failed", "community_stand_visit_failed", err))
		return
	}

	claimID, err := h.claimStandReward(r.Context(), playerID, stand)
	if errors.Is(err, errCommunityStandAlreadyClaimed) {
		releaseCooldown()
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "community stand reward already claimed"))
		return
	}
	if err != nil {
		releaseCooldown()
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

func (h *Handler) reserveQRCodeScanCooldown(ctx context.Context, playerID string, sourceID string, settings gamecontrol.Settings, now time.Time) (qrcooldown.Reservation, bool, error) {
	duration := settings.QRCodeScanCooldownDuration()
	if duration <= 0 {
		return qrcooldown.Reservation{}, false, nil
	}
	reservation, err := qrcooldown.Reserve(ctx, h.db, playerID, "community_stand:"+sourceID, duration, now)
	if err != nil {
		return qrcooldown.Reservation{}, false, err
	}
	return reservation, true, nil
}

func (h *Handler) findEnabledStandByQRToken(ctx context.Context, qrToken string, now time.Time) (mongomodel.CommunityStand, error) {
	var stand mongomodel.CommunityStand
	err := h.db.Collection(mongomodel.CommunityStandsCollection).
		FindOne(ctx, enabledStandByQRTokenFilter(qrToken, now)).
		Decode(&stand)
	return stand, err
}

func enabledStandByQRTokenFilter(qrToken string, now time.Time) bson.M {
	return bson.M{
		"qr_token":            qrToken,
		"enabled":             true,
		"qr_token_expires_at": bson.M{"$gt": now},
	}
}

func (h *Handler) ensureStandQRToken(ctx context.Context, stand mongomodel.CommunityStand, now time.Time) (mongomodel.CommunityStand, error) {
	if standQRTokenActive(stand, now) {
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
			"_id":     stand.ID,
			"enabled": true,
			"$or": []bson.M{
				{"qr_token": bson.M{"$exists": false}},
				{"qr_token": ""},
				{"qr_token_expires_at": bson.M{"$exists": false}},
				{"qr_token_expires_at": bson.M{"$lte": now}},
			},
		},
		bson.M{"$set": bson.M{
			"qr_token":            qrToken,
			"qr_token_expires_at": now.Add(communityStandQRTokenTTL),
			"updated_at":          now,
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if errors.Is(err, mongo.ErrNoDocuments) {
		current, findErr := h.findEnabledStandByID(ctx, stand.ID)
		if findErr != nil {
			return mongomodel.CommunityStand{}, findErr
		}
		return current, nil
	}
	if err != nil {
		return mongomodel.CommunityStand{}, err
	}
	return updated, nil
}

func standQRTokenActive(stand mongomodel.CommunityStand, now time.Time) bool {
	return stand.QRToken != "" && stand.QRTokenExpiresAt.After(now)
}
