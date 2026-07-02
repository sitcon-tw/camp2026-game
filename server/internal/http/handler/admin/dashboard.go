package admin

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const dashboardTopLimit = 8

// Dashboard godoc
// @Summary Get admin game dashboard
// @Description Returns an admin-only operational dashboard with player, inventory, match, and activity statistics.
// @Tags admin
// @Produce json
// @Success 200 {object} DashboardResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/dashboard [get]
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	raw, err := h.dashboardRawData(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("dashboard unavailable", "admin_dashboard_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, buildDashboardResponse(time.Now().UTC(), h.content, raw))
}

type dashboardRawData struct {
	Players          []dashboardPlayer
	Teams            []mongomodel.Team
	PlayerSitones    []mongomodel.PlayerSitone
	PlayerItems      []mongomodel.PlayerItem
	OpenPowerRecords []mongomodel.OpenPowerRecord
	Matches          []mongomodel.Match
	MatchAnswers     []mongomodel.MatchAnswer
	MatchItemDrops   []mongomodel.MatchItemDrop
	ShopPurchases    []mongomodel.ShopPurchase
	FusionRecords    []mongomodel.FusionRecord
	StaffRewards     []mongomodel.StaffReward
}

type dashboardPlayer struct {
	ID        string `bson:"_id"`
	Nickname  string `bson:"nickname"`
	TeamID    string `bson:"team_id,omitempty"`
	AvatarURL string `bson:"avatar_url,omitempty"`
	Role      string `bson:"role,omitempty"`
}

func (h *Handler) dashboardRawData(ctx context.Context) (dashboardRawData, error) {
	players, err := findAllDashboard[dashboardPlayer](
		ctx,
		h.db,
		mongomodel.PlayersCollection,
		bson.M{},
		options.Find().SetProjection(dashboardPlayerProjection()),
	)
	if err != nil {
		return dashboardRawData{}, err
	}
	teams, err := findAllDashboard[mongomodel.Team](ctx, h.db, mongomodel.TeamsCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	playerSitones, err := findAllDashboard[mongomodel.PlayerSitone](ctx, h.db, mongomodel.PlayerSitonesCollection, bson.M{"quantity": bson.M{"$gt": 0}})
	if err != nil {
		return dashboardRawData{}, err
	}
	playerItems, err := findAllDashboard[mongomodel.PlayerItem](ctx, h.db, mongomodel.PlayerItemsCollection, bson.M{"quantity": bson.M{"$gt": 0}})
	if err != nil {
		return dashboardRawData{}, err
	}
	openPowerRecords, err := findAllDashboard[mongomodel.OpenPowerRecord](ctx, h.db, mongomodel.OpenPowerRecordsCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	matches, err := findAllDashboard[mongomodel.Match](ctx, h.db, mongomodel.MatchesCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	matchAnswers, err := findAllDashboard[mongomodel.MatchAnswer](ctx, h.db, mongomodel.MatchAnswersCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	matchItemDrops, err := findAllDashboard[mongomodel.MatchItemDrop](ctx, h.db, mongomodel.MatchItemDropsCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	shopPurchases, err := findAllDashboard[mongomodel.ShopPurchase](ctx, h.db, mongomodel.ShopPurchasesCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	fusionRecords, err := findAllDashboard[mongomodel.FusionRecord](ctx, h.db, mongomodel.FusionRecordsCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}
	staffRewards, err := findAllDashboard[mongomodel.StaffReward](ctx, h.db, mongomodel.StaffRewardsCollection, bson.M{})
	if err != nil {
		return dashboardRawData{}, err
	}

	return dashboardRawData{
		Players:          players,
		Teams:            teams,
		PlayerSitones:    playerSitones,
		PlayerItems:      playerItems,
		OpenPowerRecords: openPowerRecords,
		Matches:          matches,
		MatchAnswers:     matchAnswers,
		MatchItemDrops:   matchItemDrops,
		ShopPurchases:    shopPurchases,
		FusionRecords:    fusionRecords,
		StaffRewards:     staffRewards,
	}, nil
}

func dashboardPlayerProjection() bson.D {
	return bson.D{
		{Key: "_id", Value: 1},
		{Key: "nickname", Value: 1},
		{Key: "team_id", Value: 1},
		{Key: "avatar_url", Value: 1},
		{Key: "role", Value: 1},
	}
}

func findAllDashboard[T any](
	ctx context.Context,
	db *mongo.Database,
	collection string,
	filter any,
	opts ...options.Lister[options.FindOptions],
) ([]T, error) {
	cursor, err := db.Collection(collection).Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var out []T
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return []T{}, nil
	}
	return out, nil
}

type dashboardPlayerStats struct {
	Player              dashboardPlayer
	Team                *DashboardTeamSummaryResponse
	SitoneCount         int
	ItemCount           int
	OpenPower           int
	MatchCount          int
	CompletedMatchCount int
	AnswerCount         int
	CorrectAnswerCount  int
	Score               int
	LastActivityAt      time.Time
}

type dashboardTeamStats struct {
	Team        DashboardTeamSummaryResponse
	PlayerIDs   map[string]struct{}
	SitoneCount int
	ItemCount   int
	OpenPower   int
}

type dashboardInventoryAccumulator struct {
	Quantity int
	Owners   map[string]struct{}
}

func buildDashboardResponse(now time.Time, store *content.Store, raw dashboardRawData) DashboardResponse {
	teamsByID := dashboardTeamsByID(raw.Teams)
	statsByPlayer := make(map[string]*dashboardPlayerStats, len(raw.Players))
	teamStatsByID := make(map[string]*dashboardTeamStats, len(raw.Teams))
	staffCount := 0
	ungroupedPlayerCount := 0

	for _, team := range raw.Teams {
		if team.ID == "" {
			continue
		}
		teamStatsByID[team.ID] = &dashboardTeamStats{
			Team:      DashboardTeamSummaryResponse{TeamID: team.ID, Name: team.Name},
			PlayerIDs: map[string]struct{}{},
		}
	}

	for _, player := range raw.Players {
		if player.ID == "" {
			continue
		}
		if player.Role == authctx.PlayerRoleStaff {
			staffCount++
		}
		team := dashboardTeamForPlayer(player, teamsByID)
		statsByPlayer[player.ID] = &dashboardPlayerStats{
			Player: player,
			Team:   team,
		}
		if team == nil {
			ungroupedPlayerCount++
			continue
		}
		current, ok := teamStatsByID[team.TeamID]
		if !ok {
			current = &dashboardTeamStats{Team: *team, PlayerIDs: map[string]struct{}{}}
			teamStatsByID[team.TeamID] = current
		}
		current.PlayerIDs[player.ID] = struct{}{}
	}

	sitoneInventory := make(map[string]*dashboardInventoryAccumulator)
	for _, record := range raw.PlayerSitones {
		stats, ok := statsByPlayer[record.PlayerID]
		if !ok || record.Quantity <= 0 {
			continue
		}
		stats.SitoneCount += record.Quantity
		if stats.Team != nil {
			teamStatsByID[stats.Team.TeamID].SitoneCount += record.Quantity
		}
		addDashboardInventory(sitoneInventory, record.SitoneID, record.PlayerID, record.Quantity)
	}

	itemInventory := make(map[string]*dashboardInventoryAccumulator)
	for _, record := range raw.PlayerItems {
		stats, ok := statsByPlayer[record.PlayerID]
		if !ok || record.Quantity <= 0 {
			continue
		}
		stats.ItemCount += record.Quantity
		if stats.Team != nil {
			teamStatsByID[stats.Team.TeamID].ItemCount += record.Quantity
		}
		addDashboardInventory(itemInventory, record.ItemID, record.PlayerID, record.Quantity)
	}

	for _, record := range raw.OpenPowerRecords {
		stats, ok := statsByPlayer[record.PlayerID]
		if !ok {
			continue
		}
		stats.OpenPower += record.Amount
		if stats.Team != nil {
			teamStatsByID[stats.Team.TeamID].OpenPower += record.Amount
		}
		touchDashboardPlayer(stats, record.CreatedAt)
	}

	matchSummary := buildDashboardMatchSummary(raw.Matches, raw.MatchAnswers, raw.MatchItemDrops, statsByPlayer)

	shopPurchaseCount := 0
	for _, record := range raw.ShopPurchases {
		stats, ok := statsByPlayer[record.PlayerID]
		if !ok {
			continue
		}
		shopPurchaseCount++
		touchDashboardPlayer(stats, record.CreatedAt)
	}

	fusionCount := 0
	for _, record := range raw.FusionRecords {
		stats, ok := statsByPlayer[record.PlayerID]
		if !ok {
			continue
		}
		fusionCount++
		touchDashboardPlayer(stats, record.CreatedAt)
	}

	staffRewardCount := 0
	for _, record := range raw.StaffRewards {
		stats, ok := statsByPlayer[record.RecipientPlayerID]
		if !ok {
			continue
		}
		staffRewardCount++
		touchDashboardPlayer(stats, record.CreatedAt)
	}

	players := dashboardPlayerResponses(statsByPlayer)
	teams := dashboardTeamResponses(teamStatsByID, players)

	return DashboardResponse{
		GeneratedAt: now,
		Summary: DashboardSummaryResponse{
			PlayerCount:          len(statsByPlayer),
			StaffCount:           staffCount,
			TeamCount:            len(teams),
			UngroupedPlayerCount: ungroupedPlayerCount,
			TotalSitones:         dashboardTotalPlayers(players, func(player DashboardPlayerResponse) int { return player.SitoneCount }),
			TotalItems:           dashboardTotalPlayers(players, func(player DashboardPlayerResponse) int { return player.ItemCount }),
			TotalOpenPower:       dashboardTotalPlayers(players, func(player DashboardPlayerResponse) int { return player.OpenPower }),
			TotalMatches:         matchSummary.Total,
			WaitingMatches:       matchSummary.Waiting,
			ActiveMatches:        matchSummary.Active,
			CompletedMatches:     matchSummary.Completed,
			AnswerCount:          matchSummary.AnswerCount,
			CorrectAnswerCount:   matchSummary.CorrectAnswerCount,
			AnswerAccuracy:       matchSummary.AnswerAccuracy,
			ShopPurchaseCount:    shopPurchaseCount,
			FusionCount:          fusionCount,
			StaffRewardCount:     staffRewardCount,
			ItemDropCount:        matchSummary.DropAttempts,
			DroppedItemCount:     matchSummary.DropSuccesses,
		},
		TopPlayers: DashboardTopPlayersResponse{
			BySitones:   dashboardTopPlayers(players, sortPlayersBySitones),
			ByOpenPower: dashboardTopPlayers(players, sortPlayersByOpenPower),
			ByItems:     dashboardTopPlayers(players, sortPlayersByItems),
			ByScore:     dashboardTopPlayers(players, sortPlayersByScore),
			ByAccuracy:  dashboardTopPlayers(dashboardPlayersWithAnswers(players), sortPlayersByAccuracy),
		},
		Teams:   teams,
		Players: players,
		Inventory: DashboardInventoryResponse{
			Sitones: dashboardSitoneInventoryResponses(store, sitoneInventory),
			Items:   dashboardItemInventoryResponses(store, itemInventory),
		},
		Matches: matchSummary,
	}
}

func dashboardTeamsByID(teams []mongomodel.Team) map[string]mongomodel.Team {
	out := make(map[string]mongomodel.Team, len(teams))
	for _, team := range teams {
		if team.ID == "" {
			continue
		}
		out[team.ID] = team
	}
	return out
}

func dashboardTeamForPlayer(player dashboardPlayer, teamsByID map[string]mongomodel.Team) *DashboardTeamSummaryResponse {
	if player.TeamID == "" {
		return nil
	}
	if team, ok := teamsByID[player.TeamID]; ok {
		return &DashboardTeamSummaryResponse{TeamID: team.ID, Name: team.Name}
	}
	return &DashboardTeamSummaryResponse{TeamID: player.TeamID, Name: player.TeamID}
}

func addDashboardInventory(inventory map[string]*dashboardInventoryAccumulator, refID string, playerID string, quantity int) {
	if refID == "" || playerID == "" || quantity <= 0 {
		return
	}
	current, ok := inventory[refID]
	if !ok {
		current = &dashboardInventoryAccumulator{Owners: map[string]struct{}{}}
		inventory[refID] = current
	}
	current.Quantity += quantity
	current.Owners[playerID] = struct{}{}
}

func touchDashboardPlayer(stats *dashboardPlayerStats, at time.Time) {
	if stats == nil || at.IsZero() {
		return
	}
	if stats.LastActivityAt.IsZero() || at.After(stats.LastActivityAt) {
		stats.LastActivityAt = at
	}
}

func buildDashboardMatchSummary(
	matches []mongomodel.Match,
	answers []mongomodel.MatchAnswer,
	drops []mongomodel.MatchItemDrop,
	statsByPlayer map[string]*dashboardPlayerStats,
) DashboardMatchesResponse {
	summary := DashboardMatchesResponse{
		Recent: []DashboardRecentMatchResponse{},
	}
	completed := make([]mongomodel.Match, 0, len(matches))
	for _, match := range matches {
		if match.ID == "" {
			continue
		}
		summary.Total++
		switch match.Status {
		case mongomodel.MatchStatusWaiting:
			summary.Waiting++
		case mongomodel.MatchStatusActive:
			summary.Active++
		case mongomodel.MatchStatusCompleted:
			summary.Completed++
			completed = append(completed, match)
		}
		if match.Mode == mongomodel.MatchModeComputer {
			summary.Computer++
		} else {
			summary.PVP++
		}
		for _, matchPlayer := range match.Players {
			if matchPlayer.Kind == mongomodel.MatchPlayerKindComputer {
				continue
			}
			stats, ok := statsByPlayer[matchPlayer.PlayerID]
			if !ok {
				continue
			}
			stats.MatchCount++
			if match.Status == mongomodel.MatchStatusCompleted {
				stats.CompletedMatchCount++
			}
			touchDashboardPlayer(stats, match.CreatedAt)
			touchDashboardPlayer(stats, match.StartedAt)
			touchDashboardPlayer(stats, match.CompletedAt)
		}
	}

	totalScore := 0
	totalElapsedMillis := int64(0)
	for _, answer := range answers {
		stats, ok := statsByPlayer[answer.PlayerID]
		if !ok {
			continue
		}
		stats.AnswerCount++
		if answer.Correct {
			stats.CorrectAnswerCount++
			summary.CorrectAnswerCount++
		}
		stats.Score += answer.Score
		summary.AnswerCount++
		totalScore += answer.Score
		totalElapsedMillis += answer.ElapsedMillis
		touchDashboardPlayer(stats, answer.AnsweredAt)
	}
	summary.AnswerAccuracy = dashboardPercent(summary.CorrectAnswerCount, summary.AnswerCount)
	summary.AverageScore = dashboardAverage(totalScore, summary.AnswerCount)
	summary.AverageElapsedMillis = dashboardAverageInt64(totalElapsedMillis, summary.AnswerCount)

	for _, drop := range drops {
		stats, ok := statsByPlayer[drop.PlayerID]
		if !ok {
			continue
		}
		summary.DropAttempts++
		if drop.Dropped {
			summary.DropSuccesses++
		}
		touchDashboardPlayer(stats, drop.CreatedAt)
	}
	summary.DropRate = dashboardPercent(summary.DropSuccesses, summary.DropAttempts)

	sort.Slice(completed, func(i, j int) bool {
		return dashboardMatchSortTime(completed[i]).After(dashboardMatchSortTime(completed[j]))
	})
	for _, match := range completed {
		if len(summary.Recent) >= 12 {
			break
		}
		summary.Recent = append(summary.Recent, dashboardRecentMatch(match))
	}

	return summary
}

func dashboardMatchSortTime(match mongomodel.Match) time.Time {
	if !match.CompletedAt.IsZero() {
		return match.CompletedAt
	}
	if !match.StartedAt.IsZero() {
		return match.StartedAt
	}
	return match.CreatedAt
}

func dashboardRecentMatch(match mongomodel.Match) DashboardRecentMatchResponse {
	response := DashboardRecentMatchResponse{
		MatchID:     match.ID,
		Code:        match.Code,
		Mode:        dashboardMatchMode(match.Mode),
		Status:      match.Status,
		PlayerCount: len(match.Players),
		CreatedAt:   dashboardTimePtr(match.CreatedAt),
		StartedAt:   dashboardTimePtr(match.StartedAt),
		CompletedAt: dashboardTimePtr(match.CompletedAt),
	}
	for _, player := range match.Players {
		if response.WinnerPlayerID == "" || player.Score > response.TopScore {
			response.WinnerPlayerID = player.PlayerID
			response.WinnerNickname = player.Nickname
			response.TopScore = player.Score
		}
	}
	return response
}

func dashboardMatchMode(mode string) string {
	if mode == mongomodel.MatchModeComputer {
		return mongomodel.MatchModeComputer
	}
	return mongomodel.MatchModePVP
}

func dashboardTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func dashboardPlayerResponses(statsByPlayer map[string]*dashboardPlayerStats) []DashboardPlayerResponse {
	players := make([]DashboardPlayerResponse, 0, len(statsByPlayer))
	for _, stats := range statsByPlayer {
		player := stats.Player
		response := DashboardPlayerResponse{
			PlayerID:            player.ID,
			Nickname:            dashboardPlayerName(player),
			Team:                stats.Team,
			AvatarURL:           player.AvatarURL,
			Role:                player.Role,
			SitoneCount:         stats.SitoneCount,
			ItemCount:           stats.ItemCount,
			OpenPower:           stats.OpenPower,
			MatchCount:          stats.MatchCount,
			CompletedMatchCount: stats.CompletedMatchCount,
			AnswerCount:         stats.AnswerCount,
			CorrectAnswerCount:  stats.CorrectAnswerCount,
			AnswerAccuracy:      dashboardPercent(stats.CorrectAnswerCount, stats.AnswerCount),
			Score:               stats.Score,
			LastActivityAt:      dashboardTimePtr(stats.LastActivityAt),
		}
		players = append(players, response)
	}
	sortPlayersBySitones(players)
	for index := range players {
		players[index].Rank = index + 1
	}
	return players
}

func dashboardPlayerName(player dashboardPlayer) string {
	if player.Nickname != "" {
		return player.Nickname
	}
	return player.ID
}

func dashboardTeamResponses(teamStatsByID map[string]*dashboardTeamStats, players []DashboardPlayerResponse) []DashboardTeamResponse {
	playersByID := make(map[string]DashboardPlayerResponse, len(players))
	for _, player := range players {
		playersByID[player.PlayerID] = player
	}

	teams := make([]DashboardTeamResponse, 0, len(teamStatsByID))
	for _, stats := range teamStatsByID {
		playerCount := len(stats.PlayerIDs)
		team := DashboardTeamResponse{
			TeamID:           stats.Team.TeamID,
			Name:             stats.Team.Name,
			PlayerCount:      playerCount,
			SitoneCount:      stats.SitoneCount,
			ItemCount:        stats.ItemCount,
			OpenPower:        stats.OpenPower,
			AverageSitones:   dashboardAverage(stats.SitoneCount, playerCount),
			AverageItems:     dashboardAverage(stats.ItemCount, playerCount),
			AverageOpenPower: dashboardAverage(stats.OpenPower, playerCount),
		}
		for playerID := range stats.PlayerIDs {
			player := playersByID[playerID]
			if team.TopPlayer == nil || dashboardPlayerLessForTop(player, *team.TopPlayer) {
				candidate := dashboardPlayerRankResponse(player)
				team.TopPlayer = &candidate
			}
		}
		teams = append(teams, team)
	}
	sort.Slice(teams, func(i, j int) bool {
		if teams[i].SitoneCount != teams[j].SitoneCount {
			return teams[i].SitoneCount > teams[j].SitoneCount
		}
		if teams[i].OpenPower != teams[j].OpenPower {
			return teams[i].OpenPower > teams[j].OpenPower
		}
		if teams[i].Name != teams[j].Name {
			return teams[i].Name < teams[j].Name
		}
		return teams[i].TeamID < teams[j].TeamID
	})
	for index := range teams {
		teams[index].Rank = index + 1
	}
	return teams
}

func dashboardPlayerLessForTop(player DashboardPlayerResponse, current DashboardPlayerRankResponse) bool {
	if player.SitoneCount != current.SitoneCount {
		return player.SitoneCount > current.SitoneCount
	}
	if player.OpenPower != current.OpenPower {
		return player.OpenPower > current.OpenPower
	}
	if player.Nickname != current.Nickname {
		return player.Nickname < current.Nickname
	}
	return player.PlayerID < current.PlayerID
}

func dashboardPlayerRankResponse(player DashboardPlayerResponse) DashboardPlayerRankResponse {
	return DashboardPlayerRankResponse{
		Rank:                player.Rank,
		PlayerID:            player.PlayerID,
		Nickname:            player.Nickname,
		Team:                player.Team,
		AvatarURL:           player.AvatarURL,
		SitoneCount:         player.SitoneCount,
		ItemCount:           player.ItemCount,
		OpenPower:           player.OpenPower,
		MatchCount:          player.MatchCount,
		CompletedMatchCount: player.CompletedMatchCount,
		AnswerCount:         player.AnswerCount,
		CorrectAnswerCount:  player.CorrectAnswerCount,
		AnswerAccuracy:      player.AnswerAccuracy,
		Score:               player.Score,
		LastActivityAt:      player.LastActivityAt,
	}
}

func dashboardTopPlayers(players []DashboardPlayerResponse, sorter func([]DashboardPlayerResponse)) []DashboardPlayerRankResponse {
	copyPlayers := make([]DashboardPlayerResponse, len(players))
	copy(copyPlayers, players)
	sorter(copyPlayers)
	if len(copyPlayers) > dashboardTopLimit {
		copyPlayers = copyPlayers[:dashboardTopLimit]
	}
	out := make([]DashboardPlayerRankResponse, 0, len(copyPlayers))
	for index, player := range copyPlayers {
		rank := dashboardPlayerRankResponse(player)
		rank.Rank = index + 1
		out = append(out, rank)
	}
	return out
}

func dashboardPlayersWithAnswers(players []DashboardPlayerResponse) []DashboardPlayerResponse {
	out := make([]DashboardPlayerResponse, 0, len(players))
	for _, player := range players {
		if player.AnswerCount > 0 {
			out = append(out, player)
		}
	}
	return out
}

func sortPlayersBySitones(players []DashboardPlayerResponse) {
	sort.Slice(players, func(i, j int) bool {
		if players[i].SitoneCount != players[j].SitoneCount {
			return players[i].SitoneCount > players[j].SitoneCount
		}
		if players[i].OpenPower != players[j].OpenPower {
			return players[i].OpenPower > players[j].OpenPower
		}
		return dashboardPlayerNameLess(players[i], players[j])
	})
}

func sortPlayersByOpenPower(players []DashboardPlayerResponse) {
	sort.Slice(players, func(i, j int) bool {
		if players[i].OpenPower != players[j].OpenPower {
			return players[i].OpenPower > players[j].OpenPower
		}
		if players[i].SitoneCount != players[j].SitoneCount {
			return players[i].SitoneCount > players[j].SitoneCount
		}
		return dashboardPlayerNameLess(players[i], players[j])
	})
}

func sortPlayersByItems(players []DashboardPlayerResponse) {
	sort.Slice(players, func(i, j int) bool {
		if players[i].ItemCount != players[j].ItemCount {
			return players[i].ItemCount > players[j].ItemCount
		}
		if players[i].OpenPower != players[j].OpenPower {
			return players[i].OpenPower > players[j].OpenPower
		}
		return dashboardPlayerNameLess(players[i], players[j])
	})
}

func sortPlayersByScore(players []DashboardPlayerResponse) {
	sort.Slice(players, func(i, j int) bool {
		if players[i].Score != players[j].Score {
			return players[i].Score > players[j].Score
		}
		if players[i].AnswerCount != players[j].AnswerCount {
			return players[i].AnswerCount > players[j].AnswerCount
		}
		return dashboardPlayerNameLess(players[i], players[j])
	})
}

func sortPlayersByAccuracy(players []DashboardPlayerResponse) {
	sort.Slice(players, func(i, j int) bool {
		if players[i].AnswerAccuracy != players[j].AnswerAccuracy {
			return players[i].AnswerAccuracy > players[j].AnswerAccuracy
		}
		if players[i].AnswerCount != players[j].AnswerCount {
			return players[i].AnswerCount > players[j].AnswerCount
		}
		return dashboardPlayerNameLess(players[i], players[j])
	})
}

func dashboardPlayerNameLess(a DashboardPlayerResponse, b DashboardPlayerResponse) bool {
	if a.Nickname != b.Nickname {
		return a.Nickname < b.Nickname
	}
	return a.PlayerID < b.PlayerID
}

func dashboardSitoneInventoryResponses(store *content.Store, inventory map[string]*dashboardInventoryAccumulator) []DashboardInventoryEntryResponse {
	entries := make([]DashboardInventoryEntryResponse, 0, len(inventory))
	for sitoneID, stats := range inventory {
		entry := DashboardInventoryEntryResponse{
			ID:             sitoneID,
			Name:           sitoneID,
			Quantity:       stats.Quantity,
			OwnerCount:     len(stats.Owners),
			CatalogMissing: true,
		}
		if sitone, ok := store.GetSitone(sitoneID); ok {
			entry.Name = sitone.Name
			entry.Type = sitone.Type
			entry.Rarity = sitone.Rarity
			entry.IconPath = sitone.IconPath
			entry.CatalogMissing = false
		}
		entries = append(entries, entry)
	}
	sortDashboardInventory(entries)
	return entries
}

func dashboardItemInventoryResponses(store *content.Store, inventory map[string]*dashboardInventoryAccumulator) []DashboardInventoryEntryResponse {
	entries := make([]DashboardInventoryEntryResponse, 0, len(inventory))
	for itemID, stats := range inventory {
		entry := DashboardInventoryEntryResponse{
			ID:             itemID,
			Name:           itemID,
			Quantity:       stats.Quantity,
			OwnerCount:     len(stats.Owners),
			CatalogMissing: true,
		}
		if item, ok := store.GetItem(itemID); ok {
			entry.Name = item.Name
			entry.Type = item.Type
			entry.Rarity = item.Rarity
			entry.IconPath = item.IconPath
			entry.Source = item.Source
			entry.CatalogMissing = false
		}
		entries = append(entries, entry)
	}
	sortDashboardInventory(entries)
	return entries
}

func sortDashboardInventory(entries []DashboardInventoryEntryResponse) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Quantity != entries[j].Quantity {
			return entries[i].Quantity > entries[j].Quantity
		}
		if entries[i].OwnerCount != entries[j].OwnerCount {
			return entries[i].OwnerCount > entries[j].OwnerCount
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].ID < entries[j].ID
	})
}

func dashboardTotalPlayers(players []DashboardPlayerResponse, pick func(DashboardPlayerResponse) int) int {
	total := 0
	for _, player := range players {
		total += pick(player)
	}
	return total
}

func dashboardPercent(part int, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(part) * 100 / float64(total)))
}

func dashboardAverage(total int, count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Round(float64(total)/float64(count)*10) / 10
}

func dashboardAverageInt64(total int64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Round(float64(total)/float64(count)*10) / 10
}
