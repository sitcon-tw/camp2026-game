package staff

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/roomteam"
)

const (
	rewardKindItem      = "item"
	rewardKindSitone    = "sitone"
	rewardKindOpenPower = "open_power"
)

type rewardDefinition struct {
	kind string
	id   string
	name string
}

// CreateReward godoc
// @Summary Grant sitone, item, or open power as staff
// @Description Staff-only endpoint. Grants one sitone, item, or open power reward to a player selected by player ID or QR code identifier, every player in a team or dorm room, or all players, and records the staff grant.
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
	body.TeamID = strings.TrimSpace(body.TeamID)
	body.RoomNumber = roomteam.NormalizeRoomNumber(body.RoomNumber)
	body.Kind = strings.TrimSpace(body.Kind)
	body.RefID = strings.TrimSpace(body.RefID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if problems := validateRewardTarget(body); len(problems) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid request body", problems...))
		return
	}
	if err := validateRewardBody(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	reward, found := h.rewardDefinition(body.Kind, body.RefID)
	if !found {
		httpx.WriteProblem(w, r, httpx.NotFound("reward content not found"))
		return
	}

	response, status, problem := h.createRewardResponse(r.Context(), staffPlayer, reward, body)
	if problem != nil {
		httpx.WriteProblem(w, r, problem)
		return
	}

	httpx.WriteJSON(w, status, response)
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
	case rewardKindOpenPower:
		return rewardDefinition{kind: kind, id: rewardKindOpenPower, name: "開源力"}, true
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

func (h *Handler) findPlayersByTeamID(ctx context.Context, teamID string) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": teamID},
		options.Find().
			SetProjection(bson.D{
				{Key: "auth_token", Value: 0},
				{Key: "qrcode_token", Value: 0},
				{Key: "default_sitone_ids", Value: 0},
			}).
			SetSort(bson.D{
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	if players == nil {
		return []mongomodel.Player{}, nil
	}
	return players, nil
}

func (h *Handler) findPlayersByRoomNumber(ctx context.Context, roomNumber string) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.RoomTeamMembershipsCollection).Find(
		ctx,
		bson.M{"room_team_id": roomteam.RoomID(roomNumber)},
		options.Find().SetProjection(bson.D{{Key: "player_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var memberships []mongomodel.RoomTeamMembership
	if err := cursor.All(ctx, &memberships); err != nil {
		return nil, err
	}
	playerIDs := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		if membership.PlayerID != "" {
			playerIDs = append(playerIDs, membership.PlayerID)
		}
	}
	if len(playerIDs) == 0 {
		return []mongomodel.Player{}, nil
	}

	cursor, err = h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"_id": bson.M{"$in": playerIDs}},
		options.Find().
			SetProjection(bson.D{
				{Key: "auth_token", Value: 0},
				{Key: "qrcode_token", Value: 0},
				{Key: "default_sitone_ids", Value: 0},
			}).
			SetSort(bson.D{{Key: "nickname", Value: 1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	if players == nil {
		return []mongomodel.Player{}, nil
	}
	return players, nil
}

func (h *Handler) findAllPlayers(ctx context.Context) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{},
		options.Find().
			SetProjection(bson.D{
				{Key: "auth_token", Value: 0},
				{Key: "qrcode_token", Value: 0},
				{Key: "default_sitone_ids", Value: 0},
			}).
			SetSort(bson.D{
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	if players == nil {
		return []mongomodel.Player{}, nil
	}
	return players, nil
}

func (h *Handler) createReward(ctx context.Context, staffPlayerID string, recipientPlayerID string, reward rewardDefinition, quantity int) (mongomodel.StaffReward, error) {
	return h.createRewardWithID(ctx, newID("staff_reward"), staffPlayerID, recipientPlayerID, reward, quantity, time.Now().UTC())
}

func (h *Handler) createRewardWithID(ctx context.Context, rewardID string, staffPlayerID string, recipientPlayerID string, reward rewardDefinition, quantity int, createdAt time.Time) (mongomodel.StaffReward, error) {
	record := mongomodel.StaffReward{
		ID:                  rewardID,
		StaffPlayerID:       staffPlayerID,
		RecipientPlayerID:   recipientPlayerID,
		Kind:                reward.kind,
		RefID:               reward.id,
		Quantity:            quantity,
		CreatedAt:           createdAt,
		NotificationPending: true,
	}
	switch reward.kind {
	case rewardKindOpenPower:
		if err := h.insertOpenPowerReward(ctx, record.ID, recipientPlayerID, quantity, record.CreatedAt); err != nil {
			return mongomodel.StaffReward{}, err
		}
	default:
		if err := h.incrementInventory(ctx, recipientPlayerID, reward.kind, reward.id, quantity); err != nil {
			return mongomodel.StaffReward{}, err
		}
	}
	if err := h.insertRewardRecord(ctx, record); err != nil {
		return mongomodel.StaffReward{}, err
	}
	return record, nil
}

func (h *Handler) insertOpenPowerReward(ctx context.Context, rewardID string, playerID string, amount int, createdAt time.Time) error {
	_, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        newID("open_power"),
		PlayerID:  playerID,
		Amount:    amount,
		Reason:    "staff_reward",
		Source:    rewardID,
		CreatedAt: createdAt,
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

func (h *Handler) insertRewardRecord(ctx context.Context, reward mongomodel.StaffReward) error {
	_, err := h.db.Collection(mongomodel.StaffRewardsCollection).InsertOne(ctx, reward)
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

func validateRewardTarget(body CreateRewardRequest) []httpx.ErrorDetail {
	switch {
	case body.AllPlayers && (body.TeamID != "" || body.RoomNumber != "" || body.PlayerID != "" || body.QRCodeToken != ""):
		return []httpx.ErrorDetail{{
			Location: "body.allPlayers",
			Message:  "allPlayers cannot be combined with playerId, qrcodeToken, teamId, or roomNumber",
		}}
	case body.TeamID != "" && (body.RoomNumber != "" || body.PlayerID != "" || body.QRCodeToken != ""):
		return []httpx.ErrorDetail{{
			Location: "body.teamId",
			Message:  "teamId cannot be combined with playerId, qrcodeToken, or roomNumber",
		}}
	case body.RoomNumber != "" && (body.PlayerID != "" || body.QRCodeToken != ""):
		return []httpx.ErrorDetail{{
			Location: "body.roomNumber",
			Message:  "roomNumber cannot be combined with playerId or qrcodeToken",
		}}
	case body.RoomNumber != "" && !roomteam.ValidRoomNumber(body.RoomNumber):
		return []httpx.ErrorDetail{{
			Location: "body.roomNumber",
			Message:  "roomNumber must be one of the configured dorm rooms",
		}}
	case !body.AllPlayers && body.TeamID == "" && body.RoomNumber == "" && body.PlayerID == "" && body.QRCodeToken == "":
		return []httpx.ErrorDetail{{
			Location: "body.playerId",
			Message:  "playerId, qrcodeToken, teamId, roomNumber, or allPlayers is required",
		}}
	default:
		return nil
	}
}

func validateRewardBody(body CreateRewardRequest) error {
	switch body.Kind {
	case rewardKindOpenPower:
		if body.Amount <= 0 {
			return httpx.UnprocessableEntity(
				"invalid request body",
				httpx.ErrorDetail{Location: "body.amount", Message: "amount is required for open_power rewards"},
			)
		}
		return nil
	case rewardKindItem, rewardKindSitone:
		if body.RefID == "" {
			return httpx.UnprocessableEntity(
				"invalid request body",
				httpx.ErrorDetail{Location: "body.refId", Message: "refId is required for item or sitone rewards"},
			)
		}
		if body.Quantity <= 0 {
			return httpx.UnprocessableEntity(
				"invalid request body",
				httpx.ErrorDetail{Location: "body.quantity", Message: "quantity is required for item or sitone rewards"},
			)
		}
		return nil
	default:
		return nil
	}
}

func (h *Handler) createRewardResponse(
	ctx context.Context,
	staffPlayer mongomodel.Player,
	reward rewardDefinition,
	body CreateRewardRequest,
) (CreateRewardResponse, int, error) {
	if body.AllPlayers {
		return h.createAllPlayersRewardResponse(ctx, staffPlayer, reward, body)
	}
	if body.TeamID != "" {
		return h.createTeamRewardResponse(ctx, staffPlayer, reward, body)
	}
	if body.RoomNumber != "" {
		return h.createRoomRewardResponse(ctx, staffPlayer, reward, body)
	}
	return h.createPlayerRewardResponse(ctx, staffPlayer, reward, body)
}

func (h *Handler) createRoomRewardResponse(
	ctx context.Context,
	staffPlayer mongomodel.Player,
	reward rewardDefinition,
	body CreateRewardRequest,
) (CreateRewardResponse, int, error) {
	recipients, err := h.findPlayersByRoomNumber(ctx, body.RoomNumber)
	if err != nil {
		return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_room_player_lookup_failed", err)
	}
	if len(recipients) == 0 {
		return CreateRewardResponse{}, 0, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{Location: "body.roomNumber", Message: "dorm room has no players"},
		)
	}

	return h.createMultiPlayerRewardResponse(ctx, staffPlayer, reward, body, recipients, CreateRewardResponse{
		Room: &RewardRoomResponse{RoomNumber: body.RoomNumber},
	}, "body.roomNumber", "dorm room has no players")
}

func (h *Handler) createAllPlayersRewardResponse(
	ctx context.Context,
	staffPlayer mongomodel.Player,
	reward rewardDefinition,
	body CreateRewardRequest,
) (CreateRewardResponse, int, error) {
	recipients, err := h.findAllPlayers(ctx)
	if err != nil {
		return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_player_lookup_failed", err)
	}
	if len(recipients) == 0 {
		return CreateRewardResponse{}, 0, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{
				Location: "body.allPlayers",
				Message:  "no players found",
			},
		)
	}

	return h.createMultiPlayerRewardResponse(ctx, staffPlayer, reward, body, recipients, CreateRewardResponse{AllPlayers: true}, "body.allPlayers", "no players found")
}

func (h *Handler) createPlayerRewardResponse(
	ctx context.Context,
	staffPlayer mongomodel.Player,
	reward rewardDefinition,
	body CreateRewardRequest,
) (CreateRewardResponse, int, error) {
	recipient, err := h.findRewardRecipient(ctx, body)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if body.PlayerID != "" {
			return CreateRewardResponse{}, 0, httpx.NotFound("player not found")
		}
		return CreateRewardResponse{}, 0, httpx.NotFound("qr code not found")
	}
	if err != nil {
		return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_player_lookup_failed", err)
	}

	var team *RewardTeamResponse
	if recipient.TeamID != "" {
		loadedTeam, err := h.findTeam(ctx, recipient.TeamID)
		if err != nil {
			return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_team_lookup_failed", err)
		}
		team = &RewardTeamResponse{TeamID: loadedTeam.ID, Name: loadedTeam.Name}
	}

	rewardValue := body.Quantity
	if body.Kind == rewardKindOpenPower {
		rewardValue = body.Amount
	}
	rewardRecord, err := h.createReward(ctx, staffPlayer.ID, recipient.ID, reward, rewardValue)
	if err != nil {
		return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_create_failed", err)
	}
	h.publishRewardGranted(ctx, recipient.ID, staffPlayer, rewardRecord)

	return CreateRewardResponse{
		RewardIDs:    []string{rewardRecord.ID},
		GrantedCount: 1,
		Player: &RewardPlayerResponse{
			PlayerID:  recipient.ID,
			Nickname:  recipient.Nickname,
			AvatarURL: recipient.AvatarURL,
			Team:      team,
		},
		Reward: RewardResponse{
			Kind:     reward.kind,
			ID:       reward.id,
			Name:     reward.name,
			Quantity: body.Quantity,
			Amount:   body.Amount,
		},
	}, http.StatusCreated, nil
}

func (h *Handler) createTeamRewardResponse(
	ctx context.Context,
	staffPlayer mongomodel.Player,
	reward rewardDefinition,
	body CreateRewardRequest,
) (CreateRewardResponse, int, error) {
	team, err := h.findTeam(ctx, body.TeamID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return CreateRewardResponse{}, 0, httpx.NotFound("team not found")
	}
	if err != nil {
		return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_team_lookup_failed", err)
	}

	recipients, err := h.findPlayersByTeamID(ctx, team.ID)
	if err != nil {
		return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_player_lookup_failed", err)
	}
	if len(recipients) == 0 {
		return CreateRewardResponse{}, 0, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{
				Location: "body.teamId",
				Message:  "team has no players",
			},
		)
	}

	return h.createMultiPlayerRewardResponse(ctx, staffPlayer, reward, body, recipients, CreateRewardResponse{
		Team: &RewardTeamResponse{
			TeamID: team.ID,
			Name:   team.Name,
		},
	}, "body.teamId", "team has no players")
}

func (h *Handler) createMultiPlayerRewardResponse(
	ctx context.Context,
	staffPlayer mongomodel.Player,
	reward rewardDefinition,
	body CreateRewardRequest,
	recipients []mongomodel.Player,
	response CreateRewardResponse,
	emptyLocation string,
	emptyMessage string,
) (CreateRewardResponse, int, error) {
	rewardIDs := make([]string, 0, len(recipients))
	rewardValue := body.Quantity
	if body.Kind == rewardKindOpenPower {
		rewardValue = body.Amount
	}
	for _, recipient := range recipients {
		if recipient.ID == "" {
			continue
		}
		rewardRecord, err := h.createReward(ctx, staffPlayer.ID, recipient.ID, reward, rewardValue)
		if err != nil {
			return CreateRewardResponse{}, 0, httpx.InternalServerError("reward failed", "reward_create_failed", err)
		}
		rewardIDs = append(rewardIDs, rewardRecord.ID)
		h.publishRewardGranted(ctx, recipient.ID, staffPlayer, rewardRecord)
	}
	if len(rewardIDs) == 0 {
		return CreateRewardResponse{}, 0, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{
				Location: emptyLocation,
				Message:  emptyMessage,
			},
		)
	}
	slices.Sort(rewardIDs)

	response.RewardIDs = rewardIDs
	response.GrantedCount = len(rewardIDs)
	response.Reward = RewardResponse{
		Kind:     reward.kind,
		ID:       reward.id,
		Name:     reward.name,
		Quantity: body.Quantity,
		Amount:   body.Amount,
	}
	return response, http.StatusCreated, nil
}

func (h *Handler) publishRewardGranted(ctx context.Context, playerID string, staffPlayer mongomodel.Player, reward mongomodel.StaffReward) {
	if h.broker == nil {
		return
	}
	event, err := playerevents.StaffRewardGrantedEvent(h.content, reward, staffPlayer, false)
	if err != nil {
		return
	}
	delivered := h.broker.Publish(playerID, playerevents.Event{
		Name:   "reward_granted",
		Reward: &event,
	})
	if delivered > 0 {
		_ = h.markRewardNotified(ctx, reward.ID, time.Now().UTC())
	}
}

func (h *Handler) markRewardNotified(ctx context.Context, rewardID string, notifiedAt time.Time) error {
	_, err := h.db.Collection(mongomodel.StaffRewardsCollection).UpdateOne(
		ctx,
		bson.M{"_id": rewardID, "notification_pending": true},
		bson.M{
			"$set":   bson.M{"notified_at": notifiedAt},
			"$unset": bson.M{"notification_pending": ""},
		},
	)
	return err
}
