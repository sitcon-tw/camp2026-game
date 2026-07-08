package bot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sitcon-tw/camp2026-game/tgbot/internal/domain"
	"github.com/sitcon-tw/camp2026-game/tgbot/internal/telegram"
)

func TestHandleGroupStartCreatesPrivateDeepLinkRequest(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 42, FirstName: "Alice", Username: "alice"},
			Chat:      telegram.Chat{ID: -1001, Type: "supergroup"},
			Text:      "vip-666",
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}

	request, ok := store.requests["nonce-1"]
	if !ok {
		t.Fatalf("expected login request to be stored")
	}
	if request.TelegramUserID != 42 || request.TelegramChatID != -1001 || request.TeamID != "team-001" {
		t.Fatalf("unexpected login request: %#v", request)
	}
	if !request.ExpiresAt.Equal(now.Add(10 * time.Minute)) {
		t.Fatalf("unexpected expiration: %s", request.ExpiresAt)
	}

	if len(messenger.sent) != 1 {
		t.Fatalf("expected one outbound message, got %d", len(messenger.sent))
	}
	if len(messenger.deleted) != 1 {
		t.Fatalf("expected original command to be deleted, got %#v", messenger.deleted)
	}
	if messenger.deleted[0].ChatID != -1001 || messenger.deleted[0].MessageID != 12 {
		t.Fatalf("unexpected deleted command: %#v", messenger.deleted[0])
	}
	msg := messenger.sent[0]
	if msg.ChatID != -1001 || msg.ReplyToMessageID != 0 || msg.ParseMode != "HTML" {
		t.Fatalf("unexpected group message target: %#v", msg)
	}
	if !strings.Contains(msg.Text, `<a href="tg://user?id=42">@alice</a>`) || !strings.Contains(msg.Text, "VIP 傳送門") {
		t.Fatalf("expected generated message to tag the player, got %q", msg.Text)
	}
	if msg.ReplyMarkup == nil || len(msg.ReplyMarkup.InlineKeyboard) != 1 || len(msg.ReplyMarkup.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected one inline deep-link button: %#v", msg.ReplyMarkup)
	}
	button := msg.ReplyMarkup.InlineKeyboard[0][0]
	if button.URL != "" {
		t.Fatalf("group button must use callback data, got URL %q", button.URL)
	}
	if button.CallbackData != "login:42:nonce-1" {
		t.Fatalf("unexpected button callback data: %q", button.CallbackData)
	}
	if strings.Contains(button.CallbackData, "tg_token") {
		t.Fatalf("group callback data must not contain auth token: %q", button.CallbackData)
	}
}

func TestHandleGroupStartIgnoresStartCommand(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 12,
			From:      &telegram.User{ID: 42, FirstName: "Alice"},
			Chat:      telegram.Chat{ID: -1001, Type: "supergroup"},
			Text:      "/start",
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}
	if len(store.requests) != 0 {
		t.Fatalf("expected /start to be ignored in groups, got %#v", store.requests)
	}
	if len(messenger.sent) != 0 {
		t.Fatalf("expected no outbound messages, got %#v", messenger.sent)
	}
}

func TestHandleLoginCallbackAnswersWithDeepLinkAndDeletesMessages(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "callback-1",
			From: telegram.User{ID: 42, FirstName: "Alice"},
			Data: loginCallbackData(42, "nonce-1"),
			Message: &telegram.Message{
				MessageID: 99,
				Chat:      telegram.Chat{ID: -1001, Type: "supergroup"},
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}

	if len(messenger.answered) != 1 {
		t.Fatalf("expected one callback answer, got %d", len(messenger.answered))
	}
	answer := messenger.answered[0]
	if answer.CallbackQueryID != "callback-1" || answer.URL != "https://t.me/CampGameBot?start=nonce-1" {
		t.Fatalf("unexpected callback answer: %#v", answer)
	}
	if len(messenger.deleted) != 1 {
		t.Fatalf("expected generated message to be deleted, got %#v", messenger.deleted)
	}
	if messenger.deleted[0].ChatID != -1001 || messenger.deleted[0].MessageID != 99 {
		t.Fatalf("expected generated message to be deleted, got %#v", messenger.deleted[0])
	}
}

