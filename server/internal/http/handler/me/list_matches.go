package me

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	completedMatchesPerPage = 50
)

// ListCompletedMatches godoc
// @Summary List current player completed matches
// @Description Returns completed quiz matches joined by the authenticated player. Waiting and active matches are not returned. Results are paginated with 50 items per page.
// @Tags me
// @Produce json
// @Security AuthCookieAuth
// @Param page query int false "Page number (default 1)"
// @Success 200 {object} CompletedMatchListResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/matches [get]
func (h *Handler) ListCompletedMatches(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil || parsed < 1 {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity(
				"invalid page parameter",
				httpx.ErrorDetail{
					Location: "query.page",
					Message:  "page must be a positive integer",
				},
			))
			return
		}
		page = parsed
	}

	skip := int64((page - 1) * completedMatchesPerPage)

	total, err := h.countCompletedMatches(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("matches unavailable", "me_matches_count_failed", err))
		return
	}

	matches, err := h.findCompletedMatches(r.Context(), player.ID, skip)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("matches unavailable", "me_matches_lookup_failed", err))
		return
	}

	totalPages := int(total / completedMatchesPerPage)
	if total%completedMatchesPerPage > 0 {
		totalPages++
	}

	httpx.WriteJSON(w, http.StatusOK, CompletedMatchListResponse{
		Matches:    mapCompletedMatches(matches),
		Page:       page,
		PerPage:    completedMatchesPerPage,
		Total:      int(total),
		TotalPages: totalPages,
	})
}

func (h *Handler) findCompletedMatches(ctx context.Context, playerID string, skip int64) ([]mongomodel.Match, error) {
	cursor, err := h.db.Collection(mongomodel.MatchesCollection).Find(
		ctx,
		completedMatchesFilter(playerID),
		options.Find().SetSort(bson.D{
			{Key: "completed_at", Value: -1},
			{Key: "created_at", Value: -1},
		}).SetSkip(skip).SetLimit(completedMatchesPerPage),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var matches []mongomodel.Match
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func (h *Handler) countCompletedMatches(ctx context.Context, playerID string) (int64, error) {
	return h.db.Collection(mongomodel.MatchesCollection).CountDocuments(
		ctx,
		completedMatchesFilter(playerID),
	)
}

func completedMatchesFilter(playerID string) bson.D {
	return bson.D{
		{Key: "status", Value: mongomodel.MatchStatusCompleted},
		{Key: "players.player_id", Value: playerID},
	}
}

func mapCompletedMatches(records []mongomodel.Match) []CompletedMatchResponse {
	matches := make([]CompletedMatchResponse, 0, len(records))
	for _, record := range records {
		matches = append(matches, CompletedMatchResponse{
			MatchID:       record.ID,
			Status:        record.Status,
			HostPlayerID:  record.HostPlayerID,
			Players:       mapCompletedMatchPlayers(record.Players),
			QuestionCount: len(record.QuestionIDs),
			CreatedAt:     record.CreatedAt,
			StartedAt:     optionalTime(record.StartedAt),
			CompletedAt:   optionalTime(record.CompletedAt),
		})
	}
	return matches
}

func mapCompletedMatchPlayers(records []mongomodel.MatchPlayer) []CompletedMatchPlayerResponse {
	players := make([]CompletedMatchPlayerResponse, 0, len(records))
	for _, record := range records {
		players = append(players, CompletedMatchPlayerResponse{
			PlayerID:  record.PlayerID,
			Nickname:  record.Nickname,
			SitoneIDs: cloneStringSlice(record.SitoneIDs),
			Score:     record.Score,
		})
	}
	return players
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func cloneStringSlice(input []string) []string {
	if input == nil {
		return nil
	}
	return append([]string(nil), input...)
}
