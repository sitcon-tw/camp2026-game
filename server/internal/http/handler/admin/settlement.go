package admin

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const settlementTShirtItemID = "item_tshirt_2026"
const settlementExcludedTeamID = "team-010"

type SettlementResponse struct {
	GeneratedAt time.Time                  `json:"generatedAt"`
	TShirts     []SettlementTShirtResponse `json:"tshirts"`
	Awards      []SettlementAwardResponse  `json:"awards"`
}

type SettlementTShirtResponse struct {
	SettlementPlayerResponse
	Quantity int `json:"quantity" example:"2"`
}

type SettlementAwardResponse struct {
	Key     string                     `json:"key"`
	Title   string                     `json:"title"`
	Metric  string                     `json:"metric,omitempty"`
	Players []SettlementPlayerResponse `json:"players"`
	Team    *SettlementTeamResponse    `json:"team,omitempty"`
}

type SettlementPlayerResponse struct {
	PlayerID  string                        `json:"playerId"`
	Nickname  string                        `json:"nickname"`
	AvatarURL string                        `json:"avatarUrl,omitempty"`
	Team      *DashboardTeamSummaryResponse `json:"team,omitempty"`
}

type SettlementTeamResponse struct {
	TeamID    string  `json:"teamId"`
	Name      string  `json:"name"`
	AvatarURL string  `json:"avatarUrl,omitempty"`
	Value     float64 `json:"value"`
}

type settlementData struct {
	Dashboard DashboardResponse
	TShirts   []settlementPlayerQuantityStat
	Front     *mongomodel.Front
	Matches   []mongomodel.Match
}

type settlementPlayerQuantityStat struct {
	PlayerID string `bson:"_id"`
	Quantity int    `bson:"quantity"`
}