func TestHandleLoginCallbackRejectsOtherTelegramUser(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "callback-1",
			From: telegram.User{ID: 77, FirstName: "Mallory"},
			Data: loginCallbackData(42, "nonce-1"),
			Message: &telegram.Message{
				MessageID: 99,
				Chat:      telegram.Chat{ID: -1001, Type: "supergroup"},
				ReplyToMessage: &telegram.Message{
					MessageID: 12,
					Chat:      telegram.Chat{ID: -1001, Type: "supergroup"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}
	if len(messenger.answered) != 1 || !messenger.answered[0].ShowAlert || messenger.answered[0].URL != "" {
		t.Fatalf("expected alert callback answer without URL, got %#v", messenger.answered)
	}
	if len(messenger.deleted) != 0 {
		t.Fatalf("expected no deleted messages, got %#v", messenger.deleted)
	}
}

func TestHandlePrivateStartCreatesPlayerAndSendsLoginURL(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	store.requests["nonce-1"] = domain.LoginRequest{
		ID:             "nonce-1",
		TelegramUserID: 42,
		TelegramChatID: -1001,
		TeamID:         "team-001",
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Minute),
	}
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		Message: &telegram.Message{
			MessageID: 1,
			From:      &telegram.User{ID: 42, FirstName: "Alice", LastName: "Wang", Username: "alice"},
			Chat:      telegram.Chat{ID: 42, Type: "private"},
			Text:      "/start nonce-1",
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}

	player, ok := store.players[42]
	if !ok {
		t.Fatalf("expected player to be created")
	}
	if player.ID != "tg_42" || player.AuthToken != "tg_token" || player.QRCodeToken != "qr_token" || player.Nickname != "Alice Wang" || player.TeamID != "team-001" {
		t.Fatalf("unexpected player: %#v", player)
	}
	wantSitones := []string{
		"stone_explorer_base",
		"stone_inspiration_base",
		"stone_resonance_base",
		"stone_engineering_base",
		"stone_entertainment_base",
	}
	if strings.Join(player.DefaultSitoneIDs, ",") != strings.Join(wantSitones, ",") {
		t.Fatalf("unexpected default sitones: %#v", player.DefaultSitoneIDs)
	}
	if strings.Join(store.playerSitones["tg_42"], ",") != strings.Join(wantSitones, ",") {
		t.Fatalf("expected initial sitones to be granted, got %#v", store.playerSitones["tg_42"])
	}
	if _, ok := store.requests["nonce-1"]; ok {
		t.Fatalf("expected login request to be consumed")
	}
	if len(messenger.sent) != 1 {
		t.Fatalf("expected one outbound message, got %d", len(messenger.sent))
	}
	msg := messenger.sent[0]
	loginURL := "https://game.example.test/login?token=tg_token"
	if msg.ChatID != 42 || !msg.DisableWebPagePreview || msg.ParseMode != "HTML" {
		t.Fatalf("unexpected private message: %#v", msg)
	}
	if !strings.Contains(msg.Text, "VIP-666") || !strings.Contains(msg.Text, "請不要轉傳給其他人") {
		t.Fatalf("expected VIP copy and safety warning, got %q", msg.Text)
	}
	if !strings.Contains(msg.Text, `<a href="`+loginURL+`">點我登入遊戲</a>`) {
		t.Fatalf("expected login URL as a text hyperlink, got %q", msg.Text)
	}
	if !strings.Contains(msg.Text, "Token 登入：`tg_token`") {
		t.Fatalf("expected backtick-wrapped fallback token, got %q", msg.Text)
	}
	if strings.Contains(msg.Text, "\n"+loginURL+"\n") {
		t.Fatalf("expected message body not to show a bare URL line, got %q", msg.Text)
	}
	if msg.ReplyMarkup == nil || len(msg.ReplyMarkup.InlineKeyboard) != 1 || len(msg.ReplyMarkup.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected one login URL button, got %#v", msg.ReplyMarkup)
	}
	button := msg.ReplyMarkup.InlineKeyboard[0][0]
	if button.Text != "🚪 打開遊戲入口" || button.URL != loginURL || button.CallbackData != "" {
		t.Fatalf("unexpected login button: %#v", button)
	}
}

func TestHandlePrivateStartRejectsWrongTelegramUser(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	store.requests["nonce-1"] = domain.LoginRequest{
		ID:             "nonce-1",
		TelegramUserID: 42,
		TelegramChatID: -1001,
		TeamID:         "team-001",
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Minute),
	}
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		Message: &telegram.Message{
			From: &telegram.User{ID: 77, FirstName: "Mallory"},
			Chat: telegram.Chat{ID: 77, Type: "private"},
			Text: "/start nonce-1",
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}
	if len(store.players) != 0 {
		t.Fatalf("expected no player creation, got %#v", store.players)
	}
	if len(messenger.sent) != 1 || !strings.Contains(messenger.sent[0].Text, "已失效") {
		t.Fatalf("expected invalid request message, got %#v", messenger.sent)
	}
}

