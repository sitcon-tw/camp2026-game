package me

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestEventsRequiresPlayerContext(t *testing.T) {
	handler := New(Dependencies{Broker: playerevents.NewBroker()})
	req := httptest.NewRequest(http.MethodGet, "/api/me/events", nil)
	res := httptest.NewRecorder()

	handler.Events(res, req)

	assertProblem(t, res, http.StatusUnauthorized)
}

func TestEventsRequiresBroker(t *testing.T) {
	handler := New(Dependencies{})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	res := httptest.NewRecorder()

	handler.Events(res, req)

	assertProblem(t, res, http.StatusInternalServerError)
}

func TestEventsStreamsRewardGrantedEvent(t *testing.T) {
	broker := playerevents.NewBroker()
	handler := New(Dependencies{Broker: broker})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Events(res, req)
	}()

	time.Sleep(20 * time.Millisecond)
	broker.Publish("7H9K2Q", playerevents.Event{
		Name: "reward_granted",
		Reward: &playerevents.RewardGrantedEvent{
			Kind:       "item",
			RefID:      "item_adventure_backpack",
			Name:       "冒險背包",
			Quantity:   1,
			ItemType:   "material",
			IconPath:   "/game-icons/items/item_adventure_backpack.png",
			Source:     "shop_purchase",
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
	})
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if strings.Contains(res.Body.String(), "event: reward_granted") {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	body := res.Body.String()
	if !strings.Contains(body, "event: reward_granted") {
		t.Fatalf("expected reward event, got %q", body)
	}
	if !strings.Contains(body, `"name":"冒險背包"`) {
		t.Fatalf("expected reward payload, got %q", body)
	}
}

func TestEventsStreamsPendingStaffRewardsOnConnect(t *testing.T) {
	createdAt := time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC)
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.staff_rewards", bson.D{
			{Key: "_id", Value: "reward-offline"},
			{Key: "staff_player_id", Value: "staff-a"},
			{Key: "recipient_player_id", Value: "7H9K2Q"},
			{Key: "kind", Value: "item"},
			{Key: "ref_id", Value: "item_adventure_backpack"},
			{Key: "quantity", Value: 2},
			{Key: "created_at", Value: createdAt},
			{Key: "notification_pending", Value: true},
		}),
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "staff-a"},
			{Key: "nickname", Value: "Staff One"},
		}),
		createMeCursorResponse("camp2026_game_test.open_power_transfers"),
		createMeCursorResponse("camp2026_game_test.inventory_trims"),
		updateMeResponse(1),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
		Broker:  playerevents.NewBroker(),
	})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Events(res, req)
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		body := res.Body.String()
		if strings.Contains(body, `"rewardId":"reward-offline"`) && strings.Contains(body, `"delayed":true`) {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	body := res.Body.String()
	if !strings.Contains(body, "event: reward_granted") {
		t.Fatalf("expected pending reward event, got %q", body)
	}
	if !strings.Contains(body, `"rewardId":"reward-offline"`) || !strings.Contains(body, `"delayed":true`) {
		t.Fatalf("expected delayed reward payload, got %q", body)
	}
	if !strings.Contains(body, `"staffNickname":"Staff One"`) || !strings.Contains(body, `"quantity":2`) {
		t.Fatalf("expected staff and quantity fields, got %q", body)
	}
}

func TestEventsStreamsPendingInventoryTrimsOnConnect(t *testing.T) {
	createdAt := time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC)
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.open_power_transfers"),
		createMeCursorResponse("camp2026_game_test.inventory_trims", bson.D{
			{Key: "_id", Value: "trim-offline"},
			{Key: "player_id", Value: "7H9K2Q"},
			{Key: "sitone_count", Value: 2},
			{Key: "open_power", Value: 500},
			{Key: "message", Value: "小石看著自己的 AI server ，感覺記憶體不太夠，於是帶著 500 開源力去排隊購買記憶體了...應該很快就會回來"},
			{Key: "created_at", Value: createdAt},
			{Key: "notification_pending", Value: true},
		}),
		updateMeResponse(1),
	)
	handler := New(Dependencies{
		MongoDB: db,
		Broker:  playerevents.NewBroker(),
	})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Events(res, req)
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		body := res.Body.String()
		if strings.Contains(body, `"trimId":"trim-offline"`) && strings.Contains(body, `"delayed":true`) {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	body := res.Body.String()
	if !strings.Contains(body, "event: inventory_trimmed") {
		t.Fatalf("expected pending inventory trim event, got %q", body)
	}
	if !strings.Contains(body, `"trimId":"trim-offline"`) || !strings.Contains(body, `"delayed":true`) {
		t.Fatalf("expected delayed trim payload, got %q", body)
	}
	if !strings.Contains(body, `"sitoneCount":2`) || !strings.Contains(body, `"openPower":500`) {
		t.Fatalf("expected trim quantities, got %q", body)
	}
}

