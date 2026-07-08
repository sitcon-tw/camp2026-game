package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"

	"github.com/sitcon-tw/camp2026-game/internal/gifthistory"
	"github.com/sitcon-tw/camp2026-game/internal/http/handler/admin"
)

const (
	testGiftHistoryAdminPassword = "secret"
	testGiftHistoryStaffID       = "staff-a"
	testGiftHistoryStaffToken    = "staff_token_gift_history"
	testGiftHistoryPlayerAID     = "player-a"
	testGiftHistoryPlayerAToken  = "auth_token_player_a"
	testGiftHistoryPlayerBID     = "player-b"
	testGiftHistoryPlayerBToken  = "auth_token_player_b"
)

func TestPlayerGiftHistoryAPIEmptyState(t *testing.T) {
	db := startGiftHistoryMockDatabase(t,
		createCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: testGiftHistoryPlayerAID},
			{Key: "auth_token", Value: testGiftHistoryPlayerAToken},
			{Key: "nickname", Value: "Alice"},
		}),
		createCursorResponse("camp2026_game_test.staff_rewards"),
	)

	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})

	res := performRequestWithCookie(router, http.MethodGet, "/api/me/gift-history", nil, testGiftHistoryPlayerAToken)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	body := decodeGiftHistoryResponse(t, res)
	if len(body.Entries) != 0 {
		t.Fatalf("expected empty gift history, got %#v", body.Entries)
	}
}

func TestPlayerGiftHistoryAPIReturnsRecipientScopedSingleRecord(t *testing.T) {
	recordTime := time.Date(2026, 7, 7, 8, 30, 0, 0, time.UTC)
	db := startGiftHistoryMockDatabase(t,
		createCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: testGiftHistoryPlayerAID},
			{Key: "auth_token", Value: testGiftHistoryPlayerAToken},
			{Key: "nickname", Value: "Alice"},
		}),
		createCursorResponse("camp2026_game_test.staff_rewards", bson.D{
			{Key: "_id", Value: "reward-player-a"},
			{Key: "staff_player_id", Value: testGiftHistoryStaffID},
			{Key: "recipient_player_id", Value: testGiftHistoryPlayerAID},
			{Key: "kind", Value: "item"},
			{Key: "ref_id", Value: "item_adventure_backpack"},
			{Key: "quantity", Value: 2},
			{Key: "created_at", Value: recordTime},
		}),
		createCursorResponse("camp2026_game_test.players",
			bson.D{
				{Key: "_id", Value: testGiftHistoryStaffID},
				{Key: "nickname", Value: "Staff One"},
			},
			bson.D{
				{Key: "_id", Value: testGiftHistoryPlayerAID},
				{Key: "nickname", Value: "Alice"},
			},
		),
	)

	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})

	res := performRequestWithCookie(router, http.MethodGet, "/api/me/gift-history", nil, testGiftHistoryPlayerAToken)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	body := decodeGiftHistoryResponse(t, res)
	if len(body.Entries) != 1 {
		t.Fatalf("expected one gift history entry, got %#v", body.Entries)
	}

	entry := body.Entries[0]
	if entry.RewardID != "reward-player-a" {
		t.Fatalf("expected reward id %q, got %#v", "reward-player-a", entry)
	}
	if entry.StaffPlayerID != testGiftHistoryStaffID || entry.StaffNickname != "Staff One" {
		t.Fatalf("expected sender fields to be populated, got %#v", entry)
	}
	if entry.RecipientPlayerID != testGiftHistoryPlayerAID || entry.RecipientNickname != "Alice" {
		t.Fatalf("expected recipient fields to be populated, got %#v", entry)
	}
	if entry.Name != "冒險背包" || entry.Quantity != 2 {
		t.Fatalf("expected item name and quantity, got %#v", entry)
	}
	if entry.IconPath != "/game-icons/items/item_adventure_backpack.png" {
		t.Fatalf("expected item icon path, got %#v", entry)
	}
	if !entry.CreatedAt.Equal(recordTime) {
		t.Fatalf("expected timestamp %s, got %s", recordTime.Format(time.RFC3339), entry.CreatedAt.Format(time.RFC3339))
	}
}

func TestAdminGiftHistoryAPIRequiresAdminCookie(t *testing.T) {
	db := startGiftHistoryMockDatabase(t)

	router := NewRouter(Dependencies{
		Content:       loadTestContent(t),
		MongoDB:       db,
		AdminPassword: testGiftHistoryAdminPassword,
	})

	res := performRequest(router, http.MethodGet, "/api/admin/gift-history", nil)
	assertProblem(t, res, http.StatusUnauthorized, "")
}

func TestAdminGiftHistoryAPIEmptyState(t *testing.T) {
	db := startGiftHistoryMockDatabase(t, createCursorResponse("camp2026_game_test.staff_rewards"))

	router := NewRouter(Dependencies{
		Content:       loadTestContent(t),
		MongoDB:       db,
		AdminPassword: testGiftHistoryAdminPassword,
	})

	res := performRequestWithAdminSession(t, router, http.MethodGet, "/api/admin/gift-history")
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	body := decodeGiftHistoryResponse(t, res)
	if len(body.Entries) != 0 {
		t.Fatalf("expected empty admin gift history, got %#v", body.Entries)
	}
}

