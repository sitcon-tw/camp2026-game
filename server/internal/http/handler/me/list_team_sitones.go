package me

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ListTeamSitones godoc
// @Summary List current player's team sitones
// @Description Returns the union of sitones owned by members of the authenticated player's team with catalog definitions.
// @Tags me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} SitoneListResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/team/sitones [get]
func (h *Handler) ListTeamSitones(w http.ResponseWriter, r *http.Request) {
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
	if strings.TrimSpace(player.TeamID) == "" {
		httpx.WriteProblem(w, r, httpx.BadRequest("player has no team"))
		return
	}

	counts, err := h.teamOwnedSitoneCounts(r.Context(), player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("team sitones unavailable", "me_team_sitones_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, SitoneListResponse{
		Sitones: mapTeamSitones(h.content, counts),
	})
}

func (h *Handler) teamOwnedSitoneCounts(ctx context.Context, teamID string) (map[string]int, error) {
	memberIDs, err := h.teamMemberIDs(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return map[string]int{}, nil
	}

	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": bson.M{"$in": memberIDs}, "quantity": bson.M{"$gt": 0}},
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var record mongomodel.PlayerSitone
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		if record.SitoneID != "" {
			counts[record.SitoneID] += record.Quantity
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func (h *Handler) teamMemberIDs(ctx context.Context, teamID string) ([]string, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": strings.TrimSpace(teamID)},
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	ids := make([]string, 0)
	for cursor.Next(ctx) {
		var member mongomodel.Player
		if err := cursor.Decode(&member); err != nil {
			return nil, err
		}
		if strings.TrimSpace(member.ID) != "" {
			ids = append(ids, member.ID)
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func mapTeamSitones(store interface {
	GetSitone(string) (content.Sitone, bool)
}, counts map[string]int) []PlayerSitoneResponse {
	ids := make([]string, 0, len(counts))
	for sitoneID, count := range counts {
		if count > 0 {
			ids = append(ids, sitoneID)
		}
	}
	sort.Strings(ids)

	out := make([]PlayerSitoneResponse, 0, len(ids))
	for _, sitoneID := range ids {
		sitone, ok := store.GetSitone(sitoneID)
		if !ok {
			continue
		}
		out = append(out, PlayerSitoneResponse{
			ID:       sitoneID,
			SitoneID: sitoneID,
			Quantity: counts[sitoneID],
			Sitone:   sitoneResponse(sitone),
		})
	}
	return out
}
