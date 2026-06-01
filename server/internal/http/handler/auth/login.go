package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Login godoc
// @Summary User login with auth token
// @Description User-facing endpoint. Validates the stable auth token from the issued game URL and writes it to the camp2026_auth cookie.
// @Tags User Authentication
// @Accept json
// @Produce json
// @Param request body apimodel.AuthLoginRequest true "Auth login request"
// @Success 200 {object} apimodel.AuthLoginResponse
// @Header 200 {string} Set-Cookie "camp2026_auth=<auth-token>; Path=/; HttpOnly; Secure; SameSite=Lax"
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body apimodel.AuthLoginRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.Token = strings.TrimSpace(body.Token)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}

	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(r.Context(), bson.M{"auth_token": body.Token}).
		Decode(&player)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "invalid auth token"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "login failed"))
		return
	}
	if player.ID == "" || player.Nickname == "" || player.TeamID == "" {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "login failed"))
		return
	}

	team, err := h.findTeam(r.Context(), player.TeamID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "login failed"))
		return
	}

	openPower, err := h.sumOpenPower(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusInternalServerError, "login failed"))
		return
	}

	setAuthCookie(w, body.Token)
	httpx.WriteJSON(w, http.StatusOK, loginResponse(player, team, openPower))
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "camp2026_auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) findTeam(ctx context.Context, teamID string) (mongomodel.Team, error) {
	var team mongomodel.Team
	err := h.db.Collection(mongomodel.TeamsCollection).
		FindOne(ctx, bson.M{"_id": teamID}).
		Decode(&team)
	if err != nil {
		return mongomodel.Team{}, err
	}
	if team.ID == "" || team.Name == "" {
		return mongomodel.Team{}, mongo.ErrNoDocuments
	}
	return team, nil
}

func (h *Handler) sumOpenPower(ctx context.Context, playerID string) (int, error) {
	cursor, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).
		Aggregate(ctx, openPowerTotalPipeline(playerID))
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	return openPowerTotalFromCursor(ctx, cursor)
}

func openPowerTotalFromCursor(ctx context.Context, cursor *mongo.Cursor) (int, error) {
	var totals []struct {
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &totals); err != nil {
		return 0, err
	}
	if len(totals) == 0 {
		return 0, nil
	}
	return totals[0].Total, nil
}

func openPowerTotalPipeline(playerID string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: playerID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	}
}

func loginResponse(player mongomodel.Player, team mongomodel.Team, openPower int) apimodel.AuthLoginResponse {
	return apimodel.AuthLoginResponse{
		Player: apimodel.AuthPlayerSummary{
			PlayerID:  player.ID,
			Nickname:  player.Nickname,
			OpenPower: openPower,
			AvatarURL: player.AvatarURL,
			Team: apimodel.AuthTeamSummary{
				TeamID: team.ID,
				Name:   team.Name,
			},
		},
	}
}
