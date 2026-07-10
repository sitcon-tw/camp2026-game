package me

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/achievement"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ListAchievements godoc
// @Summary List current player achievements
// @Description Returns all codex achievements with the authenticated player's unlock state.
// @Tags me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} AchievementListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/achievements [get]
func (h *Handler) ListAchievements(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}
	if h.content == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("content store is unavailable"))
		return
	}

	collectedCount, err := h.reconcileCodexAchievements(r.Context(), player.ID, time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("achievements unavailable", "me_achievements_reconcile_failed", err))
		return
	}

	cursor, err := h.db.Collection(mongomodel.AchievementsCollection).Find(
		r.Context(),
		bson.M{"player_id": player.ID},
		options.Find().SetSort(bson.D{{Key: "sort_order", Value: 1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("achievements unavailable", "me_achievements_lookup_failed", err))
		return
	}
	defer func() { _ = cursor.Close(r.Context()) }()

	var records []mongomodel.Achievement
	if err := cursor.All(r.Context(), &records); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("achievements unavailable", "me_achievements_decode_failed", err))
		return
	}

	unlocked := make(map[string]mongomodel.Achievement, len(records))
	for _, record := range records {
		unlocked[record.Key] = record
	}
	definitions := achievement.CodexDefinitions(len(h.content.ListSitones()))
	responses, unlockedCount := mapAchievementResponses(definitions, unlocked)

	httpx.WriteJSON(w, http.StatusOK, AchievementListResponse{
		Achievements:         responses,
		CollectedSitoneCount: collectedCount,
		TotalSitoneCount:     len(h.content.ListSitones()),
		UnlockedCount:        unlockedCount,
	})
}

func mapAchievementResponses(definitions []achievement.Definition, unlocked map[string]mongomodel.Achievement) ([]AchievementResponse, int) {
	responses := make([]AchievementResponse, 0, len(definitions))
	unlockedCount := 0
	for _, definition := range definitions {
		response := AchievementResponse{
			Key:                 definition.Key,
			Name:                definition.Name,
			Tier:                definition.Tier,
			RequiredSitoneCount: definition.RequiredSitoneCount,
			OpenPowerReward:     definition.OpenPowerReward,
		}
		if record, ok := unlocked[definition.Key]; ok {
			response.Unlocked = true
			unlockedCount++
			unlockedAt := record.CreatedAt
			response.UnlockedAt = &unlockedAt
		}
		responses = append(responses, response)
	}
	return responses, unlockedCount
}
