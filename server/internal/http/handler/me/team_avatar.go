package me

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// UpdateTeamAvatar godoc
// @Summary Update current player's team avatar
// @Description Updates the authenticated player's team avatar to any sitone catalog icon, or clears it to restore the default team avatar.
// @Tags me
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body UpdateTeamAvatarRequest true "Team avatar update request"
// @Success 200 {object} UpdateTeamAvatarResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/team/avatar [put]
func (h *Handler) UpdateTeamAvatar(w http.ResponseWriter, r *http.Request) {
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

	var body UpdateTeamAvatarRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	avatarURL, err := h.teamAvatarURLForRequest(body)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	team, err := h.saveTeamAvatar(r.Context(), player.TeamID, avatarURL)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			httpx.WriteProblem(w, r, httpx.BadRequest("player team not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("team avatar update failed", "me_team_avatar_save_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UpdateTeamAvatarResponse{Team: teamResponse(team)})
}

func (h *Handler) teamAvatarURLForRequest(body UpdateTeamAvatarRequest) (string, error) {
	if body.SitoneID == nil {
		return "", nil
	}

	sitoneID := strings.TrimSpace(*body.SitoneID)
	if sitoneID == "" {
		return "", nil
	}

	sitone, ok := h.content.GetSitone(sitoneID)
	if !ok {
		return "", httpx.BadRequest("unknown sitone")
	}
	if strings.TrimSpace(sitone.IconPath) == "" {
		return "", httpx.BadRequest("sitone has no avatar icon")
	}
	return sitone.IconPath, nil
}

func (h *Handler) saveTeamAvatar(ctx context.Context, teamID string, avatarURL string) (mongomodel.Team, error) {
	update := bson.M{
		"$set": bson.M{"updated_at": time.Now().UTC()},
	}
	if avatarURL == "" {
		update["$unset"] = bson.M{"avatar_url": ""}
	} else {
		update["$set"].(bson.M)["avatar_url"] = avatarURL
	}

	var team mongomodel.Team
	err := h.db.Collection(mongomodel.TeamsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": strings.TrimSpace(teamID)},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&team)
	if err != nil {
		return mongomodel.Team{}, err
	}
	return team, nil
}
