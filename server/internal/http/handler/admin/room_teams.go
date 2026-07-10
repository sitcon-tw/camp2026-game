package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/roomteam"
)

const roomTeamMembershipJoinedByAdmin = "admin"

type RoomTeamResponse struct {
	RoomID           string     `json:"roomId" example:"room-208"`
	RoomNumber       string     `json:"roomNumber" example:"208"`
	MemberCount      int64      `json:"memberCount" example:"6"`
	QRTokenExpiresAt *time.Time `json:"qrTokenExpiresAt,omitempty"`
}

type RoomTeamsResponse struct {
	Rooms []RoomTeamResponse `json:"rooms"`
}

type RoomTeamTokenResponse struct {
	Room             RoomTeamResponse `json:"room"`
	QRToken          string           `json:"qrToken" example:"rmt_6H_x7lM20CK8BBnPfwEG1E"`
	QRTokenExpiresAt time.Time        `json:"qrTokenExpiresAt"`
}

type RoomTeamMemberTeamResponse struct {
	TeamID string `json:"teamId" example:"team-001"`
	Name   string `json:"name" example:"Team 001"`
}

type RoomTeamMemberResponse struct {
	PlayerID  string                      `json:"playerId" example:"7H9K2Q"`
	Nickname  string                      `json:"nickname" example:"Alice"`
	AvatarURL string                      `json:"avatarUrl,omitempty"`
	Team      *RoomTeamMemberTeamResponse `json:"team,omitempty"`
	JoinedAt  time.Time                   `json:"joinedAt"`
	JoinedBy  string                      `json:"joinedBy,omitempty" example:"scan"`
}

type RoomTeamMembersResponse struct {
	Room    RoomTeamResponse         `json:"room"`
	Members []RoomTeamMemberResponse `json:"members"`
}

type AddRoomTeamMemberRequest struct {
	PlayerID    string `json:"playerId,omitempty" validate:"omitempty,min=1,max=128" example:"7H9K2Q"`
	QRCodeToken string `json:"qrcodeToken,omitempty" validate:"omitempty,min=4,max=512" example:"qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok"`
}

type AddRoomTeamMemberResponse struct {
	Room   RoomTeamResponse       `json:"room"`
	Member RoomTeamMemberResponse `json:"member"`
	Added  bool                   `json:"added" example:"true"`
}

// ListRoomTeams godoc
// @Summary List dorm room teams as admin
// @Description Admin-only endpoint. Returns the configured dorm rooms and membership counters for hidden reward groups.
// @Tags admin
// @Produce json
// @Success 200 {object} RoomTeamsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/room-teams [get]
func (h *Handler) ListRoomTeams(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	rooms, err := h.ensureDefaultRoomTeams(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room teams unavailable", "admin_room_teams_seed_failed", err))
		return
	}
	responses, err := h.roomTeamResponses(r.Context(), rooms)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room teams unavailable", "admin_room_team_counts_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, RoomTeamsResponse{Rooms: responses})
}

// CreateRoomTeamToken godoc
// @Summary Create a dorm room QR token as admin
// @Description Admin-only endpoint. Rotates a short-lived QR token for a configured dorm room. The token expires after 10 minutes.
// @Tags admin
// @Produce json
// @Param roomNumber path string true "Dorm room number"
// @Success 201 {object} RoomTeamTokenResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/room-teams/{roomNumber}/token [post]
func (h *Handler) CreateRoomTeamToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	roomNumber := roomteam.NormalizeRoomNumber(chi.URLParam(r, "roomNumber"))
	if details := validateRoomNumber("path.roomNumber", roomNumber); len(details) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid room team token request", details...))
		return
	}

	room, qrToken, err := h.rotateRoomTeamToken(r.Context(), roomNumber, time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team token create failed", "admin_room_team_token_create_failed", err))
		return
	}
	response, err := h.roomTeamResponse(r.Context(), room)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team token unavailable", "admin_room_team_count_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, RoomTeamTokenResponse{
		Room:             response,
		QRToken:          qrToken,
		QRTokenExpiresAt: room.QRTokenExpiresAt,
	})
}

// ListRoomTeamMembers godoc
// @Summary List dorm room members as admin
// @Description Admin-only endpoint. Lists hidden room-team members for manual correction.
// @Tags admin
// @Produce json
// @Param roomNumber path string true "Dorm room number"
// @Success 200 {object} RoomTeamMembersResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/room-teams/{roomNumber}/members [get]
func (h *Handler) ListRoomTeamMembers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	room, ok := h.roomTeamFromRequest(w, r)
	if !ok {
		return
	}
	response, members, err := h.roomTeamMembersResponse(r.Context(), room)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team members unavailable", "admin_room_team_members_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, RoomTeamMembersResponse{
		Room:    response,
		Members: members,
	})
}

