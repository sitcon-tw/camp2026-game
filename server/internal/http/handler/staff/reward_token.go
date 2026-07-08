package staff

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const staffRewardTokenTTL = 10 * time.Minute

var errRewardTokenAlreadyClaimed = errors.New("staff reward token already claimed")

// CreateRewardToken godoc
// @Summary Create a short-lived staff reward QR token
// @Description Staff-only endpoint. Creates a short-lived token that authenticated players can scan to claim the selected reward once per player before expiry.
// @Tags staff
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body CreateRewardTokenRequest true "Staff reward token request"
// @Success 201 {object} CreateRewardTokenResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/reward-tokens [post]
func (h *Handler) CreateRewardToken(w http.ResponseWriter, r *http.Request) {
	staffPlayer, ok := currentStaff(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	var body CreateRewardTokenRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.Kind = strings.TrimSpace(body.Kind)
	body.RefID = strings.TrimSpace(body.RefID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := validateRewardTokenBody(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	reward, found := h.rewardDefinition(body.Kind, body.RefID)
	if !found {
		httpx.WriteProblem(w, r, httpx.NotFound("reward content not found"))
		return
	}

	now := time.Now().UTC()
	token, err := newRewardToken()
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward token failed", "reward_token_generate_failed", err))
		return
	}

	rewardValue := body.Quantity
	if body.Kind == rewardKindOpenPower {
		rewardValue = body.Amount
	}
	record := mongomodel.StaffRewardToken{
		ID:            newID("staff_reward_token"),
		Token:         token,
		StaffPlayerID: staffPlayer.ID,
		Kind:          reward.kind,
		RefID:         reward.id,
		Quantity:      body.Quantity,
		Amount:        body.Amount,
		CreatedAt:     now,
		ExpiresAt:     now.Add(staffRewardTokenTTL),
	}
	if record.Kind == rewardKindOpenPower {
		record.Quantity = 0
		record.Amount = rewardValue
	}
	if err := h.insertRewardToken(r.Context(), record); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward token failed", "reward_token_create_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, CreateRewardTokenResponse{
		Token:     record.Token,
		ExpiresAt: record.ExpiresAt.Format(time.RFC3339),
		Reward: RewardResponse{
			Kind:     reward.kind,
			ID:       reward.id,
			Name:     reward.name,
			Quantity: body.Quantity,
			Amount:   body.Amount,
		},
	})
}

// ClaimRewardToken godoc
// @Summary Claim a scanned staff reward QR token
// @Description Claims a short-lived staff reward token for the authenticated player. Each player can claim each token once before expiry.
// @Tags staff
// @Produce json
// @Security AuthCookieAuth
// @Param token path string true "Opaque staff reward token"
// @Success 201 {object} ClaimRewardTokenResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/reward-tokens/{token}/claim [post]
func (h *Handler) ClaimRewardToken(w http.ResponseWriter, r *http.Request) {
	player, ok := authctx.PlayerFromContext(r.Context())
	if !ok {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "authentication required"))
		return
	}
	if !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	token := strings.TrimSpace(chi.URLParam(r, "token"))
	record, err := h.findActiveRewardToken(r.Context(), token, time.Now().UTC())
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("reward token not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward token unavailable", "reward_token_lookup_failed", err))
		return
	}

	reward, found := h.rewardDefinition(record.Kind, record.RefID)
	if !found {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward token unavailable", "reward_token_content_missing", errors.New("reward content missing")))
		return
	}

	staffPlayer, err := h.findPlayerByID(r.Context(), record.StaffPlayerID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward token unavailable", "reward_token_staff_lookup_failed", err))
		return
	}

	rewardValue := record.Quantity
	if record.Kind == rewardKindOpenPower {
		rewardValue = record.Amount
	}
	rewardRecord, err := h.claimRewardToken(r.Context(), player.ID, staffPlayer.ID, record.ID, reward, rewardValue)
	if errors.Is(err, errRewardTokenAlreadyClaimed) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "reward token already claimed"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward token claim failed", "reward_token_claim_failed", err))
		return
	}
	h.publishRewardGranted(r.Context(), player.ID, staffPlayer, rewardRecord)

	httpx.WriteJSON(w, http.StatusCreated, ClaimRewardTokenResponse{
		RewardID: rewardRecord.ID,
		Reward: RewardResponse{
			Kind:     reward.kind,
			ID:       reward.id,
			Name:     reward.name,
			Quantity: record.Quantity,
			Amount:   record.Amount,
		},
		Staff: RewardStaffResponse{
			PlayerID: staffPlayer.ID,
			Nickname: staffPlayer.Nickname,
		},
	})
}

func validateRewardTokenBody(body CreateRewardTokenRequest) error {
	return validateRewardBody(CreateRewardRequest{
		Kind:     body.Kind,
		RefID:    body.RefID,
		Quantity: body.Quantity,
		Amount:   body.Amount,
	})
}

func newRewardToken() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "srt_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func (h *Handler) insertRewardToken(ctx context.Context, token mongomodel.StaffRewardToken) error {
	_, err := h.db.Collection(mongomodel.StaffRewardTokensCollection).InsertOne(ctx, token)
	return err
}

func (h *Handler) findActiveRewardToken(ctx context.Context, token string, now time.Time) (mongomodel.StaffRewardToken, error) {
	var record mongomodel.StaffRewardToken
	err := h.db.Collection(mongomodel.StaffRewardTokensCollection).
		FindOne(ctx, bson.M{
			"token":      token,
			"expires_at": bson.M{"$gt": now},
		}).
		Decode(&record)
	return record, err
}

func (h *Handler) claimRewardToken(ctx context.Context, playerID string, staffPlayerID string, tokenID string, reward rewardDefinition, quantity int) (mongomodel.StaffReward, error) {
	rewardID := newID("staff_reward")
	claimID := newID("staff_reward_token_claim")
	claim := mongomodel.StaffRewardTokenClaim{
		ID:            claimID,
		TokenID:       tokenID,
		PlayerID:      playerID,
		StaffRewardID: rewardID,
		ClaimedAt:     time.Now().UTC(),
	}
	if err := h.insertRewardTokenClaim(ctx, claim); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return mongomodel.StaffReward{}, errRewardTokenAlreadyClaimed
		}
		return mongomodel.StaffReward{}, err
	}

	rewardRecord, err := h.createRewardWithID(ctx, rewardID, staffPlayerID, playerID, reward, quantity, claim.ClaimedAt)
	if err != nil {
		_ = h.deleteRewardTokenClaim(ctx, claimID)
		return mongomodel.StaffReward{}, err
	}
	return rewardRecord, nil
}

func (h *Handler) insertRewardTokenClaim(ctx context.Context, claim mongomodel.StaffRewardTokenClaim) error {
	_, err := h.db.Collection(mongomodel.StaffRewardTokenClaimsCollection).InsertOne(ctx, claim)
	return err
}

func (h *Handler) deleteRewardTokenClaim(ctx context.Context, claimID string) error {
	_, err := h.db.Collection(mongomodel.StaffRewardTokenClaimsCollection).DeleteOne(ctx, bson.M{"_id": claimID})
	return err
}
