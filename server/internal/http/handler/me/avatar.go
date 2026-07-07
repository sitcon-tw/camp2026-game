package me

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// UpdateAvatar godoc
// @Summary Update current player avatar
// @Description Updates the authenticated player's avatar to an owned sitone icon, or clears it to restore the default generated avatar.
// @Tags me
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body UpdateAvatarRequest true "Avatar update request"
// @Success 200 {object} UpdateAvatarResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/avatar [put]
func (h *Handler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
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

	var body UpdateAvatarRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	avatarURL, err := h.avatarURLForRequest(r.Context(), player.ID, body)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := h.savePlayerAvatar(r.Context(), player.ID, avatarURL); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("avatar update failed", "me_avatar_save_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UpdateAvatarResponse{AvatarURL: avatarURL})
}

func (h *Handler) avatarURLForRequest(ctx context.Context, playerID string, body UpdateAvatarRequest) (string, error) {
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

	owned, err := h.ownedSitoneCounts(ctx, playerID)
	if err != nil {
		return "", httpx.InternalServerError("avatar unavailable", "me_avatar_inventory_lookup_failed", err)
	}
	if owned[sitoneID] <= 0 {
		return "", httpx.BadRequest("sitone is not owned")
	}

	return sitone.IconPath, nil
}

func (h *Handler) savePlayerAvatar(ctx context.Context, playerID string, avatarURL string) error {
	update := bson.M{
		"$set": bson.M{"updated_at": time.Now().UTC()},
	}
	if avatarURL == "" {
		update["$unset"] = bson.M{"avatar_url": ""}
	} else {
		update["$set"].(bson.M)["avatar_url"] = avatarURL
	}

	_, err := h.db.Collection(mongomodel.PlayersCollection).UpdateOne(
		ctx,
		bson.M{"_id": playerID},
		update,
	)
	return err
}
