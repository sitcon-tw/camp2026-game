package admin

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	defaultBalanceUpdateMessage = "營運平衡已更新你的小石與開源力數量。"
	balanceFallbackSitoneID     = "stone_explorer_base"
	inventoryTrimMaxTop         = 100
)

func defaultInventoryTrimMessage(openPower int) string {
	return "小石看著自己的 AI server ，感覺記憶體不太夠，於是帶著" + strconv.Itoa(openPower) + "開源力去排隊購買記憶體了...應該很快就會回來"
}

type CreateInventoryTrimRequest struct {
	Top         int    `json:"top" validate:"required,min=1,max=100" example:"3"`
	SitoneCount int    `json:"sitoneCount,omitempty" validate:"omitempty,min=0,max=9999" example:"2"`
	OpenPower   int    `json:"openPower,omitempty" validate:"omitempty,min=0,max=999999" example:"500"`
	Message     string `json:"message,omitempty" validate:"omitempty,max=240"`
	DryRun      bool   `json:"dryRun,omitempty"`
}

type CreateInventoryTrimResponse struct {
	DryRun        bool                          `json:"dryRun"`
	Message       string                        `json:"message"`
	RequestedTop  int                           `json:"requestedTop"`
	AffectedCount int                           `json:"affectedCount"`
	Players       []InventoryTrimPlayerResponse `json:"players"`
}

type InventoryTrimPlayerResponse struct {
	TrimID           string `json:"trimId,omitempty"`
	Rank             int    `json:"rank" example:"1"`
	PlayerID         string `json:"playerId" example:"7H9K2Q"`
	Nickname         string `json:"nickname" example:"Alice"`
	TeamID           string `json:"teamId,omitempty" example:"8M4RXP"`
	SitoneBefore     int    `json:"sitoneBefore" example:"18"`
	SitoneTrimmed    int    `json:"sitoneTrimmed" example:"2"`
	SitoneAfter      int    `json:"sitoneAfter" example:"16"`
	OpenPowerBefore  int    `json:"openPowerBefore" example:"1200"`
	OpenPowerTrimmed int    `json:"openPowerTrimmed" example:"500"`
	OpenPowerAfter   int    `json:"openPowerAfter" example:"700"`
}

type UpdatePlayerBalanceRequest struct {
	SitoneCount int    `json:"sitoneCount" validate:"min=0,max=99999" example:"12"`
	OpenPower   int    `json:"openPower" validate:"min=0,max=9999999" example:"2400"`
	Message     string `json:"message,omitempty" validate:"omitempty,max=240"`
}

type inventoryTrimCandidate struct {
	Rank        int
	PlayerID    string
	Nickname    string
	TeamID      string
	SitoneCount int
	OpenPower   int
}

type playerBalance struct {
	SitoneCount int
	OpenPower   int
}

// CreateInventoryTrim godoc
// @Summary Trim top player inventory
// @Description Admin-only endpoint. Selects the top N leaderboard players, removes sitones and open power, then notifies affected players.
// @Tags admin
// @Accept json
// @Produce json
// @Param request body CreateInventoryTrimRequest true "Inventory trim request"
// @Success 200 {object} CreateInventoryTrimResponse
// @Success 201 {object} CreateInventoryTrimResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/inventory-trims [post]
func (h *Handler) CreateInventoryTrim(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	var body CreateInventoryTrimRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		body.Message = defaultInventoryTrimMessage(body.OpenPower)
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if body.SitoneCount == 0 && body.OpenPower == 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{Location: "body.sitoneCount", Message: "sitoneCount or openPower must be greater than 0"},
		))
		return
	}
	if body.Top > inventoryTrimMaxTop {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(
			"invalid request body",
			httpx.ErrorDetail{Location: "body.top", Message: "top must be at most 100"},
		))
		return
	}

	response, err := h.createInventoryTrim(r.Context(), body)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("inventory trim failed", "admin_inventory_trim_failed", err))
		return
	}
	status := http.StatusCreated
	if body.DryRun {
		status = http.StatusOK
	}
	httpx.WriteJSON(w, status, response)
}

