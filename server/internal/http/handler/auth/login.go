package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

// Login godoc
// @Summary User login with auth token
// @Description User-facing endpoint. Validates the issued game URL token, rotates it, and writes the fresh session token to the camp2026_auth cookie.
// @Tags User Authentication
// @Accept json
// @Produce json
// @Param request body apimodel.AuthLoginRequest true "Auth login request"
// @Success 200 {object} apimodel.AuthLoginResponse
// @Header 200 {string} Set-Cookie "camp2026_auth=<auth-token>; Path=/; HttpOnly; Secure; SameSite=Strict"
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
		httpx.WriteProblem(w, r, httpx.InternalServerError("login failed", "login_player_lookup_failed", err))
		return
	}
	teamID := playerTeamID(player)
	if player.ID == "" || player.Nickname == "" || (player.Role != authctx.PlayerRoleStaff && teamID == "") {
		httpx.WriteProblem(w, r, httpx.InternalServerError("login failed", "login_player_invalid", errors.New("player record is missing required login fields")))
		return
	}

	var team *mongomodel.Team
	if teamID != "" {
		foundTeam, err := h.findTeam(r.Context(), teamID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("login failed", "login_team_lookup_failed", err))
			return
		}
		team = &foundTeam
	}

	openPower, err := h.sumOpenPower(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("login failed", "login_open_power_sum_failed", err))
		return
	}

	sessionToken, err := h.rotateAuthToken(r.Context(), player.ID, body.Token)
	if errors.Is(err, errAuthTokenStale) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "invalid auth token"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("login failed", "login_auth_token_rotate_failed", err))
		return
	}

	setAuthCookie(w, sessionToken)
	httpx.WriteJSON(w, http.StatusOK, loginResponse(player, team, openPower))
}

func playerTeamID(player mongomodel.Player) string {
	return player.TeamID
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, authCookie(token, 0))
}

func authCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     authctx.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
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
	return openpower.TotalForPlayer(ctx, h.db, playerID)
}

func openPowerTotalFromCursor(ctx context.Context, cursor *mongo.Cursor) (int, error) {
	return openpower.TotalFromCursor(ctx, cursor)
}

func openPowerTotalPipeline(playerID string) mongo.Pipeline {
	return openpower.TotalPipeline(playerID)
}

func loginResponse(player mongomodel.Player, team *mongomodel.Team, openPower int) apimodel.AuthLoginResponse {
	response := apimodel.AuthLoginResponse{
		Player: apimodel.AuthPlayerSummary{
			PlayerID:  player.ID,
			Nickname:  player.Nickname,
			OpenPower: openPower,
			AvatarURL: player.AvatarURL,
			Role:      player.Role,
		},
	}
	if team != nil {
		response.Player.Team = &apimodel.AuthTeamSummary{
			TeamID: team.ID,
			Name:   team.Name,
		}
	}
	return response
}
