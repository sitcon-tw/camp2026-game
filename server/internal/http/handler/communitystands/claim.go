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
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var errCommunityStandAlreadyClaimed = errors.New("community stand already claimed")

// Claim godoc
// @Summary Claim community stand reward
// @Description Claims the scanned community stand reward for the authenticated player. Each player can claim each stand once.
// @Tags community-stands
// @Produce json
// @Security AuthCookieAuth
// @Param standID path string true "Community stand ID from the scanned URL"
// @Success 201 {object} ClaimResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /community/{standID}/claim [post]
func (h *Handler) Claim(w http.ResponseWriter, r *http.Request) {
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

	claimID, err := h.claimStandReward(r.Context(), player.ID, stand)
	if errors.Is(err, errCommunityStandAlreadyClaimed) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "community stand reward already claimed"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand claim failed", "community_stand_claim_failed", err))
		return
	}
	h.publishRewardGranted(player.ID, reward)

	httpx.WriteJSON(w, http.StatusCreated, ClaimResponse{
		ClaimID: claimID,
		Stand:   standResponse(stand, reward),
		Reward:  reward,
		Claimed: true,
	})
}

func (h *Handler) claimStandReward(ctx context.Context, playerID string, stand mongomodel.CommunityStand) (string, error) {
	claimID := newID("community_claim")
	claim := mongomodel.CommunityStandClaim{
		ID:        claimID,
		StandID:   stand.ID,
		PlayerID:  playerID,
		RewardID:  claimID,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.insertClaim(ctx, claim); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", errCommunityStandAlreadyClaimed
		}
		return "", err
	}
	if err := h.grantReward(ctx, playerID, claimID, stand.Reward); err != nil {
		_ = h.deleteClaim(ctx, claimID)
		return "", err
	}
	return claimID, nil
}

func (h *Handler) insertClaim(ctx context.Context, claim mongomodel.CommunityStandClaim) error {
	_, err := h.db.Collection(mongomodel.CommunityStandClaimsCollection).InsertOne(ctx, claim)
	return err
}

func (h *Handler) deleteClaim(ctx context.Context, claimID string) error {
	_, err := h.db.Collection(mongomodel.CommunityStandClaimsCollection).DeleteOne(ctx, bson.M{"_id": claimID})
	return err
}

func (h *Handler) grantReward(ctx context.Context, playerID string, claimID string, reward mongomodel.StandReward) error {
	switch reward.Kind {
	case rewardKindOpenPower:
		return h.insertOpenPowerReward(ctx, playerID, claimID, reward.Amount)
	case rewardKindItem, rewardKindSitone:
		return h.incrementInventory(ctx, playerID, reward.Kind, reward.RefID, reward.Quantity)
	default:
		return errors.New("unsupported community stand reward kind")
	}
}

func (h *Handler) insertOpenPowerReward(ctx context.Context, playerID string, claimID string, amount int) error {
	_, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        newID("open_power"),
		PlayerID:  playerID,
		Amount:    amount,
		Reason:    "community_stand",
		Source:    claimID,
		CreatedAt: time.Now().UTC(),
	})
	return err
}

func (h *Handler) incrementInventory(ctx context.Context, playerID string, kind string, refID string, quantity int) error {
	collection, field, err := inventoryCollection(kind)
	if err != nil {
		return err
	}
	_, err = h.db.Collection(collection).UpdateOne(
		ctx,
		bson.M{
			"player_id": playerID,
			field:       refID,
		},
		bson.M{
			"$setOnInsert": bson.M{
				"_id":       newID("player_" + kind),
				"player_id": playerID,
				field:       refID,
			},
			"$inc": bson.M{"quantity": quantity},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func inventoryCollection(kind string) (collection string, idField string, err error) {
	switch kind {
	case rewardKindItem:
		return mongomodel.PlayerItemsCollection, "item_id", nil
	case rewardKindSitone:
		return mongomodel.PlayerSitonesCollection, "sitone_id", nil
	default:
		return "", "", errors.New("unsupported reward kind")
	}
}

func (h *Handler) publishRewardGranted(playerID string, reward RewardResponse) {
	if h.broker == nil {
		return
	}
	event := playerevents.RewardGrantedEvent{
		Kind:       reward.Kind,
		RefID:      reward.RefID,
		Name:       reward.Name,
		Quantity:   reward.Quantity,
		Amount:     reward.Amount,
		IconPath:   reward.IconPath,
		Source:     "community_stand",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	switch reward.Kind {
	case rewardKindItem:
		if item, ok := h.content.GetItem(reward.RefID); ok {
			event.ItemType = item.Type
		}
	case rewardKindSitone:
		if sitone, ok := h.content.GetSitone(reward.RefID); ok {
			event.SitoneType = sitone.Type
		}
	}
	h.broker.Publish(playerID, playerevents.Event{
		Name:   "reward_granted",
		Reward: &event,
	})
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
