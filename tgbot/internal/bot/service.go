package bot

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sitcon-tw/camp2026-game/tgbot/internal/domain"
	"github.com/sitcon-tw/camp2026-game/tgbot/internal/telegram"
)

const (
	defaultRequestTTL   = 10 * time.Minute
	nonceBytes          = 24
	authTokenBytes      = 32
	qrCodeTokenBytes    = 32
	groupLoginCommand   = "vip-666"
	loginCallbackPrefix = "login:"
	parseModeHTML       = "HTML"
)

type Store interface {
	CreateLoginRequest(ctx context.Context, request domain.LoginRequest) error
	RedeemLoginRequest(ctx context.Context, nonce string, telegramUserID int64, now time.Time) (domain.LoginRequest, error)
	GetOrCreatePlayer(ctx context.Context, input domain.CreatePlayerInput) (domain.Player, bool, error)
}

type Messenger interface {
	SendMessage(ctx context.Context, req telegram.SendMessageRequest) error
	DeleteMessage(ctx context.Context, req telegram.DeleteMessageRequest) error
	AnswerCallbackQuery(ctx context.Context, req telegram.AnswerCallbackQueryRequest) error
}

type Service struct {
	store            Store
	messenger        Messenger
	botUsername      string
	loginBaseURL     string
	groupTeamMap     map[int64]string
	initialSitoneIDs []string
	requestTTL       time.Duration
	now              func() time.Time
	newNonce         func() (string, error)
	newAuthToken     func() (string, error)
	newQRCodeToken   func() (string, error)
	log              *slog.Logger
}

type Dependencies struct {
	Store            Store
	Messenger        Messenger
	BotUsername      string
	LoginBaseURL     string
	GroupTeamMap     map[int64]string
	InitialSitoneIDs []string
	RequestTTL       time.Duration
	Now              func() time.Time
	NewNonce         func() (string, error)
	NewAuthToken     func() (string, error)
	NewQRCodeToken   func() (string, error)
	Log              *slog.Logger
}

