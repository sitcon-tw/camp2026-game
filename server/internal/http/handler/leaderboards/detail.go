package leaderboards

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var errLeaderboardNotFound = errors.New("leaderboard resource not found")

// TeamPlayers godoc
// @Summary List leaderboard team players
// @Description Lists players in a ranked team with their inventory totals.
// @Tags leaderboards
// @Produce json
// @Security AuthCookieAuth
// @Param teamID path string true "Team ID"
// @Success 200 {object} TeamPlayersResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /leaderboards/teams/{teamID}/players [get]
func (h *Handler) TeamPlayers(w http.ResponseWriter, r *http.Request) {
	currentPlayer, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	teamID := strings.TrimSpace(chi.URLParam(r, "teamID"))
	team, err := h.findTeamByID(r.Context(), teamID)
	if errors.Is(err, errLeaderboardNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("team not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard team unavailable", "leaderboard_team_lookup_failed", err))
		return
	}

	players, err := h.findTeamPlayers(r.Context(), team.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard team unavailable", "leaderboard_team_players_lookup_failed", err))
		return
	}
	stats, err := h.playerStats(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard team unavailable", "leaderboard_team_stats_lookup_failed", err))
		return
	}
	itemCounts, err := h.playerItemCounts(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard team unavailable", "leaderboard_team_items_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, TeamPlayersResponse{
		Team:    teamSummary(team),
		Players: teamPlayerSummaries(players, stats, itemCounts, currentPlayer.ID),
	})
}

// PlayerInventory godoc
// @Summary Get leaderboard player inventory
// @Description Returns read-only item and sitone inventory for a leaderboard player.
// @Tags leaderboards
// @Produce json
// @Security AuthCookieAuth
// @Param playerID path string true "Player ID"
// @Success 200 {object} PlayerInventoryResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /leaderboards/players/{playerID}/inventory [get]
func (h *Handler) PlayerInventory(w http.ResponseWriter, r *http.Request) {
	currentPlayer, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	playerID := strings.TrimSpace(chi.URLParam(r, "playerID"))
	player, err := h.findLeaderboardPlayerByID(r.Context(), playerID)
	if errors.Is(err, errLeaderboardNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("player not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard player unavailable", "leaderboard_player_lookup_failed", err))
		return
	}

	team, err := h.findTeamByID(r.Context(), player.TeamID)
	if errors.Is(err, errLeaderboardNotFound) {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard player unavailable", "leaderboard_player_team_lookup_failed", err))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard player unavailable", "leaderboard_player_team_lookup_failed", err))
		return
	}

	itemRecords, err := h.findPlayerItems(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard player inventory unavailable", "leaderboard_player_items_lookup_failed", err))
		return
	}
	sitoneRecords, err := h.findPlayerSitones(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard player inventory unavailable", "leaderboard_player_sitones_lookup_failed", err))
		return
	}
	stats, err := h.playerStats(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard player inventory unavailable", "leaderboard_player_stats_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, PlayerInventoryResponse{
		Player: InventoryPlayerResponse{
			PlayerID:    player.ID,
			Nickname:    player.Nickname,
			AvatarURL:   player.AvatarURL,
			SitoneCount: quantityTotalSitones(sitoneRecords),
			ItemCount:   quantityTotalItems(itemRecords),
			OpenPower:   stats[player.ID].OpenPower,
			Current:     player.ID == currentPlayer.ID,
		},
		Team:    teamSummary(team),
		Items:   inventoryItemResponses(h.content, itemRecords),
		Sitones: inventorySitoneResponses(h.content, sitoneRecords),
	})
}

func (h *Handler) findTeamByID(ctx context.Context, teamID string) (mongomodel.Team, error) {
	if teamID == "" {
		return mongomodel.Team{}, errLeaderboardNotFound
	}
	var team mongomodel.Team
	err := h.db.Collection(mongomodel.TeamsCollection).FindOne(ctx, bson.M{"_id": teamID}).Decode(&team)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return mongomodel.Team{}, errLeaderboardNotFound
	}
	if err != nil {
		return mongomodel.Team{}, err
	}
	if team.ID == "" {
		return mongomodel.Team{}, errLeaderboardNotFound
	}
	return team, nil
}

func (h *Handler) findTeamPlayers(ctx context.Context, teamID string) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": teamID},
		options.Find().
			SetProjection(nonSensitivePlayerProjection()).
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

func (h *Handler) findLeaderboardPlayerByID(ctx context.Context, playerID string) (mongomodel.Player, error) {
	if playerID == "" {
		return mongomodel.Player{}, errLeaderboardNotFound
	}
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).FindOne(
		ctx,
		bson.M{
			"_id":     playerID,
			"team_id": bson.M{"$exists": true, "$ne": ""},
		},
		options.FindOne().SetProjection(nonSensitivePlayerProjection()),
	).Decode(&player)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return mongomodel.Player{}, errLeaderboardNotFound
	}
	if err != nil {
		return mongomodel.Player{}, err
	}
	if !isLeaderboardPlayer(player) {
		return mongomodel.Player{}, errLeaderboardNotFound
	}
	return player, nil
}