func TestEventsStreamsPendingOpenPowerTransfersOnConnect(t *testing.T) {
	createdAt := time.Date(2026, 7, 11, 8, 30, 0, 0, time.UTC)
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.open_power_transfers", bson.D{
			{Key: "_id", Value: "transfer-offline"},
			{Key: "sender_player_id", Value: "player-a"},
			{Key: "recipient_player_id", Value: "7H9K2Q"},
			{Key: "team_id", Value: "team-a"},
			{Key: "amount", Value: 120},
			{Key: "created_at", Value: createdAt},
			{Key: "notification_pending", Value: true},
		}),
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "player-a"},
			{Key: "nickname", Value: "Alice"},
		}),
		createMeCursorResponse("camp2026_game_test.inventory_trims"),
		updateMeResponse(1),
	)
	handler := New(Dependencies{
		MongoDB: db,
		Broker:  playerevents.NewBroker(),
	})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Events(res, req)
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		body := res.Body.String()
		if strings.Contains(body, `"rewardId":"transfer-offline"`) && strings.Contains(body, `"senderNickname":"Alice"`) {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	body := res.Body.String()
	if !strings.Contains(body, "event: reward_granted") {
		t.Fatalf("expected transfer reward event, got %q", body)
	}
	if !strings.Contains(body, `"source":"open_power_transfer"`) || !strings.Contains(body, `"amount":120`) {
		t.Fatalf("expected transfer payload fields, got %q", body)
	}
	if !strings.Contains(body, `"senderPlayerId":"player-a"`) || !strings.Contains(body, `"senderNickname":"Alice"`) || !strings.Contains(body, `"delayed":true`) {
		t.Fatalf("expected sender and delayed fields, got %q", body)
	}
}

