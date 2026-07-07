package me

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const maxNicknameRunes = 20

// UpdateNickname godoc
// @Summary Update current player nickname
// @Description Updates the authenticated player's display nickname.
// @Tags me
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body UpdateNicknameRequest true "Nickname update request"
// @Success 200 {object} UpdateNicknameResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/nickname [put]
func (h *Handler) UpdateNickname(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}

	var body UpdateNicknameRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	nickname, err := validateNickname(body.Nickname)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := h.savePlayerNickname(r.Context(), player.ID, nickname); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("nickname update failed", "me_nickname_save_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UpdateNicknameResponse{Nickname: nickname})
}

func validateNickname(value string) (string, error) {
	nickname := strings.TrimSpace(value)
	if nickname == "" {
		return "", httpx.BadRequest("nickname is required")
	}
	if utf8.RuneCountInString(nickname) > maxNicknameRunes {
		return "", httpx.BadRequest("nickname must be 20 characters or fewer")
	}
	return nickname, nil
}

func (h *Handler) savePlayerNickname(ctx context.Context, playerID string, nickname string) error {
	_, err := h.db.Collection(mongomodel.PlayersCollection).UpdateOne(
		ctx,
		bson.M{"_id": playerID},
		bson.M{"$set": bson.M{
			"nickname":   nickname,
			"updated_at": time.Now().UTC(),
		}},
	)
	return err
}
