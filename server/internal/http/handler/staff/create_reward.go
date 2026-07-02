package staff

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	rewardKindItem   = "item"
	rewardKindSitone = "sitone"
)

type rewardDefinition struct {
	kind string
	id   string
	name string
}

// CreateReward godoc
// @Summary Grant sitone or item as staff
// @Description Staff-only endpoint. Grants one sitone or item to a player selected by player ID or QR code identifier, and records the staff grant.
// @Tags staff
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body CreateRewardRequest true "Staff reward request"
// @Success 201 {object} CreateRewardResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/rewards [post]
func (h *Handler) CreateReward(w http.ResponseWriter, r *http.Request) {
	staffPlayer, ok := currentStaff(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	var body CreateRewardRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.QRCodeToken = strings.TrimSpace(body.QRCodeToken)
	body.PlayerID = strings.TrimSpace(body.PlayerID)
	body.Kind = strings.TrimSpace(body.Kind)
	body.RefID = strings.TrimSpace(body.RefID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if body.PlayerID == "" && body.QRCodeToken == "" {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{
				Location: "body.playerId",
				Message:  "playerId or qrcodeToken is required",
			},
		))
		return
	}

	reward, found := h.rewardDefinition(body.Kind, body.RefID)
	if !found {
		httpx.WriteProblem(w, r, httpx.NotFound("reward content not found"))
		return
	}

	recipient, err := h.findRewardRecipient(r.Context(), body)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if body.PlayerID != "" {
			httpx.WriteProblem(w, r, httpx.NotFound("player not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.NotFound("qr code not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward failed", "reward_player_lookup_failed", err))
		return
	}

	team, err := h.findTeam(r.Context(), recipient.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward failed", "reward_team_lookup_failed", err))
		return
	}

	rewardID, err := h.createReward(r.Context(), staffPlayer.ID, recipient.ID, reward, body.Quantity)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("reward failed", "reward_create_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, CreateRewardResponse{
		RewardID: rewardID,
		Player: RewardPlayerResponse{
			PlayerID: recipient.ID,
			Nickname: recipient.Nickname,
			Team: RewardTeamResponse{
				TeamID: team.ID,
				Name:   team.Name,
			},
		},
		Reward: RewardResponse{
			Kind:     reward.kind,
			ID:       reward.id,
			Name:     reward.name,
			Quantity: body.Quantity,
		},
	})
}

func (h *Handler) rewardDefinition(kind string, refID string) (rewardDefinition, bool) {
	switch kind {
	case rewardKindSitone:
		sitone, ok := h.content.GetSitone(refID)
		if !ok {
			return rewardDefinition{}, false
		}
		return rewardDefinition{kind: kind, id: sitone.ID, name: sitone.Name}, true
	case rewardKindItem:
		item, ok := h.content.GetItem(refID)
		if !ok || !item.Enabled {
			return rewardDefinition{}, false
		}
		return rewardDefinition{kind: kind, id: item.ID, name: item.Name}, true
	default:
		return rewardDefinition{}, false
	}
}

func (h *Handler) findRewardRecipient(ctx context.Context, body CreateRewardRequest) (mongomodel.Player, error) {
	if body.PlayerID != "" {
		return h.findPlayerByID(ctx, body.PlayerID)
	}
	return h.findPlayerByQRCodeToken(ctx, body.QRCodeToken)
}

func (h *Handler) findPlayerByID(ctx context.Context, playerID string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(ctx, bson.M{"_id": playerID}).
		Decode(&player)
	return player, err
}

func (h *Handler) findPlayerByQRCodeToken(ctx context.Context, token string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(ctx, bson.M{"qrcode_token": token}).
		Decode(&player)
	return player, err
}

func (h *Handler) findTeam(ctx context.Context, teamID string) (mongomodel.Team, error) {
	var team mongomodel.Team
	err := h.db.Collection(mongomodel.TeamsCollection).
		FindOne(ctx, bson.M{"_id": teamID}).
		Decode(&team)
	return team, err
}

func (h *Handler) createReward(ctx context.Context, staffPlayerID string, recipientPlayerID string, reward rewardDefinition, quantity int) (string, error) {
	rewardID := newID("staff_reward")
	if err := h.incrementInventory(ctx, recipientPlayerID, reward.kind, reward.id, quantity); err != nil {
		return "", err
	}
	if err := h.insertRewardRecord(ctx, rewardID, staffPlayerID, recipientPlayerID, reward, quantity, time.Now().UTC()); err != nil {
		return "", err
	}
	return rewardID, nil
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

func (h *Handler) insertRewardRecord(
	ctx context.Context,
	rewardID string,
	staffPlayerID string,
	recipientPlayerID string,
	reward rewardDefinition,
	quantity int,
	createdAt time.Time,
) error {
	_, err := h.db.Collection(mongomodel.StaffRewardsCollection).InsertOne(ctx, mongomodel.StaffReward{
		ID:                rewardID,
		StaffPlayerID:     staffPlayerID,
		RecipientPlayerID: recipientPlayerID,
		Kind:              reward.kind,
		RefID:             reward.id,
		Quantity:          quantity,
		CreatedAt:         createdAt,
	})
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

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