// Settlement godoc
// @Summary Calculate admin final settlement
// @Description Calculates the current T-shirt item count and final award recipients without modifying inventory.
// @Tags admin
// @Produce json
// @Success 200 {object} SettlementResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/settlement [get]
func (h *Handler) Settlement(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	data, err := h.settlementData(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("settlement unavailable", "admin_settlement_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, buildSettlementResponse(time.Now().UTC(), data))
}

func (h *Handler) settlementData(ctx context.Context) (settlementData, error) {
	raw, err := h.dashboardRawData(ctx)
	if err != nil {
		return settlementData{}, err
	}
	eligiblePlayerIDs := make([]string, 0, len(raw.Players))
	for _, player := range raw.Players {
		if player.Role != authctx.PlayerRoleStaff && player.TeamID != settlementExcludedTeamID {
			eligiblePlayerIDs = append(eligiblePlayerIDs, player.ID)
		}
	}
	tshirts, err := aggregateAllDashboard[settlementPlayerQuantityStat](ctx, h.db, mongomodel.PlayerItemsCollection, []bson.D{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: bson.D{{Key: "$in", Value: eligiblePlayerIDs}}}, {Key: "item_id", Value: settlementTShirtItemID}, {Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$player_id"}, {Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$quantity"}}}}}},
	})
	if err != nil {
		return settlementData{}, err
	}
	fronts, err := findAllDashboard[mongomodel.Front](ctx, h.db, mongomodel.FrontsCollection, bson.M{"current": true}, options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(1))
	if err != nil {
		return settlementData{}, err
	}
	matches, err := findAllDashboard[mongomodel.Match](ctx, h.db, mongomodel.MatchesCollection, bson.M{"status": mongomodel.MatchStatusCompleted}, options.Find().SetSort(bson.D{{Key: "completed_at", Value: -1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return settlementData{}, err
	}
	var front *mongomodel.Front
	if len(fronts) > 0 {
		front = &fronts[0]
	}
	return settlementData{Dashboard: buildDashboardResponse(time.Now().UTC(), h.content, raw), TShirts: tshirts, Front: front, Matches: matches}, nil
}

func buildSettlementResponse(now time.Time, data settlementData) SettlementResponse {
	players := make([]DashboardPlayerResponse, 0, len(data.Dashboard.Players))
	for _, player := range data.Dashboard.Players {
		if player.Role != authctx.PlayerRoleStaff && (player.Team == nil || player.Team.TeamID != settlementExcludedTeamID) {
			players = append(players, player)
		}
	}
	byID := make(map[string]DashboardPlayerResponse, len(players))
	for _, player := range players {
		byID[player.PlayerID] = player
	}

	response := SettlementResponse{GeneratedAt: now, TShirts: []SettlementTShirtResponse{}, Awards: []SettlementAwardResponse{}}
	for _, record := range data.TShirts {
		if player, ok := byID[record.PlayerID]; ok && record.Quantity > 0 {
			response.TShirts = append(response.TShirts, SettlementTShirtResponse{SettlementPlayerResponse: settlementPlayer(player), Quantity: record.Quantity})
		}
	}
	sort.Slice(response.TShirts, func(i, j int) bool {
		if response.TShirts[i].Quantity != response.TShirts[j].Quantity {
			return response.TShirts[i].Quantity > response.TShirts[j].Quantity
		}
		if response.TShirts[i].Nickname != response.TShirts[j].Nickname {
			return response.TShirts[i].Nickname < response.TShirts[j].Nickname
		}
		return response.TShirts[i].PlayerID < response.TShirts[j].PlayerID
	})

	response.Awards = append(response.Awards,
		playerSettlementAward("most_sitones", "小隊最多小石的人", players, sortSettlementPlayers(func(p DashboardPlayerResponse) float64 { return float64(p.SitoneCount) }), func(p DashboardPlayerResponse) string { return settlementIntMetric(p.SitoneCount, " 顆") }),
		largestIslandAward(data.Front),
		playerSettlementAward("most_items", "最多道具的人", players, sortSettlementPlayers(func(p DashboardPlayerResponse) float64 { return float64(p.ItemCount) }), func(p DashboardPlayerResponse) string { return settlementIntMetric(p.ItemCount, " 個") }),
		lowestAccuracyAward(players),
		playerSettlementAward("lowest_open_power", "開源力最低的人", players, sortSettlementPlayersAscending(func(p DashboardPlayerResponse) float64 { return float64(p.OpenPower) }), func(p DashboardPlayerResponse) string { return settlementIntMetric(p.OpenPower, " OP") }),
		lastBattleAward(data.Matches, byID),
		lowestAverageSitonesAward(players, data.Dashboard.Teams),
		mostHumanBattlesAward(data.Matches, byID),
		teamFirstPlacesAward(players),
	)
	return response
}

func playerSettlementAward(key, title string, players []DashboardPlayerResponse, sorter func([]DashboardPlayerResponse), metric func(DashboardPlayerResponse) string) SettlementAwardResponse {
	copyPlayers := append([]DashboardPlayerResponse(nil), players...)
	if len(copyPlayers) == 0 {
		return SettlementAwardResponse{Key: key, Title: title, Players: []SettlementPlayerResponse{}}
	}
	sorter(copyPlayers)
	winner := copyPlayers[0]
	return SettlementAwardResponse{Key: key, Title: title, Metric: metric(winner), Players: []SettlementPlayerResponse{settlementPlayer(winner)}}
}

func sortSettlementPlayers(value func(DashboardPlayerResponse) float64) func([]DashboardPlayerResponse) {
	return func(players []DashboardPlayerResponse) {
		sort.Slice(players, func(i, j int) bool {
			if value(players[i]) != value(players[j]) {
				return value(players[i]) > value(players[j])
			}
			return settlementPlayerTieLess(players[i], players[j])
		})
	}
}

func sortSettlementPlayersAscending(value func(DashboardPlayerResponse) float64) func([]DashboardPlayerResponse) {
	return func(players []DashboardPlayerResponse) {
		sort.Slice(players, func(i, j int) bool {
			if value(players[i]) != value(players[j]) {
				return value(players[i]) < value(players[j])
			}
			return settlementPlayerTieLess(players[i], players[j])
		})
	}
}

func settlementPlayerTieLess(a, b DashboardPlayerResponse) bool {
	if a.Nickname != b.Nickname {
		return a.Nickname < b.Nickname
	}
	return a.PlayerID < b.PlayerID
}

func lowestAccuracyAward(players []DashboardPlayerResponse) SettlementAwardResponse {
	eligible := make([]DashboardPlayerResponse, 0, len(players))
	for _, player := range players {
		if player.AnswerCount > 0 {
			eligible = append(eligible, player)
		}
	}
	return playerSettlementAward("lowest_accuracy", "答對率最低的人", eligible, sortSettlementPlayersAscending(func(p DashboardPlayerResponse) float64 { return float64(p.AnswerAccuracy) }), func(p DashboardPlayerResponse) string { return settlementIntMetric(p.AnswerAccuracy, "%") })
}

func largestIslandAward(front *mongomodel.Front) SettlementAwardResponse {
	award := SettlementAwardResponse{Key: "largest_island", Title: "島嶼最大隊伍", Players: []SettlementPlayerResponse{}}
	if front == nil {
		return award
	}
	var winner mongomodel.FrontTeam
	for _, team := range front.Teams {
		if team.TeamID == settlementExcludedTeamID {
			continue
		}
		if team.ControlledCells > winner.ControlledCells || (team.ControlledCells == winner.ControlledCells && team.Name < winner.Name) {
			winner = team
		}
	}
	if winner.TeamID != "" {
		award.Metric = settlementIntMetric(winner.ControlledCells, " 格")
		award.Team = &SettlementTeamResponse{TeamID: winner.TeamID, Name: winner.Name, Value: float64(winner.ControlledCells)}
	}
	return award
}

func lowestAverageSitonesAward(players []DashboardPlayerResponse, teams []DashboardTeamResponse) SettlementAwardResponse {
	award := SettlementAwardResponse{Key: "lowest_team_average_sitones", Title: "隊伍平均小石最少", Players: []SettlementPlayerResponse{}}
	totals := map[string]int{}
	counts := map[string]int{}
	for _, player := range players {
		if player.Team != nil {
			totals[player.Team.TeamID] += player.SitoneCount
			counts[player.Team.TeamID]++
		}
	}
	eligible := make([]DashboardTeamResponse, 0, len(teams))
	for _, team := range teams {
		if counts[team.TeamID] > 0 {
			team.AverageSitones = float64(totals[team.TeamID]) / float64(counts[team.TeamID])
			eligible = append(eligible, team)
		}
	}
	if len(eligible) == 0 {
		return award
	}
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].AverageSitones != eligible[j].AverageSitones {
			return eligible[i].AverageSitones < eligible[j].AverageSitones
		}
		return eligible[i].Name < eligible[j].Name
	})
	winner := eligible[0]
	award.Metric = settlementFloatMetric(winner.AverageSitones, " 顆 / 人")
	award.Team = &SettlementTeamResponse{TeamID: winner.TeamID, Name: winner.Name, AvatarURL: winner.AvatarURL, Value: winner.AverageSitones}
	return award
}

