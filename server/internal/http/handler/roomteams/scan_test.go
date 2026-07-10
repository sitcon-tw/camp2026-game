package roomteams

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestScanJoinAllowsStaffPlayer(t *testing.T) {
	token := "rmt_stafftoken123"
	db := startRoomTeamsMockDatabase(t,
		createRoomTeamsCursorResponse("camp2026_game_test.room_teams", bson.D{
			{Key: "_id", Value: "room-208"},
			{Key: "room_number", Value: "208"},
			{Key: "qr_token", Value: token},
			{Key: "qr_token_expires_at", Value: time.Now().Add(time.Hour)},
			{Key: "created_at", Value: time.Now()},
			{Key: "updated_at", Value: time.Now()},
		}),
		deleteRoomTeamsResponse(0),
		updateRoomTeamsResponse("room_team_membership_staff"),
	)

	router := chi.NewRouter()
	New(Dependencies{MongoDB: db}).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/room-teams/scans/"+token+"/join", nil)
	req = req.WithContext(authctx.WithPlayer(req.Context(), mongomodel.Player{
		ID:   "staff-a",
		Role: authctx.PlayerRoleStaff,
	}))
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, res.Code, res.Body.String())
	}
	var body JoinRoomTeamResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Joined || body.Room.RoomNumber != "208" {
		t.Fatalf("unexpected join response: %#v", body)
	}
}

func startRoomTeamsMockDatabase(t *testing.T, responses ...bson.D) *mongo.Database {
	t.Helper()

	deployment := drivertest.NewMockDeployment(responses...)
	clientOptions := options.Client()
	clientOptions.Deployment = deployment

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		t.Fatalf("connect mock mongodb client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(t.Context())
	})

	return client.Database("camp2026_game_test")
}

func createRoomTeamsCursorResponse(ns string, batch ...bson.D) bson.D {
	values := make(bson.A, 0, len(batch))
	for _, doc := range batch {
		values = append(values, doc)
	}
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "cursor", Value: bson.D{
			{Key: "id", Value: int64(0)},
			{Key: "ns", Value: ns},
			{Key: "firstBatch", Value: values},
		}},
	}
}

func deleteRoomTeamsResponse(deleted int32) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: deleted},
	}
}

func updateRoomTeamsResponse(upsertedID string) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: int32(1)},
		{Key: "nModified", Value: int32(0)},
		{Key: "upserted", Value: bson.A{bson.D{
			{Key: "index", Value: int32(0)},
			{Key: "_id", Value: upsertedID},
		}}},
	}
}
