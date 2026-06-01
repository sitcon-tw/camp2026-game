package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestLoginRequiresDatabase(t *testing.T) {
	handler := New(Dependencies{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"token":"auth_token_123456"}`))
	res := httptest.NewRecorder()
	handler.Login(res, req)

	problem := assertProblem(t, res, http.StatusServiceUnavailable)
	if problem.Detail != "database is unavailable" {
		t.Fatalf("expected database unavailable detail, got %q", problem.Detail)
	}
}

func TestLoginValidatesToken(t *testing.T) {
	handler := New(Dependencies{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"token":"short"}`))
	res := httptest.NewRecorder()
	handler.Login(res, req)

	assertProblem(t, res, http.StatusUnprocessableEntity)
}

func TestSetAuthCookie(t *testing.T) {
	res := httptest.NewRecorder()

	setAuthCookie(res, "auth_token_123456")

	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "camp2026_auth" {
		t.Fatalf("expected auth cookie name, got %q", cookie.Name)
	}
	if cookie.Value != "auth_token_123456" {
		t.Fatalf("expected auth token cookie value, got %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected cookie path /, got %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatalf("expected cookie to be HttpOnly")
	}
	if !cookie.Secure {
		t.Fatalf("expected cookie to be Secure")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
}

func TestLoginResponseFromPlayer(t *testing.T) {
	response := loginResponse(
		mongomodel.Player{
			ID:          "7H9K2Q",
			AuthToken:   "auth_token_123456",
			QRCodeToken: "qr_token_123456",
			Nickname:    "Alice",
			TeamID:      "8M4RXP",
			AvatarURL:   "https://example.test/avatar/alice.png",
		},
		mongomodel.Team{
			ID:   "8M4RXP",
			Name: "Blue Team",
		},
		1280,
	)

	if response.Player.PlayerID != "7H9K2Q" {
		t.Fatalf("expected player id, got %q", response.Player.PlayerID)
	}
	if response.Player.Team.TeamID != "8M4RXP" {
		t.Fatalf("expected team id, got %q", response.Player.Team.TeamID)
	}
	if response.Player.Team.Name != "Blue Team" {
		t.Fatalf("expected team name, got %q", response.Player.Team.Name)
	}
	if response.Player.OpenPower != 1280 {
		t.Fatalf("expected open power 1280, got %d", response.Player.OpenPower)
	}
	if response.Player.AvatarURL == "" {
		t.Fatalf("expected avatar url")
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

func TestOpenPowerTotalFromCursorWithoutRecords(t *testing.T) {
	cursor, err := mongo.NewCursorFromDocuments(nil, nil, nil)
	if err != nil {
		t.Fatalf("new cursor: %v", err)
	}

	total, err := openPowerTotalFromCursor(context.Background(), cursor)
	if err != nil {
		t.Fatalf("open power total: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
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