func nonSensitivePlayerProjection() bson.D {
	return bson.D{
		{Key: "auth_token", Value: 0},
		{Key: "qrcode_token", Value: 0},
		{Key: "default_sitone_ids", Value: 0},
		{Key: "telegram_user_id", Value: 0},
		{Key: "telegram_username", Value: 0},
		{Key: "telegram_chat_id", Value: 0},
	}
}

func (h *Handler) playerItemCounts(ctx context.Context) (map[string]int, error) {
	return h.scoreMap(ctx, mongomodel.PlayerItemsCollection, playerItemCountsPipeline())
}

func playerItemCountsPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	}
}

func (h *Handler) findPlayerItems(ctx context.Context, playerID string) ([]mongomodel.PlayerItem, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerItemsCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "item_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.PlayerItem
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if records == nil {
		return []mongomodel.PlayerItem{}, nil
	}
	return records, nil
}

func (h *Handler) findPlayerSitones(ctx context.Context, playerID string) ([]mongomodel.PlayerSitone, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "sitone_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.PlayerSitone
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if records == nil {
		return []mongomodel.PlayerSitone{}, nil
	}
	return records, nil
}

func teamSummary(team mongomodel.Team) TeamSummaryResponse {
	return TeamSummaryResponse{
		TeamID: team.ID,
		Name:   team.Name,
	}
}

func teamPlayerSummaries(
	players []mongomodel.Player,
	stats map[string]rankStats,
	itemCounts map[string]int,
	currentPlayerID string,
) []TeamPlayerSummaryResponse {
	responses := make([]TeamPlayerSummaryResponse, 0, len(players))
	for _, player := range players {
		if !isLeaderboardPlayer(player) {
			continue
		}
		playerStats := stats[player.ID]
		responses = append(responses, TeamPlayerSummaryResponse{
			PlayerID:    player.ID,
			Nickname:    player.Nickname,
			AvatarURL:   player.AvatarURL,
			SitoneCount: playerStats.SitoneCount,
			ItemCount:   itemCounts[player.ID],
			OpenPower:   playerStats.OpenPower,
			Current:     player.ID == currentPlayerID,
		})
	}
	sort.Slice(responses, func(i, j int) bool {
		if responses[i].SitoneCount != responses[j].SitoneCount {
			return responses[i].SitoneCount > responses[j].SitoneCount
		}
		if responses[i].OpenPower != responses[j].OpenPower {
			return responses[i].OpenPower > responses[j].OpenPower
		}
		if responses[i].Nickname != responses[j].Nickname {
			return responses[i].Nickname < responses[j].Nickname
		}
		return responses[i].PlayerID < responses[j].PlayerID
	})
	return responses
}

func inventoryItemResponses(store *content.Store, records []mongomodel.PlayerItem) []InventoryItemResponse {
	responses := make([]InventoryItemResponse, 0, len(records))
	for _, record := range records {
		item, ok := store.GetItem(record.ItemID)
		if !ok {
			continue
		}
		responses = append(responses, InventoryItemResponse{
			ID:       record.ID,
			ItemID:   record.ItemID,
			Quantity: record.Quantity,
			Item: InventoryItemDetail{
				ID:          item.ID,
				Name:        item.Name,
				Type:        item.Type,
				Rarity:      item.Rarity,
				Description: item.Description,
				IconPath:    item.IconPath,
				Source:      item.Source,
			},
		})
	}
	return responses
}

func inventorySitoneResponses(store *content.Store, records []mongomodel.PlayerSitone) []InventorySitoneResponse {
	responses := make([]InventorySitoneResponse, 0, len(records))
	for _, record := range records {
		sitone, ok := store.GetSitone(record.SitoneID)
		if !ok {
			continue
		}
		responses = append(responses, InventorySitoneResponse{
			ID:       record.ID,
			SitoneID: record.SitoneID,
			Quantity: record.Quantity,
			Sitone: InventorySitoneDetail{
				ID:                 sitone.ID,
				Name:               sitone.Name,
				Type:               sitone.Type,
				Rarity:             sitone.Rarity,
				Style:              sitone.Style,
				Description:        sitone.Description,
				IconPath:           sitone.IconPath,
				AbilityName:        sitone.AbilityName,
				AbilityKind:        sitone.AbilityKind,
				AbilityValue:       sitone.AbilityValue,
				AbilityCount:       sitone.AbilityCount,
				AbilityDescription: sitone.AbilityDescription,
			},
		})
	}
	return responses
}

func quantityTotalItems(records []mongomodel.PlayerItem) int {
	total := 0
	for _, record := range records {
		if record.Quantity > 0 {
			total += record.Quantity
		}
	}
	return total
}

func quantityTotalSitones(records []mongomodel.PlayerSitone) int {
	total := 0
	for _, record := range records {
		if record.Quantity > 0 {
			total += record.Quantity
		}
	}
	return total
}
