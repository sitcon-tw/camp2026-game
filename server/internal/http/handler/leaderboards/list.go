package leaderboards

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	ScopeTeams   = "teams"
	ScopePlayers = "players"
)

type rankStats struct {
	SitoneCount int
	OpenPower   int
}

// List godoc
// @Summary List leaderboard
// @Description Lists team or player rankings by sitone count, then open power.
// @Tags leaderboards
// @Produce json
// @Security AuthCookieAuth
// @Param scope query string false "Leaderboard scope" Enums(teams,players)
// @Success 200 {object} ListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /leaderboards [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	scope := normalizeScope(r.URL.Query().Get("scope"))
	if !isValidScope(scope) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnprocessableEntity, "unsupported leaderboard scope"))
		return
	}

	response, err := h.leaderboard(r.Context(), scope, player)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("leaderboard unavailable", "leaderboard_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func normalizeScope(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ScopeTeams
	}
	return value
}

func isValidScope(value string) bool {
	switch value {
	case ScopeTeams, ScopePlayers:
		return true
	default:
		return false
	}
}

func (h *Handler) leaderboard(ctx context.Context, scope string, currentPlayer mongomodel.Player) (ListResponse, error) {
	teams, err := h.findTeams(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	players, err := h.findLeaderboardPlayers(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	stats, err := h.playerStats(ctx)
	if err != nil {
		return ListResponse{}, err
	}

	var entries []RankEntryResponse
	switch scope {
	case ScopeTeams:
		entries = teamEntries(teams, players, stats, currentPlayerTeamID(currentPlayer))
	case ScopePlayers:
		entries = playerEntries(players, teams, stats, currentPlayer.ID)
	default:
		entries = []RankEntryResponse{}
	}
	currentEntry, gapToPrevious := currentEntryAndGap(entries)

	return ListResponse{
		Scope:         scope,
		Entries:       entries,
		CurrentEntry:  currentEntry,
		GapToPrevious: gapToPrevious,
	}, nil
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

func (h *Handler) playerStats(ctx context.Context) (map[string]rankStats, error) {
	sitoneCounts, err := h.scoreMap(ctx, mongomodel.PlayerSitonesCollection, playerSitoneCountsPipeline())
	if err != nil {
		return nil, err
	}
	openPower, err := h.scoreMap(ctx, mongomodel.OpenPowerRecordsCollection, openPowerScoresByPlayerPipeline())
	if err != nil {
		return nil, err
	}

	stats := make(map[string]rankStats, len(sitoneCounts)+len(openPower))
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

func teamEntries(teams []mongomodel.Team, players []mongomodel.Player, stats map[string]rankStats, currentTeamID string) []RankEntryResponse {
	statsByTeam := make(map[string]rankStats, len(teams))
	for _, player := range players {
		if !isLeaderboardPlayer(player) {
			continue
		}
		current := statsByTeam[player.TeamID]
		playerStats := stats[player.ID]
		current.SitoneCount += playerStats.SitoneCount
		current.OpenPower += playerStats.OpenPower
		statsByTeam[player.TeamID] = current
	}

	entries := make([]RankEntryResponse, 0, len(teams))
	for _, team := range teams {
		if team.ID == "" {
			continue
		}
		teamStats := statsByTeam[team.ID]
		entries = append(entries, RankEntryResponse{
			ID:          team.ID,
			TeamID:      team.ID,
			Name:        team.Name,
			SitoneCount: teamStats.SitoneCount,
			OpenPower:   teamStats.OpenPower,
			Current:     team.ID == currentTeamID,
		})
	}
	sortRankEntries(entries)
	assignRanks(entries)
	return entries
}

func playerEntries(players []mongomodel.Player, teams []mongomodel.Team, stats map[string]rankStats, currentPlayerID string) []RankEntryResponse {
	teamNames := teamNamesByID(teams)
	entries := make([]RankEntryResponse, 0, len(players))
	for _, player := range players {
		if !isLeaderboardPlayer(player) {
			continue
		}
		playerStats := stats[player.ID]
		entries = append(entries, RankEntryResponse{
			ID:          player.ID,
			Name:        player.Nickname,
			TeamID:      player.TeamID,
			TeamName:    teamNames[player.TeamID],
			SitoneCount: playerStats.SitoneCount,
			OpenPower:   playerStats.OpenPower,
			Current:     player.ID == currentPlayerID,
		})
	}
	sortRankEntries(entries)
	assignRanks(entries)
	return entries
}

func teamNamesByID(teams []mongomodel.Team) map[string]string {
	names := make(map[string]string, len(teams))
	for _, team := range teams {
		if team.ID == "" {
			continue
		}
		names[team.ID] = team.Name
	}
	return names
}

func isLeaderboardPlayer(player mongomodel.Player) bool {
	return player.ID != "" && player.TeamID != ""
}

func sortRankEntries(entries []RankEntryResponse) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SitoneCount != entries[j].SitoneCount {
			return entries[i].SitoneCount > entries[j].SitoneCount
		}
		if entries[i].OpenPower != entries[j].OpenPower {
			return entries[i].OpenPower > entries[j].OpenPower
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].ID < entries[j].ID
	})
}

func assignRanks(entries []RankEntryResponse) {
	for i := range entries {
		entries[i].Rank = i + 1
	}
}

func currentEntryAndGap(entries []RankEntryResponse) (*RankEntryResponse, int) {
	for i := range entries {
		if !entries[i].Current {
			continue
		}
		if i == 0 {
			return &entries[i], 0
		}
		return &entries[i], entries[i-1].SitoneCount - entries[i].SitoneCount
	}
	return nil, 0
}

func currentPlayerTeamID(player mongomodel.Player) string {
	return player.TeamID
}
