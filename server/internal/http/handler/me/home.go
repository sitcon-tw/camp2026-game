package me

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Home godoc
// @Summary Get current player home summary
// @Description Returns the authenticated player's home page summary, resource counts, and team rank by per-player average sitone count and average open power.
// @Tags me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} HomeResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/home [get]
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}

	openPower, err := h.sumOpenPower(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_open_power_sum_failed", err))
		return
	}
	sitoneCount, err := h.sumPlayerQuantity(r.Context(), mongomodel.PlayerSitonesCollection, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_sitone_count_failed", err))
		return
	}
	itemCount, err := h.sumPlayerQuantity(r.Context(), mongomodel.PlayerItemsCollection, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_item_count_failed", err))
		return
	}
	settings, err := gamecontrol.ReadSettings(r.Context(), h.db)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_settings_lookup_failed", err))
		return
	}

	var team *mongomodel.Team
	var rank *TeamRankResponse
	var teamMembers []mongomodel.Player
	teamID := playerTeamID(player)
	if teamID != "" {
		foundTeam, err := h.findTeam(r.Context(), teamID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_team_lookup_failed", err))
			return
		}
		team = &foundTeam

		rank, err = h.teamRank(r.Context(), teamID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_team_rank_failed", err))
			return
		}
		teamMembers, err = h.findTeamMembers(r.Context(), teamID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("home summary unavailable", "me_home_team_members_lookup_failed", err))
			return
		}
	}

	httpx.WriteJSON(w, http.StatusOK, HomeResponse{
		Player: statusResponse(player, team, openPower, teamMembers),
		Summary: HomeSummaryResponse{
			OpenPower:   openPower,
			SitoneCount: sitoneCount,
			ItemCount:   itemCount,
		},
		TeamRank: rank,
		Actions:  homeActions(settings),
	})
}

func (h *Handler) sumPlayerQuantity(ctx context.Context, collection string, playerID string) (int, error) {
	cursor, err := h.db.Collection(collection).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: playerID},
			{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	return intTotalFromCursor(ctx, cursor)
}

type teamRankStats struct {
	SitoneCount int
	OpenPower   int
}

type teamRankAccumulator struct {
	teamRankStats
	PlayerCount int
}

func (h *Handler) teamRank(ctx context.Context, currentTeamID string) (*TeamRankResponse, error) {
	teams, err := h.findTeams(ctx)
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, nil
	}

	players, err := h.findLeaderboardPlayers(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := h.playerRankStats(ctx)
	if err != nil {
		return nil, err
	}

	return currentTeamRank(teamRankEntries(teams, players, stats), currentTeamID), nil
}

func (h *Handler) findTeams(ctx context.Context) ([]mongomodel.Team, error) {
	cursor, err := h.db.Collection(mongomodel.TeamsCollection).Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var teams []mongomodel.Team
	if err := cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	if teams == nil {
		return []mongomodel.Team{}, nil
	}
	return teams, nil
}

func (h *Handler) findLeaderboardPlayers(ctx context.Context) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": bson.M{"$exists": true, "$ne": ""}},
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

func (h *Handler) playerRankStats(ctx context.Context) (map[string]teamRankStats, error) {
	sitoneCounts, err := h.scoreMap(ctx, mongomodel.PlayerSitonesCollection, playerSitoneCountsPipeline())
	if err != nil {
		return nil, err
	}
	openPower, err := h.scoreMap(ctx, mongomodel.OpenPowerRecordsCollection, openPowerScoresByPlayerPipeline())
	if err != nil {
		return nil, err
	}

	stats := make(map[string]teamRankStats, len(sitoneCounts)+len(openPower))
	for playerID, count := range sitoneCounts {
		current := stats[playerID]
		current.SitoneCount = count
		stats[playerID] = current
	}
	for playerID, score := range openPower {
		current := stats[playerID]
		current.OpenPower = score
		stats[playerID] = current
	}
	return stats, nil
}

func (h *Handler) scoreMap(ctx context.Context, collection string, pipeline mongo.Pipeline) (map[string]int, error) {
	cursor, err := h.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	return scoreMapFromCursor(ctx, cursor)
}

func playerSitoneCountsPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	}
}

func openPowerScoresByPlayerPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	}
}

