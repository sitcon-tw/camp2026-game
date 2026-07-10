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
)

const roomTeamMembershipJoinedByScan = "scan"

type RoomTeamResponse struct {
	RoomID     string `json:"roomId" example:"room-208"`
	RoomNumber string `json:"roomNumber" example:"208"`
}

type JoinRoomTeamResponse struct {
	Room   RoomTeamResponse `json:"room"`
	Joined bool             `json:"joined" example:"true"`
}

// ScanJoin godoc
// @Summary Join dorm room team by QR token
// @Description Joins the authenticated non-staff player to the dorm room represented by a short-lived QR token. The player's official game team is not changed.
// @Tags room-teams
// @Produce json
// @Security AuthCookieAuth
// @Param qrToken path string true "Opaque room team QR token"
// @Success 200 {object} JoinRoomTeamResponse
// @Success 201 {object} JoinRoomTeamResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /room-teams/scans/{qrToken}/join [post]
func (h *Handler) ScanJoin(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	qrToken := strings.TrimSpace(chi.URLParam(r, "qrToken"))
	room, err := h.findActiveRoomTeamByQRToken(r.Context(), qrToken, time.Now().UTC())
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("room team token not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team token unavailable", "room_team_token_lookup_failed", err))
		return
	}

	joined, err := h.upsertRoomTeamMembership(r.Context(), room.ID, player.ID, roomTeamMembershipJoinedByScan)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("room team join failed", "room_team_join_failed", err))
		return
	}

	status := http.StatusOK
	if joined {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, JoinRoomTeamResponse{
		Room: RoomTeamResponse{
			RoomID:     room.ID,
			RoomNumber: room.RoomNumber,
		},
		Joined: joined,
	})
}

func (h *Handler) findActiveRoomTeamByQRToken(ctx context.Context, qrToken string, now time.Time) (mongomodel.RoomTeam, error) {
	var room mongomodel.RoomTeam
	err := h.db.Collection(mongomodel.RoomTeamsCollection).FindOne(ctx, bson.M{
		"qr_token":            qrToken,
		"qr_token_expires_at": bson.M{"$gt": now},
	}).Decode(&room)
	return room, err
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
			ID:         newID("room_team_membership"),
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

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