func NewService(dep Dependencies) (*Service, error) {
	if dep.Store == nil {
		return nil, errors.New("store is required")
	}
	if dep.Messenger == nil {
		return nil, errors.New("messenger is required")
	}
	if strings.TrimSpace(dep.BotUsername) == "" {
		return nil, errors.New("bot username is required")
	}
	if strings.TrimSpace(dep.LoginBaseURL) == "" {
		return nil, errors.New("login base url is required")
	}
	if len(dep.GroupTeamMap) == 0 {
		return nil, errors.New("group team map is required")
	}
	if len(dep.InitialSitoneIDs) == 0 {
		return nil, errors.New("initial sitone ids are required")
	}
	if dep.RequestTTL <= 0 {
		dep.RequestTTL = defaultRequestTTL
	}
	if dep.Now == nil {
		dep.Now = time.Now
	}
	if dep.NewNonce == nil {
		dep.NewNonce = func() (string, error) {
			return randomURLToken(nonceBytes)
		}
	}
	if dep.NewAuthToken == nil {
		dep.NewAuthToken = func() (string, error) {
			token, err := randomURLToken(authTokenBytes)
			if err != nil {
				return "", err
			}
			return "tg_" + token, nil
		}
	}
	if dep.NewQRCodeToken == nil {
		dep.NewQRCodeToken = func() (string, error) {
			token, err := randomURLToken(qrCodeTokenBytes)
			if err != nil {
				return "", err
			}
			return "qr_" + token, nil
		}
	}
	if dep.Log == nil {
		dep.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	groupTeamMap := make(map[int64]string, len(dep.GroupTeamMap))
	for chatID, teamID := range dep.GroupTeamMap {
		groupTeamMap[chatID] = teamID
	}
	initialSitoneIDs := append([]string(nil), dep.InitialSitoneIDs...)

	return &Service{
		store:            dep.Store,
		messenger:        dep.Messenger,
		botUsername:      strings.TrimPrefix(strings.TrimSpace(dep.BotUsername), "@"),
		loginBaseURL:     strings.TrimSpace(dep.LoginBaseURL),
		groupTeamMap:     groupTeamMap,
		initialSitoneIDs: initialSitoneIDs,
		requestTTL:       dep.RequestTTL,
		now:              dep.Now,
		newNonce:         dep.NewNonce,
		newAuthToken:     dep.NewAuthToken,
		newQRCodeToken:   dep.NewQRCodeToken,
		log:              dep.Log,
	}, nil
}

func (s *Service) HandleUpdate(ctx context.Context, update telegram.Update) error {
	if update.CallbackQuery != nil {
		return s.handleLoginCallback(ctx, *update.CallbackQuery)
	}

	message := update.Message
	if message == nil || message.From == nil || message.From.IsBot {
		return nil
	}

	switch message.Chat.Type {
	case "group", "supergroup":
		if _, ok := parseGroupLoginCommand(message.Text, s.botUsername); !ok {
			return nil
		}
		return s.handleGroupStart(ctx, *message)
	case "private":
		payload, ok := parseStartCommand(message.Text, s.botUsername)
		if !ok {
			return nil
		}
		return s.handlePrivateStart(ctx, *message, payload)
	default:
		return nil
	}
}

func (s *Service) handleLoginCallback(ctx context.Context, callback telegram.CallbackQuery) error {
	if callback.From.IsBot {
		return nil
	}

	telegramUserID, nonce, ok := parseLoginCallbackData(callback.Data)
	if !ok {
		return nil
	}
	if telegramUserID != callback.From.ID {
		return s.messenger.AnswerCallbackQuery(ctx, telegram.AnswerCallbackQueryRequest{
			CallbackQueryID: callback.ID,
			Text:            "這個按鈕不是你的，請輸入 vip-666 產生自己的登入連結。",
			ShowAlert:       true,
		})
	}

	deepLink := telegramDeepLink(s.botUsername, nonce)
	answerErr := s.messenger.AnswerCallbackQuery(ctx, telegram.AnswerCallbackQueryRequest{
		CallbackQueryID: callback.ID,
		URL:             deepLink,
	})

	if callback.Message != nil {
		if callback.Message.ReplyToMessage != nil {
			s.deleteMessage(ctx, callback.Message.Chat.ID, callback.Message.ReplyToMessage.MessageID, "login_original")
		}
		s.deleteMessage(ctx, callback.Message.Chat.ID, callback.Message.MessageID, "login_generated")
	}

	if answerErr != nil {
		return fmt.Errorf("answer login callback: %w", answerErr)
	}
	return nil
}

func (s *Service) handleGroupStart(ctx context.Context, message telegram.Message) error {
	s.deleteMessage(ctx, message.Chat.ID, message.MessageID, "login_command")

	teamID, ok := s.groupTeamMap[message.Chat.ID]
	if !ok {
		return s.messenger.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID: message.Chat.ID,
			Text:   "這個群組尚未設定遊戲隊伍，請聯絡工作人員。",
		})
	}

	nonce, err := s.newNonce()
	if err != nil {
		return fmt.Errorf("create login nonce: %w", err)
	}
	now := s.now().UTC()
	request := domain.LoginRequest{
		ID:             nonce,
		TelegramUserID: message.From.ID,
		TelegramChatID: message.Chat.ID,
		TeamID:         teamID,
		CreatedAt:      now,
		ExpiresAt:      now.Add(s.requestTTL),
	}
	if err := s.store.CreateLoginRequest(ctx, request); err != nil {
		return fmt.Errorf("store login request: %w", err)
	}

	return s.messenger.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:    message.Chat.ID,
		Text:      "🎟 " + telegramUserMention(*message.From) + " 的 VIP 傳送門已開啟！點下面按鈕私訊 Bot，領取你的專屬登入連結。",
		ParseMode: parseModeHTML,
		ReplyMarkup: &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{{
				{
					Text:         "私訊取得登入連結",
					CallbackData: loginCallbackData(message.From.ID, nonce),
				},
			}},
		},
	})
}

func (s *Service) handlePrivateStart(ctx context.Context, message telegram.Message, payload string) error {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return s.messenger.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID: message.Chat.ID,
			Text:   "請先在你的小隊群組輸入 vip-666，再點按鈕回到這裡取得登入連結。",
		})
	}

	now := s.now().UTC()
	request, err := s.store.RedeemLoginRequest(ctx, payload, message.From.ID, now)
	if errors.Is(err, domain.ErrLoginRequestNotFound) {
		return s.messenger.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID: message.Chat.ID,
			Text:   "這個登入請求已失效，請回到小隊群組重新輸入 vip-666。",
		})
	}
	if err != nil {
		return fmt.Errorf("redeem login request: %w", err)
	}

	authToken, err := s.newAuthToken()
	if err != nil {
		return fmt.Errorf("create auth token: %w", err)
	}
	qrCodeToken, err := s.newQRCodeToken()
	if err != nil {
		return fmt.Errorf("create qr code token: %w", err)
	}
	player, created, err := s.store.GetOrCreatePlayer(ctx, domain.CreatePlayerInput{
		PlayerID:         playerID(message.From.ID),
		AuthToken:        authToken,
		QRCodeToken:      qrCodeToken,
		Nickname:         nickname(*message.From),
		TeamID:           request.TeamID,
		InitialSitoneIDs: s.initialSitoneIDs,
		TelegramUserID:   message.From.ID,
		TelegramUsername: strings.TrimSpace(message.From.Username),
		TelegramChatID:   request.TelegramChatID,
		Now:              now,
	})
	if err != nil {
		return fmt.Errorf("get or create player: %w", err)
	}
	if player.AuthToken == "" || player.TeamID == "" || player.Nickname == "" {
		return fmt.Errorf("telegram player %q is missing required login fields", player.ID)
	}

	loginURL, err := loginURL(s.loginBaseURL, player.AuthToken)
	if err != nil {
		return err
	}

	escapedLoginURL := html.EscapeString(loginURL)
	escapedAuthToken := html.EscapeString(player.AuthToken)
	text := "🎟 VIP-666 通關碼驗證成功！\n\n" +
		"你的專屬遊戲入口已開啟：<a href=\"" + escapedLoginURL + "\">點我登入遊戲</a>\n\n" +
		"如果按鈕或連結無法使用，請用這組 Token 登入：`" + escapedAuthToken + "`\n\n" +
		"⚠️ 這張 VIP 通行證只認本人，請不要轉傳給其他人。"
	if !created && player.TeamID != request.TeamID {
		text = "ℹ️ 你已經綁定在 " + html.EscapeString(player.TeamID) + "，登入連結仍沿用原本隊伍。\n\n" + text
	}
	return s.messenger.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:                message.Chat.ID,
		Text:                  text,
		ParseMode:             parseModeHTML,
		DisableWebPagePreview: true,
		ReplyMarkup: &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{{
				{
					Text: "🚪 打開遊戲入口",
					URL:  loginURL,
				},
			}},
		},
	})
}