func TestHandlePrivateStartKeepsExistingPlayerTeam(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	store.requests["nonce-2"] = domain.LoginRequest{
		ID:             "nonce-2",
		TelegramUserID: 42,
		TelegramChatID: -1002,
		TeamID:         "team-002",
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Minute),
	}
	store.players[42] = domain.Player{
		ID:             "tg_42",
		AuthToken:      "old-token",
		Nickname:       "Alice",
		TeamID:         "team-001",
		TelegramUserID: 42,
	}
	messenger := &fakeMessenger{}
	service := newTestService(t, store, messenger, now)

	err := service.HandleUpdate(context.Background(), telegram.Update{
		Message: &telegram.Message{
			From: &telegram.User{ID: 42, FirstName: "Alice"},
			Chat: telegram.Chat{ID: 42, Type: "private"},
			Text: "/start nonce-2",
		},
	})
	if err != nil {
		t.Fatalf("HandleUpdate returned error: %v", err)
	}

	player := store.players[42]
	if player.TeamID != "team-001" || player.AuthToken != "old-token" {
		t.Fatalf("expected existing team/token to be preserved, got %#v", player)
	}
	if len(messenger.sent) != 1 {
		t.Fatalf("expected one message, got %d", len(messenger.sent))
	}
	if messenger.sent[0].ParseMode != "HTML" || !strings.Contains(messenger.sent[0].Text, "team-001") || !strings.Contains(messenger.sent[0].Text, "old-token") {
		t.Fatalf("expected existing team and old token in message, got %q", messenger.sent[0].Text)
	}
}

func TestParseStartCommand(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantPayload string
		wantOK      bool
	}{
		{name: "plain", text: "/start", wantOK: true},
		{name: "payload", text: "/start abc", wantPayload: "abc", wantOK: true},
		{name: "mention", text: "/start@CampGameBot abc", wantPayload: "abc", wantOK: true},
		{name: "other bot mention", text: "/start@OtherBot abc", wantOK: false},
		{name: "other command", text: "/help", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, ok := parseStartCommand(tt.text, "CampGameBot")
			if payload != tt.wantPayload || ok != tt.wantOK {
				t.Fatalf("parseStartCommand(%q) = %q, %v; want %q, %v", tt.text, payload, ok, tt.wantPayload, tt.wantOK)
			}
		})
	}
}

func TestParseGroupLoginCommand(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantPayload string
		wantOK      bool
	}{
		{name: "plain", text: "vip-666", wantOK: true},
		{name: "payload", text: "vip-666 abc", wantPayload: "abc", wantOK: true},
		{name: "uppercase", text: "VIP-666", wantOK: true},
		{name: "old slash command", text: "/vip-666", wantOK: false},
		{name: "mention", text: "vip-666@CampGameBot abc", wantOK: false},
		{name: "old start command", text: "/start", wantOK: false},
		{name: "other command", text: "/help", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, ok := parseGroupLoginCommand(tt.text, "CampGameBot")
			if payload != tt.wantPayload || ok != tt.wantOK {
				t.Fatalf("parseGroupLoginCommand(%q) = %q, %v; want %q, %v", tt.text, payload, ok, tt.wantPayload, tt.wantOK)
			}
		})
	}
}