func humanPlayers(match mongomodel.Match) []mongomodel.MatchPlayer {
	if match.Mode == mongomodel.MatchModeComputer {
		return nil
	}
	out := []mongomodel.MatchPlayer{}
	for _, player := range match.Players {
		if player.Kind != mongomodel.MatchPlayerKindComputer && player.PlayerID != "" {
			out = append(out, player)
		}
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

func lastBattleAward(matches []mongomodel.Match, byID map[string]DashboardPlayerResponse) SettlementAwardResponse {
	award := SettlementAwardResponse{Key: "last_human_battle", Title: "最後一場真人對戰", Players: []SettlementPlayerResponse{}}
	for _, match := range matches {
		humans := humanPlayers(match)
		if len(humans) == 0 {
			continue
		}
		players := make([]SettlementPlayerResponse, 0, len(humans))
		for _, human := range humans {
			if player, ok := byID[human.PlayerID]; ok {
				players = append(players, settlementPlayer(player))
			}
		}
		if len(players) < 2 {
			continue
		}
		award.Players = players
		award.Metric = dashboardMatchSortTime(match).Format("2006-01-02 15:04")
		return award
	}
	return award
}

func mostHumanBattlesAward(matches []mongomodel.Match, byID map[string]DashboardPlayerResponse) SettlementAwardResponse {
	counts := map[string]int{}
	for _, match := range matches {
		for _, player := range humanPlayers(match) {
			if _, ok := byID[player.PlayerID]; ok {
				counts[player.PlayerID]++
			}
		}
	}
	players := make([]DashboardPlayerResponse, 0, len(counts))
	for id := range counts {
		players = append(players, byID[id])
	}
	return playerSettlementAward("most_human_battles", "最多真人對戰的人", players, sortSettlementPlayers(func(p DashboardPlayerResponse) float64 { return float64(counts[p.PlayerID]) }), func(p DashboardPlayerResponse) string { return settlementIntMetric(counts[p.PlayerID], " 場") })
}

func teamFirstPlacesAward(players []DashboardPlayerResponse) SettlementAwardResponse {
	award := SettlementAwardResponse{Key: "team_first_places", Title: "每小隊的第一名", Players: []SettlementPlayerResponse{}}
	byTeam := map[string][]DashboardPlayerResponse{}
	for _, player := range players {
		if player.Team != nil {
			byTeam[player.Team.TeamID] = append(byTeam[player.Team.TeamID], player)
		}
	}
	teamIDs := make([]string, 0, len(byTeam))
	for id := range byTeam {
		teamIDs = append(teamIDs, id)
	}
	sort.Strings(teamIDs)
	for _, id := range teamIDs {
		group := byTeam[id]
		sortSettlementPlayers(func(p DashboardPlayerResponse) float64 { return float64(p.SitoneCount) })(group)
		award.Players = append(award.Players, settlementPlayer(group[0]))
	}
	award.Metric = settlementIntMetric(len(award.Players), " 隊")
	return award
}

func settlementPlayer(player DashboardPlayerResponse) SettlementPlayerResponse {
	return SettlementPlayerResponse{PlayerID: player.PlayerID, Nickname: player.Nickname, AvatarURL: player.AvatarURL, Team: player.Team}
}
func settlementIntMetric(value int, suffix string) string {
	return settlementFloatMetric(float64(value), suffix)
}
func settlementFloatMetric(value float64, suffix string) string {
	return fmt.Sprintf("%g%s", value, suffix)
}