func parseStartCommand(text string, botUsername string) (string, bool) {
	return parseBotCommand(text, botUsername, "start")
}

func parseGroupLoginCommand(text string, _ string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return "", false
	}
	if !strings.EqualFold(fields[0], groupLoginCommand) {
		return "", false
	}
	if len(fields) == 1 {
		return "", true
	}
	return fields[1], true
}

func parseBotCommand(text string, botUsername string, wantName string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return "", false
	}

	command := fields[0]
	if !strings.HasPrefix(command, "/") {
		return "", false
	}
	command = strings.TrimPrefix(command, "/")

	name := command
	mention := ""
	if before, after, ok := strings.Cut(command, "@"); ok {
		name = before
		mention = after
	}
	if !strings.EqualFold(name, wantName) {
		return "", false
	}
	if mention != "" && !strings.EqualFold(mention, strings.TrimPrefix(botUsername, "@")) {
		return "", false
	}
	if len(fields) == 1 {
		return "", true
	}
	return fields[1], true
}

func loginCallbackData(telegramUserID int64, nonce string) string {
	return fmt.Sprintf("%s%d:%s", loginCallbackPrefix, telegramUserID, nonce)
}

func parseLoginCallbackData(data string) (int64, string, bool) {
	payload, ok := strings.CutPrefix(strings.TrimSpace(data), loginCallbackPrefix)
	if !ok {
		return 0, "", false
	}
	userIDRaw, nonce, ok := strings.Cut(payload, ":")
	if !ok || strings.TrimSpace(nonce) == "" {
		return 0, "", false
	}
	telegramUserID, err := strconv.ParseInt(userIDRaw, 10, 64)
	if err != nil || telegramUserID == 0 {
		return 0, "", false
	}
	return telegramUserID, nonce, true
}

func telegramDeepLink(botUsername string, nonce string) string {
	parsed := url.URL{
		Scheme: "https",
		Host:   "t.me",
		Path:   strings.TrimPrefix(strings.TrimSpace(botUsername), "@"),
	}
	query := parsed.Query()
	query.Set("start", nonce)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func loginURL(baseURL string, token string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse login base url: %w", err)
	}
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func playerID(telegramUserID int64) string {
	return fmt.Sprintf("tg_%d", telegramUserID)
}

func nickname(user telegram.User) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(user.FirstName) != "" {
		parts = append(parts, strings.TrimSpace(user.FirstName))
	}
	if strings.TrimSpace(user.LastName) != "" {
		parts = append(parts, strings.TrimSpace(user.LastName))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if strings.TrimSpace(user.Username) != "" {
		return "@" + strings.TrimPrefix(strings.TrimSpace(user.Username), "@")
	}
	return fmt.Sprintf("Player %d", user.ID)
}

func telegramUserMention(user telegram.User) string {
	label := nickname(user)
	if username := strings.TrimPrefix(strings.TrimSpace(user.Username), "@"); username != "" {
		label = "@" + username
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, user.ID, html.EscapeString(label))
}

func (s *Service) deleteMessage(ctx context.Context, chatID int64, messageID int, messageType string) {
	if chatID == 0 || messageID == 0 {
		return
	}
	if err := s.messenger.DeleteMessage(ctx, telegram.DeleteMessageRequest{
		ChatID:    chatID,
		MessageID: messageID,
	}); err != nil {
		s.log.Warn("telegram message deletion failed",
			"chat_id", chatID,
			"message_id", messageID,
			"message_type", messageType,
			"error", err,
		)
	}
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