func newTestService(t *testing.T, store *fakeStore, messenger *fakeMessenger, now time.Time) *Service {
	t.Helper()

	service, err := NewService(Dependencies{
		Store:        store,
		Messenger:    messenger,
		BotUsername:  "CampGameBot",
		LoginBaseURL: "https://game.example.test/login",
		GroupTeamMap: map[int64]string{
			-1001: "team-001",
			-1002: "team-002",
		},
		InitialSitoneIDs: []string{
			"stone_explorer_base",
			"stone_inspiration_base",
			"stone_resonance_base",
			"stone_engineering_base",
			"stone_entertainment_base",
		},
		RequestTTL: 10 * time.Minute,
		Now: func() time.Time {
			return now
		},
		NewNonce: func() (string, error) {
			return "nonce-1", nil
		},
		NewAuthToken: func() (string, error) {
			return "tg_token", nil
		},
		NewQRCodeToken: func() (string, error) {
			return "qr_token", nil
		},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	return service
}

type fakeStore struct {
	requests      map[string]domain.LoginRequest
	players       map[int64]domain.Player
	playerSitones map[string][]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		requests:      make(map[string]domain.LoginRequest),
		players:       make(map[int64]domain.Player),
		playerSitones: make(map[string][]string),
	}
}

func (s *fakeStore) CreateLoginRequest(_ context.Context, request domain.LoginRequest) error {
	s.requests[request.ID] = request
	return nil
}

func (s *fakeStore) RedeemLoginRequest(_ context.Context, nonce string, telegramUserID int64, now time.Time) (domain.LoginRequest, error) {
	request, ok := s.requests[nonce]
	if !ok || request.TelegramUserID != telegramUserID || !request.ExpiresAt.After(now) {
		return domain.LoginRequest{}, domain.ErrLoginRequestNotFound
	}
	delete(s.requests, nonce)
	return request, nil
}

func (s *fakeStore) GetOrCreatePlayer(_ context.Context, input domain.CreatePlayerInput) (domain.Player, bool, error) {
	if input.TelegramUserID == 0 {
		return domain.Player{}, false, errors.New("telegram user id is required")
	}
	if player, ok := s.players[input.TelegramUserID]; ok {
		player.TelegramUsername = input.TelegramUsername
		player.UpdatedAt = input.Now
		s.players[input.TelegramUserID] = player
		return player, false, nil
	}
	player := domain.Player{
		ID:               input.PlayerID,
		AuthToken:        input.AuthToken,
		QRCodeToken:      input.QRCodeToken,
		Nickname:         input.Nickname,
		TeamID:           input.TeamID,
		DefaultSitoneIDs: append([]string(nil), input.InitialSitoneIDs...),
		TelegramUserID:   input.TelegramUserID,
		TelegramUsername: input.TelegramUsername,
		TelegramChatID:   input.TelegramChatID,
		CreatedAt:        input.Now,
		UpdatedAt:        input.Now,
	}
	s.players[input.TelegramUserID] = player
	s.playerSitones[player.ID] = append([]string(nil), input.InitialSitoneIDs...)
	return player, true, nil
}

type fakeMessenger struct {
	sent     []telegram.SendMessageRequest
	deleted  []telegram.DeleteMessageRequest
	answered []telegram.AnswerCallbackQueryRequest
}

func (m *fakeMessenger) SendMessage(_ context.Context, req telegram.SendMessageRequest) error {
	m.sent = append(m.sent, req)
	return nil
}

func (m *fakeMessenger) DeleteMessage(_ context.Context, req telegram.DeleteMessageRequest) error {
	m.deleted = append(m.deleted, req)
	return nil
}

func (m *fakeMessenger) AnswerCallbackQuery(_ context.Context, req telegram.AnswerCallbackQueryRequest) error {
	m.answered = append(m.answered, req)
	return nil
}