// AddRoomTeamMember godoc
// @Summary Add a dorm room member as admin
// @Description Admin-only endpoint. Adds a non-staff player to a hidden dorm room team by player ID or player QR code token.
// @Tags admin
// @Accept json
// @Produce json
// @Param roomNumber path string true "Dorm room number"
// @Param request body AddRoomTeamMemberRequest true "Room team member request"
// @Success 200 {object} AddRoomTeamMemberResponse
// @Success 201 {object} AddRoomTeamMemberResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/room-teams/{roomNumber}/members [post]
func (h *Handler) AddRoomTeamMember(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	room, ok := h.roomTeamFromRequest(w, r)
	if !ok {
		return
	}

	var body AddRoomTeamMemberRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.PlayerID = strings.TrimSpace(body.PlayerID)
	body.QRCodeToken = strings.TrimSpace(body.QRCodeToken)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if details := validateAddRoomTeamMemberRequest(body); len(details) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid room team member request", details...))
		return
	}

	player, err := h.findRoomTeamPlayer(r.Context(), body)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("player not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team member lookup failed", "admin_room_team_player_lookup_failed", err))
		return
	}
	if player.Role == authctx.PlayerRoleStaff {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid room team member request", httpx.ErrorDetail{
			Location: "body.playerId",
			Message:  "staff players cannot join dorm room teams",
		}))
		return
	}

	added, err := h.upsertRoomTeamMembership(r.Context(), room.ID, player.ID, roomTeamMembershipJoinedByAdmin)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team member add failed", "admin_room_team_member_add_failed", err))
		return
	}
	membership, err := h.findRoomTeamMembership(r.Context(), room.ID, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team member unavailable", "admin_room_team_member_lookup_failed", err))
		return
	}
	response, err := h.roomTeamResponse(r.Context(), room)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team unavailable", "admin_room_team_count_failed", err))
		return
	}
	member, err := h.roomTeamMemberResponse(r.Context(), membership)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team member unavailable", "admin_room_team_member_response_failed", err))
		return
	}

	status := http.StatusOK
	if added {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, AddRoomTeamMemberResponse{
		Room:   response,
		Member: member,
		Added:  added,
	})
}

// RemoveRoomTeamMember godoc
// @Summary Remove a dorm room member as admin
// @Description Admin-only endpoint. Removes a player from a hidden dorm room team.
// @Tags admin
// @Produce json
// @Param roomNumber path string true "Dorm room number"
// @Param playerID path string true "Player ID"
// @Success 204
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/room-teams/{roomNumber}/members/{playerID} [delete]
func (h *Handler) RemoveRoomTeamMember(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	room, ok := h.roomTeamFromRequest(w, r)
	if !ok {
		return
	}
	playerID := strings.TrimSpace(chi.URLParam(r, "playerID"))
	if playerID == "" {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid room team member remove request", httpx.ErrorDetail{
			Location: "path.playerId",
			Message:  "playerId is required",
		}))
		return
	}

	if err := h.deleteRoomTeamMembership(r.Context(), room.ID, playerID); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team member remove failed", "admin_room_team_member_remove_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) roomTeamFromRequest(w http.ResponseWriter, r *http.Request) (mongomodel.RoomTeam, bool) {
	roomNumber := roomteam.NormalizeRoomNumber(chi.URLParam(r, "roomNumber"))
	if details := validateRoomNumber("path.roomNumber", roomNumber); len(details) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid room team request", details...))
		return mongomodel.RoomTeam{}, false
	}

	room, err := h.ensureRoomTeam(r.Context(), roomNumber)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team unavailable", "admin_room_team_lookup_failed", err))
		return mongomodel.RoomTeam{}, false
	}
	return room, true
}

func validateRoomNumber(location string, roomNumber string) []httpx.ErrorDetail {
	if roomteam.ValidRoomNumber(roomNumber) {
		return nil
	}
	return []httpx.ErrorDetail{{
		Location: location,
		Message:  "roomNumber must be one of the configured dorm rooms",
	}}
}

