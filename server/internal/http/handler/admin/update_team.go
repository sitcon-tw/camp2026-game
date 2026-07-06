package admin

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	teamIDMaxLen        = 128
	teamNameMaxLen      = 64
	teamAvatarURLMaxLen = 512
)

type UpdateTeamRequest struct {
	Name      string `json:"name" example:"Blue Team"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/blue.png"`
}

type UpdateTeamResponse struct {
	TeamID    string `json:"teamId" example:"8M4RXP"`
	Name      string `json:"name" example:"Blue Team"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/blue.png"`
}

// UpdateTeam godoc
// @Summary Update a team as admin
// @Description Admin-only endpoint. Updates the team display name and avatar URL shown in operations dashboards.
// @Tags admin
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param request body UpdateTeamRequest true "Team update request"
// @Success 200 {object} UpdateTeamResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/teams/{teamID} [put]
func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	teamID := strings.TrimSpace(chi.URLParam(r, "teamID"))
	var body UpdateTeamRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body = normalizeUpdateTeamRequest(body)
	if details := validateUpdateTeamRequest(teamID, body); len(details) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid team update request", details...))
		return
	}

	team, err := h.updateTeam(r.Context(), teamID, body)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			httpx.WriteProblem(w, r, httpx.NotFound("team not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("team update failed", "admin_team_update_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updateTeamResponse(team))
}

func normalizeUpdateTeamRequest(body UpdateTeamRequest) UpdateTeamRequest {
	body.Name = strings.TrimSpace(body.Name)
	body.AvatarURL = strings.TrimSpace(body.AvatarURL)
	return body
}

func validateUpdateTeamRequest(teamID string, body UpdateTeamRequest) []httpx.ErrorDetail {
	details := make([]httpx.ErrorDetail, 0, 3)
	if teamID == "" {
		details = append(details, httpx.ErrorDetail{Location: "path.teamId", Message: "teamId is required"})
	} else if len([]rune(teamID)) > teamIDMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "path.teamId", Message: "teamId must be at most 128"})
	}

	if body.Name == "" {
		details = append(details, httpx.ErrorDetail{Location: "body.name", Message: "name is required"})
	} else if len([]rune(body.Name)) > teamNameMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "body.name", Message: "name must be at most 64"})
	}

	if len([]rune(body.AvatarURL)) > teamAvatarURLMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "body.avatarUrl", Message: "avatarUrl must be at most 512"})
	} else if !validTeamAvatarURL(body.AvatarURL) {
		details = append(details, httpx.ErrorDetail{Location: "body.avatarUrl", Message: "avatarUrl must be an http(s) URL or root-relative path"})
	}
	return details
}

func validTeamAvatarURL(value string) bool {
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func (h *Handler) updateTeam(ctx context.Context, teamID string, body UpdateTeamRequest) (mongomodel.Team, error) {
	setFields := bson.D{{Key: "name", Value: body.Name}}
	update := bson.D{}
	if body.AvatarURL == "" {
		update = append(update,
			bson.E{Key: "$set", Value: setFields},
			bson.E{Key: "$unset", Value: bson.D{{Key: "avatar_url", Value: ""}}},
		)
	} else {
		setFields = append(setFields, bson.E{Key: "avatar_url", Value: body.AvatarURL})
		update = append(update, bson.E{Key: "$set", Value: setFields})
	}

	var team mongomodel.Team
	err := h.db.Collection(mongomodel.TeamsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": teamID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&team)
	if err != nil {
		return mongomodel.Team{}, err
	}
	return team, nil
}

func updateTeamResponse(team mongomodel.Team) UpdateTeamResponse {
	return UpdateTeamResponse{
		TeamID:    team.ID,
		Name:      team.Name,
		AvatarURL: team.AvatarURL,
	}
}