func TestAdminGiftHistoryAPIReturnsMultipleOrderedRecords(t *testing.T) {
	oldest := time.Date(2026, 7, 6, 23, 55, 0, 0, time.UTC)
	newest := time.Date(2026, 7, 7, 0, 5, 0, 0, time.UTC)
	db := startGiftHistoryMockDatabase(t,
		createCursorResponse("camp2026_game_test.staff_rewards",
			bson.D{
				{Key: "_id", Value: "reward-newer"},
				{Key: "staff_player_id", Value: testGiftHistoryPlayerBID},
				{Key: "recipient_player_id", Value: testGiftHistoryPlayerAID},
				{Key: "kind", Value: "sitone"},
				{Key: "ref_id", Value: "stone_engineering_base"},
				{Key: "quantity", Value: 3},
				{Key: "created_at", Value: newest},
			},
			bson.D{
				{Key: "_id", Value: "reward-older"},
				{Key: "staff_player_id", Value: testGiftHistoryStaffID},
				{Key: "recipient_player_id", Value: testGiftHistoryPlayerAID},
				{Key: "kind", Value: "item"},
				{Key: "ref_id", Value: "item_adventure_backpack"},
				{Key: "quantity", Value: 1},
				{Key: "created_at", Value: oldest},
			},
		),
		createCursorResponse("camp2026_game_test.players",
			bson.D{
				{Key: "_id", Value: testGiftHistoryStaffID},
				{Key: "nickname", Value: "Staff One"},
			},
			bson.D{
				{Key: "_id", Value: testGiftHistoryPlayerAID},
				{Key: "nickname", Value: "Alice"},
			},
			bson.D{
				{Key: "_id", Value: testGiftHistoryPlayerBID},
				{Key: "nickname", Value: "Bob"},
			},
		),
	)

	router := NewRouter(Dependencies{
		Content:       loadTestContent(t),
		MongoDB:       db,
		AdminPassword: testGiftHistoryAdminPassword,
	})

	res := performRequestWithAdminSession(t, router, http.MethodGet, "/api/admin/gift-history")
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	body := decodeGiftHistoryResponse(t, res)
	if len(body.Entries) != 2 {
		t.Fatalf("expected two admin gift history entries, got %#v", body.Entries)
	}

	first := body.Entries[0]
	second := body.Entries[1]
	if first.RewardID != "reward-newer" || second.RewardID != "reward-older" {
		t.Fatalf("expected newest-first ordering, got %#v", body.Entries)
	}
	if first.Name != "工程型小石" || first.Quantity != 3 {
		t.Fatalf("expected sitone record with resolved name and quantity, got %#v", first)
	}
	if first.IconPath != "/game-icons/stones/basic_blue.png" {
		t.Fatalf("expected sitone icon path, got %#v", first)
	}
	if first.StaffPlayerID != testGiftHistoryPlayerBID || first.StaffNickname != "Bob" {
		t.Fatalf("expected first sender fields to be populated, got %#v", first)
	}
	if first.RecipientPlayerID != testGiftHistoryPlayerAID || first.RecipientNickname != "Alice" {
		t.Fatalf("expected first recipient fields to be populated, got %#v", first)
	}
	if !first.CreatedAt.Equal(newest) || !second.CreatedAt.Equal(oldest) {
		t.Fatalf("expected timestamps to match seeded order, got %#v", body.Entries)
	}
	if second.Name != "冒險背包" || second.Quantity != 1 {
		t.Fatalf("expected item record with resolved name and quantity, got %#v", second)
	}
	if second.IconPath != "/game-icons/items/item_adventure_backpack.png" {
		t.Fatalf("expected item icon path, got %#v", second)
	}
	if second.StaffPlayerID != testGiftHistoryStaffID || second.StaffNickname != "Staff One" {
		t.Fatalf("expected second sender fields to be populated, got %#v", second)
	}
}

func startGiftHistoryMockDatabase(t *testing.T, responses ...bson.D) *mongo.Database {
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

func createCursorResponse(ns string, batch ...bson.D) bson.D {
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

func performRequestWithAdminSession(t *testing.T, handler http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	login := performRequest(handler, http.MethodPost, "/api/admin/login", strings.NewReader(`{"password":"`+testGiftHistoryAdminPassword+`"}`))
	if login.Code != http.StatusOK {
		t.Fatalf("admin login failed: status=%d body=%s", login.Code, login.Body.String())
	}

	var session *http.Cookie
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == admin.CookieName {
			session = cookie
			break
		}
	}
	if session == nil {
		t.Fatal("expected admin session cookie")
	}

	req := httptest.NewRequest(method, path, strings.NewReader(""))
	req.AddCookie(session)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func decodeGiftHistoryResponse(t *testing.T, res *httptest.ResponseRecorder) gifthistory.Response {
	t.Helper()

	if contentType := res.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	var body gifthistory.Response
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode gift history response: %v", err)
	}
	return body
}
