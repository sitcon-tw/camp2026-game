# Camp 2026 Telegram Bot

Telegram login helper for the Camp 2026 game.

## Flow

1. A player sends `vip-666` in one of the configured team Telegram groups.
2. The bot deletes the player's `vip-666` message and posts a tagged group message with a button that opens a private chat deep link.
3. After the player clicks the button, the bot deletes its generated group button message.
4. The bot redeems the short-lived request in private chat.
5. The bot creates or reuses a player bound to that Telegram user and sends a short HTML hyperlink plus a bottom URL button:

```text
VIP-666 通關碼驗證成功！
點我登入遊戲
如果按鈕或連結無法使用，請用這組 Token 登入：`tg_...`
[打開遊戲入口]
```

The real auth token is never sent to the group.

## Config

```sh
TELEGRAM_BOT_TOKEN=123456:secret
APP_LOGIN_BASE_URL=https://game.example.com/login
TG_GROUP_TEAM_MAP=-1001111111111=team-001,-1002222222222=team-002
INITIAL_SITONE_IDS=stone_explorer_base,stone_inspiration_base,stone_resonance_base,stone_engineering_base,stone_entertainment_base
MONGODB_URI=mongodb://camp2026:camp2026@localhost:27017/camp2026?authSource=admin&replicaSet=rs0
MONGODB_DATABASE=camp2026
```

`TG_GROUP_TEAM_MAP` must explicitly list every allowed Telegram group chat ID
and its game team ID. Production should include all 9 team groups.

Optional settings:

```sh
TELEGRAM_API_BASE_URL=https://api.telegram.org
TG_POLL_TIMEOUT=50s
TG_LOGIN_REQUEST_TTL=10m
TG_HTTP_CLIENT_TIMEOUT=60s
LOG_LEVEL=info
SHUTDOWN_TIMEOUT=10s
```

## Local Run

```sh
go run ./cmd/tgbot
```

## Test

```sh
go test ./...
```