// UpdatePlayerBalance godoc
// @Summary Update one player's balance totals
// @Description Admin-only endpoint. Sets one player's total sitone count and open power to target values, then notifies the player.
// @Tags admin
// @Accept json
// @Produce json
// @Param playerID path string true "Player ID"
// @Param request body UpdatePlayerBalanceRequest true "Player balance request"
// @Success 200 {object} InventoryTrimPlayerResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/players/{playerID}/balance [put]
func (h *Handler) UpdatePlayerBalance(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	playerID := strings.TrimSpace(chi.URLParam(r, "playerID"))
	if playerID == "" {
		httpx.WriteProblem(w, r, httpx.NotFound("player not found"))
		return
	}

	var body UpdatePlayerBalanceRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.Message = strings.TrimSpace(body.Message)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	response, err := h.updatePlayerBalance(r.Context(), playerID, body)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("player not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("player balance update failed", "admin_player_balance_update_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) createInventoryTrim(ctx context.Context, body CreateInventoryTrimRequest) (CreateInventoryTrimResponse, error) {
	candidates, err := h.inventoryTrimCandidates(ctx, body.Top)
	if err != nil {
		return CreateInventoryTrimResponse{}, err
	}

	now := time.Now().UTC()
	players := make([]InventoryTrimPlayerResponse, 0, len(candidates))
	affectedCount := 0
	for _, candidate := range candidates {
		sitoneTrimmed := min(body.SitoneCount, max(0, candidate.SitoneCount))
		openPowerTrimmed := min(body.OpenPower, max(0, candidate.OpenPower))
		response := InventoryTrimPlayerResponse{
			Rank:             candidate.Rank,
			PlayerID:         candidate.PlayerID,
			Nickname:         candidate.Nickname,
			TeamID:           candidate.TeamID,
			SitoneBefore:     candidate.SitoneCount,
			SitoneTrimmed:    sitoneTrimmed,
			SitoneAfter:      candidate.SitoneCount - sitoneTrimmed,
			OpenPowerBefore:  candidate.OpenPower,
			OpenPowerTrimmed: openPowerTrimmed,
			OpenPowerAfter:   candidate.OpenPower - openPowerTrimmed,
		}
		if sitoneTrimmed > 0 || openPowerTrimmed > 0 {
			affectedCount++
		}
		if !body.DryRun && (sitoneTrimmed > 0 || openPowerTrimmed > 0) {
			trim, err := h.applyInventoryTrim(ctx, candidate.PlayerID, sitoneTrimmed, openPowerTrimmed, body.Message, now)
			if err != nil {
				return CreateInventoryTrimResponse{}, err
			}
			response.TrimID = trim.ID
			h.publishInventoryTrimmed(ctx, trim)
		}
		players = append(players, response)
	}

	return CreateInventoryTrimResponse{
		DryRun:        body.DryRun,
		Message:       body.Message,
		RequestedTop:  body.Top,
		AffectedCount: affectedCount,
		Players:       players,
	}, nil
}

func (h *Handler) inventoryTrimCandidates(ctx context.Context, top int) ([]inventoryTrimCandidate, error) {
	raw, err := h.dashboardRawData(ctx)
	if err != nil {
		return nil, err
	}
	response := buildDashboardResponse(time.Now().UTC(), h.content, raw)
	players := make([]DashboardPlayerResponse, 0, len(response.Players))
	for _, player := range response.Players {
		if player.PlayerID == "" || player.Team == nil || player.Role == "staff" {
			continue
		}
		players = append(players, player)
	}
	sortPlayersBySitones(players)
	if len(players) > top {
		players = players[:top]
	}

	candidates := make([]inventoryTrimCandidate, 0, len(players))
	for index, player := range players {
		teamID := ""
		if player.Team != nil {
			teamID = player.Team.TeamID
		}
		candidates = append(candidates, inventoryTrimCandidate{
			Rank:        index + 1,
			PlayerID:    player.PlayerID,
			Nickname:    player.Nickname,
			TeamID:      teamID,
			SitoneCount: player.SitoneCount,
			OpenPower:   player.OpenPower,
		})
	}
	return candidates, nil
}

func (h *Handler) updatePlayerBalance(ctx context.Context, playerID string, body UpdatePlayerBalanceRequest) (InventoryTrimPlayerResponse, error) {
	player, err := h.findBalancePlayer(ctx, playerID)
	if err != nil {
		return InventoryTrimPlayerResponse{}, err
	}
	before, err := h.currentPlayerBalance(ctx, playerID)
	if err != nil {
		return InventoryTrimPlayerResponse{}, err
	}

	sitoneDelta := body.SitoneCount - before.SitoneCount
	openPowerDelta := body.OpenPower - before.OpenPower
	if sitoneDelta < 0 {
		if _, err := h.trimPlayerSitones(ctx, playerID, -sitoneDelta); err != nil {
			return InventoryTrimPlayerResponse{}, err
		}
	} else if sitoneDelta > 0 {
		if err := h.addPlayerSitones(ctx, playerID, sitoneDelta); err != nil {
			return InventoryTrimPlayerResponse{}, err
		}
	}

	now := time.Now().UTC()
	if openPowerDelta != 0 {
		if err := h.insertOpenPowerAdjustment(ctx, playerID, openPowerDelta, now); err != nil {
			return InventoryTrimPlayerResponse{}, err
		}
	}

	message := body.Message
	sitoneTrimmed := max(0, -sitoneDelta)
	openPowerTrimmed := max(0, -openPowerDelta)
	if message == "" {
		if sitoneTrimmed > 0 || openPowerTrimmed > 0 {
			message = defaultInventoryTrimMessage(openPowerTrimmed)
		} else {
			message = defaultBalanceUpdateMessage
		}
	}

	response := InventoryTrimPlayerResponse{
		Rank:             0,
		PlayerID:         player.ID,
		Nickname:         player.Nickname,
		TeamID:           player.TeamID,
		SitoneBefore:     before.SitoneCount,
		SitoneTrimmed:    sitoneTrimmed,
		SitoneAfter:      body.SitoneCount,
		OpenPowerBefore:  before.OpenPower,
		OpenPowerTrimmed: openPowerTrimmed,
		OpenPowerAfter:   body.OpenPower,
	}

	if sitoneDelta != 0 || openPowerDelta != 0 {
		trim, err := h.createBalanceNotification(ctx, playerID, sitoneTrimmed, openPowerTrimmed, message, now)
		if err != nil {
			return InventoryTrimPlayerResponse{}, err
		}
		response.TrimID = trim.ID
		h.publishInventoryTrimmed(ctx, trim)
	}
	return response, nil
}

func (h *Handler) findBalancePlayer(ctx context.Context, playerID string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(ctx, bson.M{"_id": playerID}).
		Decode(&player)
	return player, err
}

func (h *Handler) currentPlayerBalance(ctx context.Context, playerID string) (playerBalance, error) {
	sitoneCount, err := h.playerSitoneTotal(ctx, playerID)
	if err != nil {
		return playerBalance{}, err
	}
	openPower, err := h.playerOpenPowerTotal(ctx, playerID)
	if err != nil {
		return playerBalance{}, err
	}
	return playerBalance{SitoneCount: sitoneCount, OpenPower: openPower}, nil
}

func (h *Handler) playerSitoneTotal(ctx context.Context, playerID string) (int, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: playerID},
			{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	var rows []struct {
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Total, nil
}

func (h *Handler) playerOpenPowerTotal(ctx context.Context, playerID string) (int, error) {
	cursor, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: playerID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	var rows []struct {
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Total, nil
}

func (h *Handler) applyInventoryTrim(ctx context.Context, playerID string, sitoneCount int, openPower int, message string, now time.Time) (mongomodel.InventoryTrim, error) {
	if sitoneCount > 0 {
		trimmed, err := h.trimPlayerSitones(ctx, playerID, sitoneCount)
		if err != nil {
			return mongomodel.InventoryTrim{}, err
		}
		sitoneCount = trimmed
	}
	if openPower > 0 {
		if err := h.insertOpenPowerTrim(ctx, playerID, openPower, now); err != nil {
			return mongomodel.InventoryTrim{}, err
		}
	}

	record, err := h.createBalanceNotification(ctx, playerID, sitoneCount, openPower, message, now)
	if err != nil {
		return mongomodel.InventoryTrim{}, err
	}
	return record, nil
}

func (h *Handler) createBalanceNotification(ctx context.Context, playerID string, sitoneCount int, openPower int, message string, now time.Time) (mongomodel.InventoryTrim, error) {
	record := mongomodel.InventoryTrim{
		ID:                  "inventory_trim_" + bson.NewObjectID().Hex(),
		PlayerID:            playerID,
		SitoneCount:         sitoneCount,
		OpenPower:           openPower,
		Message:             message,
		CreatedAt:           now,
		NotificationPending: true,
	}
	if _, err := h.db.Collection(mongomodel.InventoryTrimsCollection).InsertOne(ctx, record); err != nil {
		return mongomodel.InventoryTrim{}, err
	}
	return record, nil
}

func (h *Handler) addPlayerSitones(ctx context.Context, playerID string, quantity int) error {
	sitoneID, err := h.balanceSitoneID(ctx, playerID)
	if err != nil {
		return err
	}
	_, err = h.db.Collection(mongomodel.PlayerSitonesCollection).UpdateOne(
		ctx,
		bson.M{"player_id": playerID, "sitone_id": sitoneID},
		bson.M{
			"$setOnInsert": bson.M{
				"_id":       "player_sitone_" + bson.NewObjectID().Hex(),
				"player_id": playerID,
				"sitone_id": sitoneID,
			},
			"$inc": bson.M{"quantity": quantity},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (h *Handler) balanceSitoneID(ctx context.Context, playerID string) (string, error) {
	var sitone mongomodel.PlayerSitone
	err := h.db.Collection(mongomodel.PlayerSitonesCollection).FindOne(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.FindOne().SetSort(bson.D{{Key: "quantity", Value: -1}, {Key: "sitone_id", Value: 1}}),
	).Decode(&sitone)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return balanceFallbackSitoneID, nil
	}
	if err != nil {
		return "", err
	}
	if sitone.SitoneID == "" {
		return balanceFallbackSitoneID, nil
	}
	return sitone.SitoneID, nil
}

func (h *Handler) trimPlayerSitones(ctx context.Context, playerID string, target int) (int, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "quantity", Value: -1}, {Key: "sitone_id", Value: 1}}),
	)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var sitones []mongomodel.PlayerSitone
	if err := cursor.All(ctx, &sitones); err != nil {
		return 0, err
	}
	h.sortPlayerSitonesForBalanceTrim(sitones)

	remaining := target
	trimmed := 0
	for _, sitone := range sitones {
		if remaining <= 0 {
			break
		}
		delta := min(remaining, sitone.Quantity)
		if delta <= 0 {
			continue
		}
		result, err := h.db.Collection(mongomodel.PlayerSitonesCollection).UpdateOne(
			ctx,
			bson.M{"_id": sitone.ID, "quantity": bson.M{"$gte": delta}},
			bson.M{"$inc": bson.M{"quantity": -delta}},
		)
		if err != nil {
			return trimmed, err
		}
		if result.ModifiedCount == 0 {
			return trimmed, errors.New("player sitone quantity changed during trim")
		}
		remaining -= delta
		trimmed += delta
	}
	return trimmed, nil
}

func (h *Handler) sortPlayerSitonesForBalanceTrim(sitones []mongomodel.PlayerSitone) {
	sort.SliceStable(sitones, func(i, j int) bool {
		if sitones[i].Quantity != sitones[j].Quantity {
			return sitones[i].Quantity > sitones[j].Quantity
		}
		iBase := h.isBaseSitone(sitones[i].SitoneID)
		jBase := h.isBaseSitone(sitones[j].SitoneID)
		if iBase != jBase {
			return iBase
		}
		return sitones[i].SitoneID < sitones[j].SitoneID
	})
}

func (h *Handler) isBaseSitone(sitoneID string) bool {
	if h.content == nil {
		return false
	}
	sitone, ok := h.content.GetSitone(sitoneID)
	return ok && sitone.Rarity == "base"
}

func (h *Handler) insertOpenPowerTrim(ctx context.Context, playerID string, openPower int, now time.Time) error {
	return h.insertOpenPowerAdjustment(ctx, playerID, -openPower, now)
}

func (h *Handler) insertOpenPowerAdjustment(ctx context.Context, playerID string, amount int, now time.Time) error {
	_, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        "open_power_" + bson.NewObjectID().Hex(),
		PlayerID:  playerID,
		Amount:    amount,
		Reason:    "inventory_trim",
		Source:    "admin_inventory_trim",
		CreatedAt: now,
	})
	return err
}

func (h *Handler) publishInventoryTrimmed(ctx context.Context, trim mongomodel.InventoryTrim) {
	if h.broker == nil {
		return
	}
	event := playerevents.InventoryTrimmed(trim, false)
	delivered := h.broker.Publish(trim.PlayerID, playerevents.Event{
		Name:             playerevents.InventoryTrimmedEventName,
		InventoryTrimmed: &event,
	})
	if delivered > 0 {
		_ = h.markInventoryTrimNotified(ctx, trim.ID, time.Now().UTC())
	}
}

func (h *Handler) markInventoryTrimNotified(ctx context.Context, trimID string, notifiedAt time.Time) error {
	_, err := h.db.Collection(mongomodel.InventoryTrimsCollection).UpdateOne(
		ctx,
		bson.M{"_id": trimID, "notification_pending": true},
		bson.M{
			"$set":   bson.M{"notified_at": notifiedAt},
			"$unset": bson.M{"notification_pending": ""},
		},
	)
	return err
}