func TestPublishOpenPowerTransferReceivedDeliversLiveRewardEvent(t *testing.T) {
	createdAt := time.Date(2026, 7, 11, 8, 30, 0, 0, time.UTC)
	db := startMeMockDatabase(t, updateMeResponse(1))
	broker := playerevents.NewBroker()
	handler := New(Dependencies{
		MongoDB: db,
		Broker:  broker,
	})
	events, unsubscribe := broker.Subscribe("player-b")
	defer unsubscribe()

	handler.publishOpenPowerTransferReceived(
		context.Background(),
		mongomodel.Player{ID: "player-a", Nickname: "Alice"},
		"player-b",
		mongomodel.OpenPowerTransfer{ID: "transfer-live", Amount: 120, CreatedAt: createdAt},
	)

	select {
	case event := <-events:
		if event.Name != "reward_granted" || event.Reward == nil {
			t.Fatalf("expected reward_granted event, got %#v", event)
		}
		if event.Reward.Source != playerevents.OpenPowerTransferSource || event.Reward.Amount != 120 {
			t.Fatalf("expected transfer open power payload, got %#v", event.Reward)
		}
		if event.Reward.SenderPlayerID != "player-a" || event.Reward.SenderNickname != "Alice" || event.Reward.Delayed {
			t.Fatalf("expected live sender payload, got %#v", event.Reward)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected live open power transfer event")
	}
}

func TestQRCodeResponse(t *testing.T) {
	handler := New(Dependencies{})
	req := authenticatedRequest(mongomodel.Player{
		ID:          "7H9K2Q",
		AuthToken:   "auth_token_123456",
		QRCodeToken: "qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok",
	})
	res := httptest.NewRecorder()

	handler.QRCode(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["qrcodeToken"] != "qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok" {
		t.Fatalf("expected qr code identifier, got %#v", body)
	}
	if _, ok := body["authToken"]; ok {
		t.Fatalf("expected auth token to be omitted, got %#v", body)
	}
}

func TestQRCodeRequiresPlayerContext(t *testing.T) {
	handler := New(Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/api/me/qrcode", nil)
	res := httptest.NewRecorder()

	handler.QRCode(res, req)

	assertProblem(t, res, http.StatusUnauthorized)
}

func TestQRCodeRequiresToken(t *testing.T) {
	handler := New(Dependencies{})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	res := httptest.NewRecorder()

	handler.QRCode(res, req)

	assertProblem(t, res, http.StatusInternalServerError)
}

func TestStatusResponse(t *testing.T) {
	team := mongomodel.Team{
		ID:        "8M4RXP",
		Name:      "Blue Team",
		AvatarURL: "/game-icons/stones/basic_blue.png",
	}
	response := statusResponse(
		mongomodel.Player{
			ID:        "7H9K2Q",
			Nickname:  "Alice",
			TeamID:    "8M4RXP",
			AvatarURL: "https://example.test/avatar/alice.png",
		},
		&team,
		1280,
		[]mongomodel.Player{
			{
				ID:        "7H9K2Q",
				Nickname:  "Alice",
				TeamID:    "8M4RXP",
				AvatarURL: "https://example.test/avatar/alice.png",
			},
			{
				ID:        "2QK9H7",
				Nickname:  "Bob",
				TeamID:    "8M4RXP",
				AvatarURL: "https://example.test/avatar/bob.png",
			},
			{
				ID:       "staff-token-1",
				Nickname: "Staff",
				TeamID:   "8M4RXP",
				Role:     "staff",
			},
		},
	)

	if response.PlayerID != "7H9K2Q" {
		t.Fatalf("expected player id, got %q", response.PlayerID)
	}
	if response.Team == nil {
		t.Fatal("expected team")
	}
	if response.Team.TeamID != "8M4RXP" {
		t.Fatalf("expected team id, got %q", response.Team.TeamID)
	}
	if response.Team.AvatarURL != "/game-icons/stones/basic_blue.png" {
		t.Fatalf("expected team avatar url, got %q", response.Team.AvatarURL)
	}
	if response.OpenPower != 1280 {
		t.Fatalf("expected open power 1280, got %d", response.OpenPower)
	}
	if len(response.TeamMembers) != 3 {
		t.Fatalf("expected 3 team members, got %#v", response.TeamMembers)
	}
	if response.TeamMembers[0].PlayerID != "7H9K2Q" || response.TeamMembers[0].Nickname != "Alice" {
		t.Fatalf("unexpected first team member: %#v", response.TeamMembers[0])
	}
	if response.TeamMembers[1].PlayerID != "2QK9H7" || response.TeamMembers[1].Nickname != "Bob" {
		t.Fatalf("unexpected second team member: %#v", response.TeamMembers[1])
	}
	if response.TeamMembers[2].PlayerID != "staff-token-1" || response.TeamMembers[2].Role != "staff" {
		t.Fatalf("unexpected third team member: %#v", response.TeamMembers[2])
	}
	if response.AvatarURL == "" {
		t.Fatalf("expected avatar url")
	}
	if response.Role != "" {
		t.Fatalf("expected empty role, got %q", response.Role)
	}
}

func TestStaffStatusResponseKeepsStaffRoleWhenNoTeamIsLoaded(t *testing.T) {
	response := statusResponse(
		mongomodel.Player{
			ID:       "staff-token-1",
			Nickname: "Staff",
			TeamID:   "team-001",
			Role:     "staff",
		},
		nil,
		0,
		nil,
	)

	if response.Team != nil {
		t.Fatalf("expected staff team to be omitted, got %#v", response.Team)
	}
	if len(response.TeamMembers) != 0 {
		t.Fatalf("expected staff team members to be empty, got %#v", response.TeamMembers)
	}
	if response.Role != "staff" {
		t.Fatalf("expected staff role, got %q", response.Role)
	}
}

func TestCreateOpenPowerTransferTransfersToSameTeamMember(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "player-b"},
			{Key: "nickname", Value: "Bob"},
			{Key: "team_id", Value: "team-a"},
		}),
		updateMeResponse(1),
		updateMeResponse(1),
		createMeCursorResponse("camp2026_game_test.open_power_records", bson.D{{Key: "total", Value: 500}}),
		createMeCursorResponse("camp2026_game_test.open_power_records", bson.D{{Key: "total", Value: 20}}),
		insertMeResponse(2),
		insertMeResponse(1),
		deleteMeResponse(1),
		deleteMeResponse(1),
	)
	handler := New(Dependencies{MongoDB: db})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		http.MethodPost,
		"/api/me/open-power-transfers",
		`{"recipientPlayerId":" player-b ","amount":120}`,
	)
	res := httptest.NewRecorder()

	handler.CreateOpenPowerTransfer(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, res.Code, res.Body.String())
	}
	var body OpenPowerTransferResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.TransferID == "" || body.SenderPlayerID != "player-a" || body.RecipientPlayerID != "player-b" {
		t.Fatalf("unexpected transfer identifiers: %#v", body)
	}
	if body.Amount != 120 || body.SenderOpenPowerAfter != 380 || body.RecipientOpenPowerAfter != 140 {
		t.Fatalf("unexpected transfer balances: %#v", body)
	}
	if body.TeamID != "team-a" || body.SenderNickname != "Alice" || body.RecipientNickname != "Bob" {
		t.Fatalf("unexpected transfer metadata: %#v", body)
	}
}

