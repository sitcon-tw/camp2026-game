package staff

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const teamSearchQueryMaxLen = 64

// ListTeams godoc
// @Summary List teams as staff
// @Description Staff-only endpoint. Lists teams for reward targeting and supports optional name or team ID filtering.
// @Tags staff
// @Produce json
// @Security AuthCookieAuth
// @Param query query string false "Team name or team ID keyword"
// @Success 200 {object} ListTeamsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/teams [get]
func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	if _, ok := currentStaff(w, r); !ok || !h.requireDatabase(w, r) {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if len([]rune(query)) > teamSearchQueryMaxLen {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(
			"invalid query parameter",
			httpx.ErrorDetail{
				Location: "query.query",
				Message:  "query must be at most 64",
			},
		))
		return
	}

	teams, err := h.listTeams(r.Context(), query)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "team search failed"))
		return
	}

	memberCounts, err := h.countPlayersByTeamID(r.Context(), teamIDs(teams))
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "team search failed"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ListTeamsResponse{
		Teams: staffTeamResponses(teams, memberCounts),
	})
}

func (h *Handler) listTeams(ctx context.Context, query string) ([]mongomodel.Team, error) {
	filter := bson.M{}
	if query != "" {
		filter = teamSearchFilter(query)
	}

	cursor, err := h.db.Collection(mongomodel.TeamsCollection).Find(
		ctx,
		filter,
		options.Find().
			SetSort(bson.D{
				{Key: "name", Value: 1},
				{Key: "_id", Value: 1},
			}),
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

func teamSearchFilter(query string) bson.M {
	regex := bson.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"}
	return bson.M{
		"$or": bson.A{
			bson.M{"name": regex},
			bson.M{"_id": regex},
		},
	}
}

func teamIDs(teams []mongomodel.Team) []string {
	ids := make([]string, 0, len(teams))
	for _, team := range teams {
		if team.ID == "" {
			continue
		}
		ids = append(ids, team.ID)
	}
	return ids
}

func (h *Handler) countPlayersByTeamID(ctx context.Context, teamIDs []string) (map[string]int, error) {
	if len(teamIDs) == 0 {
		return map[string]int{}, nil
	}

	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Aggregate(ctx, mongoPipelineCountPlayersByTeamID(teamIDs))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var rows []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		if row.ID == "" {
			continue
		}
		counts[row.ID] = row.Count
	}
	return counts, nil
}

func mongoPipelineCountPlayersByTeamID(teamIDs []string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "team_id", Value: bson.D{{Key: "$in", Value: teamIDs}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$team_id"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
}

func staffTeamResponses(teams []mongomodel.Team, memberCounts map[string]int) []StaffTeamResponse {
	responses := make([]StaffTeamResponse, 0, len(teams))
	for _, team := range teams {
		if team.ID == "" || team.Name == "" {
			continue
		}
		responses = append(responses, StaffTeamResponse{
			TeamID:      team.ID,
			Name:        team.Name,
			MemberCount: memberCounts[team.ID],
		})
	}
	return responses
}