func validateAddRoomTeamMemberRequest(body AddRoomTeamMemberRequest) []httpx.ErrorDetail {
	switch {
	case body.PlayerID == "" && body.QRCodeToken == "":
		return []httpx.ErrorDetail{{
			Location: "body.playerId",
			Message:  "playerId or qrcodeToken is required",
		}}
	case body.PlayerID != "" && body.QRCodeToken != "":
		return []httpx.ErrorDetail{{
			Location: "body.playerId",
			Message:  "playerId cannot be combined with qrcodeToken",
		}}
	default:
		return nil
	}
}

func (h *Handler) ensureDefaultRoomTeams(ctx context.Context) ([]mongomodel.RoomTeam, error) {
	for _, roomNumber := range roomteam.DefaultRoomNumbers() {
		if _, err := h.ensureRoomTeam(ctx, roomNumber); err != nil {
			return nil, err
		}
	}

	roomIDs := make([]string, 0, len(roomteam.DefaultRoomNumbers()))
	for _, roomNumber := range roomteam.DefaultRoomNumbers() {
		roomIDs = append(roomIDs, roomteam.RoomID(roomNumber))
	}
	rooms, err := findAllDashboard[mongomodel.RoomTeam](
		ctx,
		h.db,
		mongomodel.RoomTeamsCollection,
		bson.M{"_id": bson.M{"$in": roomIDs}},
	)
	if err != nil {
		return nil, err
	}
	sortRoomTeams(rooms)
	return rooms, nil
}