func TestCreateOpenPowerTransferRejectsDifferentTeam(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "player-b"},
			{Key: "nickname", Value: "Bob"},
			{Key: "team_id", Value: "team-b"},
		}),
	)
	handler := New(Dependencies{MongoDB: db})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		http.MethodPost,
		"/api/me/open-power-transfers",
		`{"recipientPlayerId":"player-b","amount":120}`,
	)
	res := httptest.NewRecorder()

	handler.CreateOpenPowerTransfer(res, req)

	problem := assertProblem(t, res, http.StatusForbidden)
	if problem.Detail != "open power transfers require same team" {
		t.Fatalf("unexpected problem: %#v", problem)
	}
}

func TestCreateOpenPowerTransferRejectsInsufficientOpenPower(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "player-b"},
			{Key: "nickname", Value: "Bob"},
			{Key: "team_id", Value: "team-a"},
		}),
		updateMeResponse(1),
		updateMeResponse(1),
		createMeCursorResponse("camp2026_game_test.open_power_records", bson.D{{Key: "total", Value: 50}}),
		deleteMeResponse(1),
		deleteMeResponse(1),
	)
	handler := New(Dependencies{MongoDB: db})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		http.MethodPost,
		"/api/me/open-power-transfers",
		`{"recipientPlayerId":"player-b","amount":120}`,
	)
	res := httptest.NewRecorder()

	handler.CreateOpenPowerTransfer(res, req)

	problem := assertProblem(t, res, http.StatusConflict)
	if problem.Detail != "insufficient open power for transfer" {
		t.Fatalf("unexpected problem: %#v", problem)
	}
}

func TestUpdateNicknameSavesTrimmedNickname(t *testing.T) {
	db := startMeMockDatabase(t, updateMeResponse(1))
	handler := New(Dependencies{MongoDB: db})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q", Nickname: "Alice"},
		http.MethodPut,
		"/api/me/nickname",
		`{"nickname":"  小明  "}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateNickname(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	var body UpdateNicknameResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Nickname != "小明" {
		t.Fatalf("expected trimmed nickname, got %#v", body)
	}
}

func TestUpdateNicknameRejectsBlankNickname(t *testing.T) {
	handler := New(Dependencies{MongoDB: startMeMockDatabase(t)})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/nickname",
		`{"nickname":"   "}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateNickname(res, req)

	assertProblem(t, res, http.StatusBadRequest)
}

func TestUpdateNicknameRejectsTooLongNickname(t *testing.T) {
	handler := New(Dependencies{MongoDB: startMeMockDatabase(t)})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/nickname",
		`{"nickname":"一二三四五六七八九十一二三四五六七八九十一"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateNickname(res, req)

	assertProblem(t, res, http.StatusBadRequest)
}

func TestUpdateNicknameRequiresPlayerContext(t *testing.T) {
	handler := New(Dependencies{MongoDB: startMeMockDatabase(t)})
	req := httptest.NewRequest(http.MethodPut, "/api/me/nickname", strings.NewReader(`{"nickname":"Alice"}`))
	res := httptest.NewRecorder()

	handler.UpdateNickname(res, req)

	assertProblem(t, res, http.StatusUnauthorized)
}

func TestUpdateNicknameRequiresDatabase(t *testing.T) {
	handler := New(Dependencies{})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/nickname",
		`{"nickname":"Alice"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateNickname(res, req)

	assertProblem(t, res, http.StatusServiceUnavailable)
}

func TestUpdateAvatarSetsOwnedSitoneIcon(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.player_sitones", bson.D{
			{Key: "_id", Value: "owned-sitone-001"},
			{Key: "player_id", Value: "7H9K2Q"},
			{Key: "sitone_id", Value: "stone_engineering_base"},
			{Key: "quantity", Value: 1},
		}),
		updateMeResponse(1),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/avatar",
		`{"sitoneId":"stone_engineering_base"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateAvatar(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	var body UpdateAvatarResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AvatarURL != "/game-icons/stones/basic_blue.png" {
		t.Fatalf("expected basic blue avatar, got %#v", body)
	}
}

func TestUpdateAvatarRejectsUnownedSitone(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.player_sitones"),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/avatar",
		`{"sitoneId":"stone_engineering_base"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateAvatar(res, req)

	assertProblem(t, res, http.StatusBadRequest)
}

