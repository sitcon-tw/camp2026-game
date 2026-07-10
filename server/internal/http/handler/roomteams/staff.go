package roomteams

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/roomteam"
)

type StaffRoomTeamResponse struct {
	RoomID     string `json:"roomId" example:"room-208"`
	RoomNumber string `json:"roomNumber" example:"208"`
}

type StaffRoomTeamsResponse struct {
	Rooms []StaffRoomTeamResponse `json:"rooms"`
}

type StaffRoomTeamTokenResponse struct {
	Room             StaffRoomTeamResponse `json:"room"`
	QRToken          string                `json:"qrToken" example:"rmt_6H_x7lM20CK8BBnPfwEG1E"`
	QRTokenExpiresAt time.Time             `json:"qrTokenExpiresAt"`
}

type AddStaffRoomTeamMemberRequest struct {
	PlayerID    string `json:"playerId,omitempty" validate:"omitempty,min=1,max=128" example:"7H9K2Q"`
	QRCodeToken string `json:"qrcodeToken,omitempty" validate:"omitempty,min=4,max=512" example:"qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok"`
}

type AddStaffRoomTeamMemberResponse struct {
	Room   StaffRoomTeamResponse `json:"room"`
	Player StaffRoomTeamPlayer   `json:"player"`
	Added  bool                  `json:"added" example:"true"`
}

type StaffRoomTeamPlayer struct {
	PlayerID  string `json:"playerId" example:"7H9K2Q"`
	Nickname  string `json:"nickname" example:"Alice"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
}

// ListStaffRoomTeams godoc
// @Summary List available dorm rooms for staff QR codes
// @Description Staff-only endpoint. Returns the configured dorm rooms that staff may generate a join QR code for.
// @Tags staff
// @Produce json
// @Success 200 {object} StaffRoomTeamsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Router /staff/room-teams [get]
func (h *Handler) ListStaffRoomTeams(w http.ResponseWriter, r *http.Request) {
	rooms := roomteam.DefaultRoomNumbers()
	response := StaffRoomTeamsResponse{Rooms: make([]StaffRoomTeamResponse, 0, len(rooms))}
	for _, roomNumber := range rooms {
		response.Rooms = append(response.Rooms, StaffRoomTeamResponse{
			RoomID:     roomteam.RoomID(roomNumber),
			RoomNumber: roomNumber,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// CreateStaffRoomTeamToken godoc
// @Summary Create a dorm room QR token as staff
// @Description Staff-only endpoint. Rotates a short-lived join QR code for a configured dorm room. The token expires after 10 minutes.
// @Tags staff
// @Produce json
// @Param roomNumber path string true "Dorm room number"
// @Success 201 {object} StaffRoomTeamTokenResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/room-teams/{roomNumber}/token [post]
func (h *Handler) CreateStaffRoomTeamToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireDatabase(w, r) {
		return
	}

	roomNumber := roomteam.NormalizeRoomNumber(chi.URLParam(r, "roomNumber"))
	if !roomteam.ValidRoomNumber(roomNumber) {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid dorm room", httpx.ErrorDetail{
			Location: "path.roomNumber",
			Message:  "roomNumber must be one of the configured dorm rooms",
		}))
		return
	}

	room, qrToken, err := h.rotateRoomTeamToken(r.Context(), roomNumber, time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team token create failed", "staff_room_team_token_create_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, StaffRoomTeamTokenResponse{
		Room:             StaffRoomTeamResponse{RoomID: room.ID, RoomNumber: room.RoomNumber},
		QRToken:          qrToken,
		QRTokenExpiresAt: room.QRTokenExpiresAt,
	})
}

// AddStaffRoomTeamMember godoc
// @Summary Assign a player to a dorm room as staff
// @Description Staff-only endpoint. Moves a non-staff player selected by player ID or QR code identifier into a configured dorm room.
// @Tags staff
// @Accept json
// @Produce json
// @Param roomNumber path string true "Dorm room number"
// @Param request body AddStaffRoomTeamMemberRequest true "Dorm room member request"
// @Success 200 {object} AddStaffRoomTeamMemberResponse
// @Success 201 {object} AddStaffRoomTeamMemberResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /staff/room-teams/{roomNumber}/members [post]
func (h *Handler) AddStaffRoomTeamMember(w http.ResponseWriter, r *http.Request) {
	if !h.requireDatabase(w, r) {
		return
	}

	roomNumber := roomteam.NormalizeRoomNumber(chi.URLParam(r, "roomNumber"))
	if !roomteam.ValidRoomNumber(roomNumber) {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid dorm room", httpx.ErrorDetail{
			Location: "path.roomNumber",
			Message:  "roomNumber must be one of the configured dorm rooms",
		}))
		return
	}

	var body AddStaffRoomTeamMemberRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.QRCodeToken = strings.TrimSpace(body.QRCodeToken)
	body.PlayerID = strings.TrimSpace(body.PlayerID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if details := validateAddStaffRoomTeamMemberRequest(body); len(details) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid dorm room member", details...))
		return
	}

	player, err := h.findStaffRoomTeamPlayer(r.Context(), body)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("player not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team player lookup failed", "staff_room_team_player_lookup_failed", err))
		return
	}

	room, err := h.ensureRoomTeam(r.Context(), roomNumber)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team unavailable", "staff_room_team_lookup_failed", err))
		return
	}
	added, err := h.upsertRoomTeamMembership(r.Context(), room.ID, player.ID, "staff")
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team member assignment failed", "staff_room_team_member_add_failed", err))
		return
	}

	status := http.StatusOK
	if added {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, AddStaffRoomTeamMemberResponse{
		Room: StaffRoomTeamResponse{RoomID: room.ID, RoomNumber: room.RoomNumber},
		Player: StaffRoomTeamPlayer{
			PlayerID:  player.ID,
			Nickname:  player.Nickname,
			AvatarURL: player.AvatarURL,
		},
		Added: added,
	})
}

func validateAddStaffRoomTeamMemberRequest(body AddStaffRoomTeamMemberRequest) []httpx.ErrorDetail {
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

func (h *Handler) findStaffRoomTeamPlayer(ctx context.Context, body AddStaffRoomTeamMemberRequest) (mongomodel.Player, error) {
	if body.PlayerID != "" {
		return h.findPlayerByID(ctx, body.PlayerID)
	}
	return h.findPlayerByQRCodeToken(ctx, body.QRCodeToken)
}

func (h *Handler) findPlayerByID(ctx context.Context, playerID string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(ctx, bson.M{"_id": playerID}).
		Decode(&player)
	return player, err
}

func (h *Handler) findPlayerByQRCodeToken(ctx context.Context, token string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(ctx, bson.M{"qrcode_token": token}).
		Decode(&player)
	return player, err
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