func (h *Handler) ensureRoomTeam(ctx context.Context, roomNumber string) (mongomodel.RoomTeam, error) {
	now := time.Now().UTC()
	room := mongomodel.RoomTeam{
		ID:         roomteam.RoomID(roomNumber),
		RoomNumber: roomNumber,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err := h.db.Collection(mongomodel.RoomTeamsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": room.ID},
		bson.M{"$setOnInsert": room},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&room)
	return room, err
}

func (h *Handler) rotateRoomTeamToken(ctx context.Context, roomNumber string, now time.Time) (mongomodel.RoomTeam, string, error) {
	qrToken, err := roomteam.NewQRToken()
	if err != nil {
		return mongomodel.RoomTeam{}, "", err
	}

	roomID := roomteam.RoomID(roomNumber)
	var room mongomodel.RoomTeam
	err = h.db.Collection(mongomodel.RoomTeamsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": roomID},
		bson.M{
			"$set": bson.M{
				"qr_token":            qrToken,
				"qr_token_expires_at": now.Add(roomteam.TokenTTL),
				"updated_at":          now,
			},
			"$setOnInsert": bson.M{
				"_id":         roomID,
				"room_number": roomNumber,
				"created_at":  now,
			},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&room)
	if err != nil {
		return mongomodel.RoomTeam{}, "", err
	}
	return room, qrToken, nil
}

func sortRoomTeams(rooms []mongomodel.RoomTeam) {
	order := make(map[string]int)
	for index, roomNumber := range roomteam.DefaultRoomNumbers() {
		order[roomteam.RoomID(roomNumber)] = index
	}
	slices.SortFunc(rooms, func(a, b mongomodel.RoomTeam) int {
		return order[a.ID] - order[b.ID]
	})
}

func (h *Handler) roomTeamResponses(ctx context.Context, rooms []mongomodel.RoomTeam) ([]RoomTeamResponse, error) {
	responses := make([]RoomTeamResponse, 0, len(rooms))
	for _, room := range rooms {
		response, err := h.roomTeamResponse(ctx, room)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func (h *Handler) roomTeamResponse(ctx context.Context, room mongomodel.RoomTeam) (RoomTeamResponse, error) {
	count, err := h.db.Collection(mongomodel.RoomTeamMembershipsCollection).CountDocuments(ctx, bson.M{"room_team_id": room.ID})
	if err != nil {
		return RoomTeamResponse{}, err
	}
	response := RoomTeamResponse{
		RoomID:      room.ID,
		RoomNumber:  room.RoomNumber,
		MemberCount: count,
	}
	if room.QRToken != "" && room.QRTokenExpiresAt.After(time.Now().UTC()) {
		response.QRTokenExpiresAt = &room.QRTokenExpiresAt
	}
	return response, nil
}

func (h *Handler) roomTeamMembersResponse(ctx context.Context, room mongomodel.RoomTeam) (RoomTeamResponse, []RoomTeamMemberResponse, error) {
	roomResponse, err := h.roomTeamResponse(ctx, room)
	if err != nil {
		return RoomTeamResponse{}, nil, err
	}
	memberships, err := h.findRoomTeamMemberships(ctx, room.ID)
	if err != nil {
		return RoomTeamResponse{}, nil, err
	}

	responses := make([]RoomTeamMemberResponse, 0, len(memberships))
	for _, membership := range memberships {
		member, err := h.roomTeamMemberResponse(ctx, membership)
		if err != nil {
			return RoomTeamResponse{}, nil, err
		}
		responses = append(responses, member)
	}
	return roomResponse, responses, nil
}

func (h *Handler) findRoomTeamMemberships(ctx context.Context, roomTeamID string) ([]mongomodel.RoomTeamMembership, error) {
	return findAllDashboard[mongomodel.RoomTeamMembership](
		ctx,
		h.db,
		mongomodel.RoomTeamMembershipsCollection,
		bson.M{"room_team_id": roomTeamID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}),
	)
}

func (h *Handler) findRoomTeamMembership(ctx context.Context, roomTeamID string, playerID string) (mongomodel.RoomTeamMembership, error) {
	var membership mongomodel.RoomTeamMembership
	err := h.db.Collection(mongomodel.RoomTeamMembershipsCollection).FindOne(ctx, bson.M{
		"room_team_id": roomTeamID,
		"player_id":    playerID,
	}).Decode(&membership)
	return membership, err
}

func (h *Handler) findRoomTeamPlayer(ctx context.Context, body AddRoomTeamMemberRequest) (mongomodel.Player, error) {
	var player mongomodel.Player
	filter := bson.M{"_id": body.PlayerID}
	if body.QRCodeToken != "" {
		filter = bson.M{"qrcode_token": body.QRCodeToken}
	}
	err := h.db.Collection(mongomodel.PlayersCollection).FindOne(ctx, filter).Decode(&player)
	return player, err
}

func (h *Handler) roomTeamMemberResponse(ctx context.Context, membership mongomodel.RoomTeamMembership) (RoomTeamMemberResponse, error) {
	player, err := h.findPlayerByID(ctx, membership.PlayerID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return RoomTeamMemberResponse{
			PlayerID: membership.PlayerID,
			JoinedAt: membership.CreatedAt,
			JoinedBy: membership.JoinedBy,
		}, nil
	}
	if err != nil {
		return RoomTeamMemberResponse{}, err
	}

	response := RoomTeamMemberResponse{
		PlayerID:  player.ID,
		Nickname:  player.Nickname,
		AvatarURL: player.AvatarURL,
		JoinedAt:  membership.CreatedAt,
		JoinedBy:  membership.JoinedBy,
	}
	if player.TeamID != "" {
		team, err := h.findTeamByID(ctx, player.TeamID)
		if err == nil {
			response.Team = &RoomTeamMemberTeamResponse{
				TeamID: team.ID,
				Name:   team.Name,
			}
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RoomTeamMemberResponse{}, err
		}
	}
	return response, nil
}

func (h *Handler) findPlayerByID(ctx context.Context, playerID string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).FindOne(ctx, bson.M{"_id": playerID}).Decode(&player)
	return player, err
}

func (h *Handler) findTeamByID(ctx context.Context, teamID string) (mongomodel.Team, error) {
	var team mongomodel.Team
	err := h.db.Collection(mongomodel.TeamsCollection).FindOne(ctx, bson.M{"_id": teamID}).Decode(&team)
	return team, err
}

func (h *Handler) upsertRoomTeamMembership(ctx context.Context, roomTeamID string, playerID string, joinedBy string) (bool, error) {
	now := time.Now().UTC()
	removeResult, err := h.db.Collection(mongomodel.RoomTeamMembershipsCollection).DeleteMany(
		ctx,
		bson.M{
			"player_id":    playerID,
			"room_team_id": bson.M{"$ne": roomTeamID},
		},
	)
	if err != nil {
		return false, err
	}
	result, err := h.db.Collection(mongomodel.RoomTeamMembershipsCollection).UpdateOne(
		ctx,
		bson.M{
			"room_team_id": roomTeamID,
			"player_id":    playerID,
		},
		bson.M{"$setOnInsert": mongomodel.RoomTeamMembership{
			ID:         newAdminID("room_team_membership"),
			RoomTeamID: roomTeamID,
			PlayerID:   playerID,
			JoinedBy:   joinedBy,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return false, err
	}
	return removeResult.DeletedCount > 0 || result.UpsertedCount == 1, nil
}

func (h *Handler) deleteRoomTeamMembership(ctx context.Context, roomTeamID string, playerID string) error {
	_, err := h.db.Collection(mongomodel.RoomTeamMembershipsCollection).DeleteOne(ctx, bson.M{
		"room_team_id": roomTeamID,
		"player_id":    playerID,
	})
	return err
}

func newAdminID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