func TestUpdateAvatarClearsAvatar(t *testing.T) {
	db := startMeMockDatabase(t, updateMeResponse(1))
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/avatar",
		`{"sitoneId":null}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateAvatar(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	var body UpdateAvatarResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AvatarURL != "" {
		t.Fatalf("expected empty avatar url, got %#v", body)
	}
}

func TestUpdateTeamAvatarSetsCatalogSitoneIcon(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "7H9K2Q"},
			{Key: "team_id", Value: "8M4RXP"},
		}, bson.D{
			{Key: "_id", Value: "2QK9H7"},
			{Key: "team_id", Value: "8M4RXP"},
		}),
		createMeCursorResponse("camp2026_game_test.player_sitones", bson.D{
			{Key: "_id", Value: "owned-sitone-001"},
			{Key: "player_id", Value: "2QK9H7"},
			{Key: "sitone_id", Value: "stone_engineering_base"},
			{Key: "quantity", Value: 1},
		}),
		findAndModifyMeResponse(bson.D{
			{Key: "_id", Value: "8M4RXP"},
			{Key: "name", Value: "Blue Team"},
			{Key: "avatar_url", Value: "/game-icons/stones/basic_blue.png"},
		}),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q", TeamID: "8M4RXP"},
		http.MethodPut,
		"/api/me/team/avatar",
		`{"sitoneId":"stone_engineering_base"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateTeamAvatar(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	var body UpdateTeamAvatarResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Team.TeamID != "8M4RXP" || body.Team.AvatarURL != "/game-icons/stones/basic_blue.png" {
		t.Fatalf("unexpected team avatar response: %#v", body)
	}
}

func TestUpdateTeamAvatarRejectsSitoneUnownedByTeam(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "7H9K2Q"},
			{Key: "team_id", Value: "8M4RXP"},
		}),
		createMeCursorResponse("camp2026_game_test.player_sitones"),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q", TeamID: "8M4RXP"},
		http.MethodPut,
		"/api/me/team/avatar",
		`{"sitoneId":"stone_engineering_base"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateTeamAvatar(res, req)

	assertProblem(t, res, http.StatusBadRequest)
}

func TestUpdateTeamAvatarClearsAvatar(t *testing.T) {
	db := startMeMockDatabase(t,
		findAndModifyMeResponse(bson.D{
			{Key: "_id", Value: "8M4RXP"},
			{Key: "name", Value: "Blue Team"},
		}),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q", TeamID: "8M4RXP"},
		http.MethodPut,
		"/api/me/team/avatar",
		`{"sitoneId":null}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateTeamAvatar(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	var body UpdateTeamAvatarResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Team.AvatarURL != "" {
		t.Fatalf("expected empty team avatar url, got %#v", body)
	}
}