func scoreMapFromCursor(ctx context.Context, cursor *mongo.Cursor) (map[string]int, error) {
	var rows []struct {
		ID    string `bson:"_id"`
		Score int    `bson:"score"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	out := make(map[string]int, len(rows))
	for _, row := range rows {
		if row.ID == "" {
			continue
		}
		out[row.ID] = row.Score
	}
	return out, nil
}

func teamRankEntries(teams []mongomodel.Team, players []mongomodel.Player, stats map[string]teamRankStats) []TeamRankResponse {
	statsByTeam := make(map[string]teamRankAccumulator, len(teams))
	for _, player := range players {
		if player.ID == "" || player.TeamID == "" {
			continue
		}
		current := statsByTeam[player.TeamID]
		playerStats := stats[player.ID]
		current.SitoneCount += playerStats.SitoneCount
		current.OpenPower += playerStats.OpenPower
		current.PlayerCount++
		statsByTeam[player.TeamID] = current
	}

	rows := make([]TeamRankResponse, 0, len(teams))
	for _, team := range teams {
		if team.ID == "" {
			continue
		}
		teamStats := statsByTeam[team.ID]
		rows = append(rows, TeamRankResponse{
			TeamID:      team.ID,
			Name:        team.Name,
			AvatarURL:   team.AvatarURL,
			PlayerCount: teamStats.PlayerCount,
			SitoneCount: teamStats.SitoneCount,
			AverageSitones: balancedAverage(
				teamStats.SitoneCount,
				teamStats.PlayerCount,
			),
			OpenPower: teamStats.OpenPower,
			AverageOpenPower: balancedAverage(
				teamStats.OpenPower,
				teamStats.PlayerCount,
			),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if comparison := compareBalancedTotals(
			rows[i].SitoneCount,
			rows[i].PlayerCount,
			rows[j].SitoneCount,
			rows[j].PlayerCount,
		); comparison != 0 {
			return comparison > 0
		}
		if comparison := compareBalancedTotals(
			rows[i].OpenPower,
			rows[i].PlayerCount,
			rows[j].OpenPower,
			rows[j].PlayerCount,
		); comparison != 0 {
			return comparison > 0
		}
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].TeamID < rows[j].TeamID
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func currentTeamRank(rows []TeamRankResponse, currentTeamID string) *TeamRankResponse {
	for i := range rows {
		if rows[i].TeamID != currentTeamID {
			continue
		}
		if i > 0 {
			rows[i].GapToPrevious = balancedAverageGap(rows[i-1], rows[i])
		}
		return &rows[i]
	}
	return nil
}

func compareBalancedTotals(leftTotal int, leftPlayerCount int, rightTotal int, rightPlayerCount int) int {
	leftDenominator := balancedDenominator(leftPlayerCount)
	rightDenominator := balancedDenominator(rightPlayerCount)
	left := int64(leftTotal) * int64(rightDenominator)
	right := int64(rightTotal) * int64(leftDenominator)
	if left > right {
		return 1
	}
	if left < right {
		return -1
	}
	return 0
}

func balancedAverage(total int, playerCount int) float64 {
	if playerCount <= 0 {
		return 0
	}
	return roundOneDecimal(float64(total) / float64(playerCount))
}

func balancedAverageGap(previous TeamRankResponse, current TeamRankResponse) float64 {
	previousAverage := float64(previous.SitoneCount) / float64(balancedDenominator(previous.PlayerCount))
	currentAverage := float64(current.SitoneCount) / float64(balancedDenominator(current.PlayerCount))
	return roundOneDecimal(previousAverage - currentAverage)
}

func balancedDenominator(playerCount int) int {
	if playerCount <= 0 {
		return 1
	}
	return playerCount
}

func roundOneDecimal(value float64) float64 {
	return math.Round(value*10) / 10
}

func intTotalFromCursor(ctx context.Context, cursor *mongo.Cursor) (int, error) {
	var totals []struct {
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &totals); err != nil {
		return 0, err
	}
	if len(totals) == 0 {
		return 0, nil
	}
	return totals[0].Total, nil
}

func homeActions(settings gamecontrol.Settings) []HomeActionResponse {
	battleEnabled := !settings.BattleOpeningLocked(time.Now())
	return []HomeActionResponse{
		{ID: "front", Label: "開源戰線", Enabled: true},
		{ID: "battle", Label: "知識王戰", Enabled: battleEnabled},
		{ID: "shop", Label: "商店", Enabled: true},
		{ID: "sitones", Label: "小石收藏", Enabled: true},
		{ID: "inventory", Label: "道具背包", Enabled: true},
		{ID: "fusion", Label: "小石合成", Enabled: true},
		{ID: "leaderboard", Label: "排行榜", Enabled: true},
		{ID: "qrcode", Label: "個人 QR Code", Enabled: true},
		{ID: "codex", Label: "公開圖鑑", Enabled: true},
	}
}
