package staff

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	playerSearchLimit       int64 = 10
	playerSearchQueryMaxLen       = 64
)

// ListPlayers godoc
// @Summary Search players as staff
// @Description Staff-only endpoint. Searches players by nickname or player ID for reward targeting.
// @Tags staff
// @Produce json
// @Security AuthCookieAuth
// @Param query query string false "Nickname or player ID keyword"
// @Success 200 {object} ListPlayersResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/players [get]
func (h *Handler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	if _, ok := currentStaff(w, r); !ok || !h.requireDatabase(w, r) {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if len([]rune(query)) > playerSearchQueryMaxLen {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(
			"invalid query parameter",
			httpx.ErrorDetail{
				Location: "query.query",
				Message:  "query must be at most 64",
			},
		))
		return
	}
	if query == "" {
		httpx.WriteJSON(w, http.StatusOK, ListPlayersResponse{Players: []StaffPlayerResponse{}})
		return
	}

	players, err := h.searchPlayers(r.Context(), query)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "player search failed"))
		return
	}

	teams, err := h.findTeamsByID(r.Context(), playerTeamIDs(players))
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "player search failed"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ListPlayersResponse{
		Players: staffPlayerResponses(players, teams),
	})
}

func (h *Handler) searchPlayers(ctx context.Context, query string) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		playerSearchFilter(query),
		options.Find().
			SetProjection(bson.D{
				{Key: "auth_token", Value: 0},
				{Key: "qrcode_token", Value: 0},
				{Key: "default_sitone_ids", Value: 0},
			}).
			SetSort(bson.D{
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			}).
			SetLimit(playerSearchLimit),
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

func playerSearchFilter(query string) bson.M {
	regex := bson.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"}
	return bson.M{
		"$or": bson.A{
			bson.M{"nickname": regex},
			bson.M{"_id": regex},
		},
	}
}

func playerTeamIDs(players []mongomodel.Player) []string {
	seen := make(map[string]struct{}, len(players))
	ids := make([]string, 0, len(players))
	for _, player := range players {
		if player.TeamID == "" {
			continue
		}
		if _, ok := seen[player.TeamID]; ok {
			continue
		}
		seen[player.TeamID] = struct{}{}
		ids = append(ids, player.TeamID)
	}
	return ids
}

func (h *Handler) findTeamsByID(ctx context.Context, teamIDs []string) (map[string]mongomodel.Team, error) {
	if len(teamIDs) == 0 {
		return map[string]mongomodel.Team{}, nil
	}

	cursor, err := h.db.Collection(mongomodel.TeamsCollection).Find(
		ctx,
		bson.M{"_id": bson.M{"$in": teamIDs}},
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

	byID := make(map[string]mongomodel.Team, len(teams))
	for _, team := range teams {
		if team.ID == "" {
			continue
		}
		byID[team.ID] = team
	}
	return byID, nil
}

func staffPlayerResponses(players []mongomodel.Player, teams map[string]mongomodel.Team) []StaffPlayerResponse {
	responses := make([]StaffPlayerResponse, 0, len(players))
	for _, player := range players {
		if player.ID == "" || player.Nickname == "" {
			continue
		}

		response := StaffPlayerResponse{
			PlayerID:  player.ID,
			Nickname:  player.Nickname,
			AvatarURL: player.AvatarURL,
		}
		if team, ok := teams[player.TeamID]; ok {
			response.Team = &RewardTeamResponse{
				TeamID: team.ID,
				Name:   team.Name,
			}
		}
		responses = append(responses, response)
	}
	return responses
}