func TestUpdateTeamAvatarRejectsPlayerWithoutTeam(t *testing.T) {
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: startMeMockDatabase(t),
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q"},
		http.MethodPut,
		"/api/me/team/avatar",
		`{"sitoneId":"stone_engineering_base"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateTeamAvatar(res, req)

	assertProblem(t, res, http.StatusBadRequest)
}

func TestUpdateTeamAvatarRejectsUnknownSitone(t *testing.T) {
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: startMeMockDatabase(t),
	})
	req := authenticatedJSONRequest(
		mongomodel.Player{ID: "7H9K2Q", TeamID: "8M4RXP"},
		http.MethodPut,
		"/api/me/team/avatar",
		`{"sitoneId":"missing"}`,
	)
	res := httptest.NewRecorder()

	handler.UpdateTeamAvatar(res, req)

	assertProblem(t, res, http.StatusBadRequest)
}

func TestListTeamSitonesReturnsTeamOwnedUnion(t *testing.T) {
	db := startMeMockDatabase(t,
		createMeCursorResponse("camp2026_game_test.players", bson.D{
			{Key: "_id", Value: "7H9K2Q"},
			{Key: "team_id", Value: "8M4RXP"},
		}, bson.D{
			{Key: "_id", Value: "2QK9H7"},
			{Key: "team_id", Value: "8M4RXP"},
		}),
		createMeCursorResponse("camp2026_game_test.player_sitones", bson.D{
			{Key: "_id", Value: "owned-sitone-001"},
			{Key: "player_id", Value: "7H9K2Q"},
			{Key: "sitone_id", Value: "stone_engineering_base"},
			{Key: "quantity", Value: 1},
		}, bson.D{
			{Key: "_id", Value: "owned-sitone-002"},
			{Key: "player_id", Value: "2QK9H7"},
			{Key: "sitone_id", Value: "stone_engineering_base"},
			{Key: "quantity", Value: 2},
		}, bson.D{
			{Key: "_id", Value: "owned-sitone-003"},
			{Key: "player_id", Value: "2QK9H7"},
			{Key: "sitone_id", Value: "stone_explorer_base"},
			{Key: "quantity", Value: 1},
		}),
	)
	handler := New(Dependencies{
		Content: loadTestContent(t),
		MongoDB: db,
	})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q", TeamID: "8M4RXP"})
	res := httptest.NewRecorder()

	handler.ListTeamSitones(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	var body SitoneListResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Sitones) != 2 {
		t.Fatalf("expected 2 team sitones, got %#v", body.Sitones)
	}
	if body.Sitones[0].SitoneID != "stone_engineering_base" || body.Sitones[0].Quantity != 3 {
		t.Fatalf("expected summed engineering sitone, got %#v", body.Sitones[0])
	}
	if body.Sitones[1].SitoneID != "stone_explorer_base" || body.Sitones[1].Quantity != 1 {
		t.Fatalf("expected explorer sitone, got %#v", body.Sitones[1])
	}
}

func TestTeamMemberResponsesSkipsInvalidPlayers(t *testing.T) {
	members := teamMemberResponses([]mongomodel.Player{
		{ID: "7H9K2Q", Nickname: "Alice", AuthToken: "secret", QRCodeToken: "qr-secret"},
		{ID: "", Nickname: "Missing ID"},
		{ID: "2QK9H7"},
		{ID: "staff-token-1", Nickname: "Staff", Role: "staff"},
	})

	if len(members) != 2 {
		t.Fatalf("expected 2 team members, got %#v", members)
	}
	if members[0].PlayerID != "7H9K2Q" || members[0].Nickname != "Alice" {
		t.Fatalf("unexpected team member: %#v", members[0])
	}
	if members[1].PlayerID != "staff-token-1" || members[1].Role != "staff" {
		t.Fatalf("unexpected staff team member: %#v", members[1])
	}
}

func TestHomeActions(t *testing.T) {
	actions := homeActions(gamecontrol.DefaultSettings())
	if len(actions) != 9 {
		t.Fatalf("expected 9 home actions, got %#v", actions)
	}
	for _, action := range actions {
		if action.ID == "" || action.Label == "" || !action.Enabled {
			t.Fatalf("expected enabled action with id and label, got %#v", action)
		}
	}
}

func TestHomeActionsDisableBattleWhenOpeningLocked(t *testing.T) {
	settings := gamecontrol.DefaultSettings()
	settings.BattleOpeningOverride = gamecontrol.BattleOpeningOverrideForceClosed

	actions := homeActions(settings)
	for _, action := range actions {
		if action.ID == "battle" && action.Enabled {
			t.Fatalf("expected battle action to be disabled, got %#v", action)
		}
		if action.ID != "battle" && !action.Enabled {
			t.Fatalf("expected non-battle action to stay enabled, got %#v", action)
		}
	}
}

func TestOpenPowerTotalPipeline(t *testing.T) {
	got := openPowerTotalPipeline("7H9K2Q")
	want := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: "7H9K2Q"}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected pipeline: %#v", got)
	}
}

func TestPlayerSitoneCountsPipeline(t *testing.T) {
	pipeline := playerSitoneCountsPipeline()
	if len(pipeline) != 2 {
		t.Fatalf("expected 2 pipeline stages, got %#v", pipeline)
	}
}

func TestOpenPowerScoresByPlayerPipeline(t *testing.T) {
	pipeline := openPowerScoresByPlayerPipeline()
	if len(pipeline) != 1 {
		t.Fatalf("expected 1 pipeline stage, got %#v", pipeline)
	}
}

func TestTeamRankEntriesRankByAverageSitonesThenAverageOpenPower(t *testing.T) {
	teams := []mongomodel.Team{
		{ID: "team-a", Name: "Alpha"},
		{ID: "team-b", Name: "Beta"},
		{ID: "team-c", Name: "Gamma"},
	}
	players := []mongomodel.Player{
		{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		{ID: "player-b", Nickname: "Bob", TeamID: "team-b"},
		{ID: "player-c", Nickname: "Cody", TeamID: "team-c"},
		{ID: "player-d", Nickname: "Dana", TeamID: "team-c"},
		{ID: "staff-a", Nickname: "Staff", TeamID: "team-a", Role: authctx.PlayerRoleStaff},
	}
	stats := map[string]teamRankStats{
		"player-a": {SitoneCount: 2, OpenPower: 100},
		"player-b": {SitoneCount: 4, OpenPower: 20},
		"player-c": {SitoneCount: 4, OpenPower: 50},
		"player-d": {SitoneCount: 4, OpenPower: 50},
		"staff-a":  {SitoneCount: 2, OpenPower: 100},
	}

	rows := teamRankEntries(teams, players, stats)
	current := currentTeamRank(rows, "team-a")

	wantIDs := []string{"team-c", "team-b", "team-a"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("expected %d rows, got %#v", len(wantIDs), rows)
	}
	for index, wantID := range wantIDs {
		if rows[index].TeamID != wantID || rows[index].Rank != index+1 {
			t.Fatalf("unexpected row at %d: got %#v want id %q rank %d", index, rows[index], wantID, index+1)
		}
	}
	if current == nil || current.TeamID != "team-a" {
		t.Fatalf("expected current team-a rank, got %#v", current)
	}
	if current.SitoneCount != 4 || current.OpenPower != 200 || current.PlayerCount != 2 {
		t.Fatalf("expected staff stats to be included, got %#v", current)
	}
	if current.AverageSitones != 2 || current.AverageOpenPower != 100 {
		t.Fatalf("expected balanced averages, got %#v", current)
	}
	if current.GapToPrevious != 2 {
		t.Fatalf("expected average sitone gap 2, got %v", current.GapToPrevious)
	}
}

func TestOpenPowerTotalFromCursor(t *testing.T) {
	cursor, err := mongo.NewCursorFromDocuments([]any{
		bson.D{{Key: "total", Value: 1280}},
	}, nil, nil)
	if err != nil {
		t.Fatalf("new cursor: %v", err)
	}

	total, err := openPowerTotalFromCursor(context.Background(), cursor)
	if err != nil {
		t.Fatalf("open power total: %v", err)
	}
	if total != 1280 {
		t.Fatalf("expected total 1280, got %d", total)
	}
}

func TestMapPlayerSitones(t *testing.T) {
	sitones, err := mapPlayerSitones(loadTestContent(t), []mongomodel.PlayerSitone{
		{
			ID:       "owned-sitone-001",
			PlayerID: "7H9K2Q",
			SitoneID: "stone_engineering_base",
			Quantity: 1,
		},
	})
	if err != nil {
		t.Fatalf("map sitones: %v", err)
	}
	if len(sitones) != 1 {
		t.Fatalf("expected 1 sitone, got %d", len(sitones))
	}
	if sitones[0].Sitone.Name != "工程型小石" {
		t.Fatalf("expected catalog sitone name, got %#v", sitones[0])
	}
}

func TestMapPlayerSitonesSkipsMissingCatalogDefinition(t *testing.T) {
	sitones, err := mapPlayerSitones(loadTestContent(t), []mongomodel.PlayerSitone{
		{ID: "owned-sitone-001", SitoneID: "sitone-missing", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("map sitones: %v", err)
	}
	if len(sitones) != 0 {
		t.Fatalf("expected missing catalog sitone to be skipped, got %#v", sitones)
	}
}

func TestNormalizeSitoneLoadoutAllowsDuplicateSlots(t *testing.T) {
	got, err := normalizeSitoneLoadout([]string{
		" stone_engineering_base ",
		"stone_engineering_base",
		"",
	})
	if err != nil {
		t.Fatalf("normalize sitone loadout: %v", err)
	}

	want := []string{"stone_engineering_base", "stone_engineering_base"}
	if len(got) != len(want) {
		t.Fatalf("expected %d sitones, got %#v", len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected sitone at index %d: got %q want %q", index, got[index], want[index])
		}
	}
}

func TestMapPlayerItems(t *testing.T) {
	items, err := mapPlayerItems(loadTestContent(t), []mongomodel.PlayerItem{
		{
			ID:       "owned-item-001",
			PlayerID: "7H9K2Q",
			ItemID:   "item_adventure_backpack",
			Quantity: 3,
		},
	})
	if err != nil {
		t.Fatalf("map items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Item.Name != "冒險背包" {
		t.Fatalf("expected catalog item name, got %#v", items[0])
	}
}

func TestMapPlayerItemsSkipsMissingCatalogDefinition(t *testing.T) {
	items, err := mapPlayerItems(loadTestContent(t), []mongomodel.PlayerItem{
		{ID: "owned-item-001", ItemID: "item-missing", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("map items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected missing catalog item to be skipped, got %#v", items)
	}
}

func TestMapPlayerItemsReturnsEmptySlice(t *testing.T) {
	items, err := mapPlayerItems(loadTestContent(t), nil)
	if err != nil {
		t.Fatalf("map items: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestCompletedMatchesFilterOnlyReturnsCurrentPlayerCompletedMatches(t *testing.T) {
	got := completedMatchesFilter("7H9K2Q")
	want := bson.D{
		{Key: "status", Value: mongomodel.MatchStatusCompleted},
		{Key: "players.player_id", Value: "7H9K2Q"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected completed matches filter: %#v", got)
	}
}

func TestMapCompletedMatches(t *testing.T) {
	completedAt := testTime(t, "2026-06-12T06:30:00Z")
	records := []mongomodel.Match{
		{
			ID:           "match_123",
			Status:       mongomodel.MatchStatusCompleted,
			HostPlayerID: "P1",
			Players: []mongomodel.MatchPlayer{
				{
					PlayerID:  "P1",
					Nickname:  "Alice",
					Score:     850,
					SitoneIDs: []string{"stone_engineering_base"},
				},
				{
					PlayerID: "P2",
					Nickname: "Bob",
					Score:    700,
				},
			},
			QuestionIDs: []string{"quiz-001", "quiz-002"},
			CompletedAt: completedAt,
		},
	}

	matches := mapCompletedMatches(records, map[string]string{"P1": "/avatar/alice.png"})
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %#v", matches)
	}
	if matches[0].MatchID != "match_123" ||
		matches[0].Status != mongomodel.MatchStatusCompleted ||
		matches[0].QuestionCount != 2 ||
		matches[0].CompletedAt == nil {
		t.Fatalf("unexpected completed match response: %#v", matches[0])
	}
	if len(matches[0].Players) != 2 || matches[0].Players[0].Score != 850 {
		t.Fatalf("unexpected completed match players: %#v", matches[0].Players)
	}
	if matches[0].Players[0].AvatarURL != "/avatar/alice.png" {
		t.Fatalf("expected completed match player avatar, got %#v", matches[0].Players[0])
	}
}

func startMeMockDatabase(t *testing.T, responses ...bson.D) *mongo.Database {
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

func createMeCursorResponse(ns string, batch ...bson.D) bson.D {
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

func updateMeResponse(modified int32) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: modified},
		{Key: "nModified", Value: modified},
	}
}

func insertMeResponse(inserted int32) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: inserted},
	}
}

func deleteMeResponse(deleted int32) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: deleted},
	}
}

func findAndModifyMeResponse(value bson.D) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "lastErrorObject", Value: bson.D{
			{Key: "n", Value: 1},
			{Key: "updatedExisting", Value: true},
		}},
		{Key: "value", Value: value},
	}
}

func authenticatedRequest(player mongomodel.Player) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/me/qrcode", strings.NewReader(""))
	return req.WithContext(authctx.WithPlayer(req.Context(), player))
}

func authenticatedJSONRequest(player mongomodel.Player, method string, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req.WithContext(authctx.WithPlayer(req.Context(), player))
}

func assertProblem(t *testing.T, res *httptest.ResponseRecorder, status int) httpx.ProblemDetails {
	t.Helper()

	if res.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, res.Code, res.Body.String())
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("expected problem content type, got %q", contentType)
	}

	var problem httpx.ProblemDetails
	if err := json.NewDecoder(res.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Status != status {
		t.Fatalf("expected problem status %d, got %d", status, problem.Status)
	}
	return problem
}

func loadTestContent(t *testing.T) *content.Store {
	t.Helper()

	return testcontent.Load(t)
}

func testTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse test time: %v", err)
	}
	return parsed
}
