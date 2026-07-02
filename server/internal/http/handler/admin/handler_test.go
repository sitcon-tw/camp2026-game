package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestLoginDisabledWithoutAdminPassword(t *testing.T) {
	handler := New(Dependencies{})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"password":"secret"}`))
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}
}

func TestLoginSetsAdminSessionCookie(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"password":"secret"}`))
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %#v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != CookieName || cookie.Value != adminSessionValue("secret") {
		t.Fatalf("unexpected admin cookie: %#v", cookie)
	}
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.MaxAge <= 0 {
		t.Fatalf("expected http-only persistent root cookie, got %#v", cookie)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
	if cookie.Secure {
		t.Fatalf("expected admin cookie to be insecure by default for local development")
	}
}

func TestLoginSetsSecureAdminSessionCookieWhenConfigured(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret", AdminCookieSecure: true})
	req := httptest.NewRequest(http.MethodPost, "http://backend/api/admin/login", strings.NewReader(`{"password":"secret"}`))
	res := httptest.NewRecorder()

	if req.TLS != nil {
		t.Fatalf("expected backend request without TLS")
	}
	handler.Login(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %#v", cookies)
	}
	if cookie := cookies[0]; !cookie.Secure {
		t.Fatalf("expected secure admin cookie behind TLS-terminating proxy, got %#v", cookie)
	}
}

func TestLogoutClearsSecureAdminSessionCookieWhenConfigured(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret", AdminCookieSecure: true})
	req := httptest.NewRequest(http.MethodPost, "http://backend/api/admin/logout", nil)
	res := httptest.NewRecorder()

	handler.Logout(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, res.Code, res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %#v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != CookieName || cookie.Value != "" || cookie.MaxAge >= 0 {
		t.Fatalf("expected expired admin cookie, got %#v", cookie)
	}
	if !cookie.Secure {
		t.Fatalf("expected logout cookie to keep Secure flag, got %#v", cookie)
	}
}

func TestLoginRejectsInvalidPassword(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"password":"wrong"}`))
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestGetSettingsRequiresAdminCookie(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	res := httptest.NewRecorder()

	handler.GetSettings(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestDashboardRequiresAdminCookie(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	res := httptest.NewRecorder()

	handler.Dashboard(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestDashboardRequiresDatabase(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: adminSessionValue("secret")})
	res := httptest.NewRecorder()

	handler.Dashboard(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}
}

func TestDashboardPlayerProjectionFetchesOnlyDashboardFields(t *testing.T) {
	projection := dashboardPlayerProjection()
	included := make(map[string]any, len(projection))
	for _, field := range projection {
		included[field.Key] = field.Value
	}

	expected := map[string]struct{}{
		"_id":        {},
		"nickname":   {},
		"team_id":    {},
		"avatar_url": {},
		"role":       {},
	}
	if len(included) != len(expected) {
		t.Fatalf("expected only dashboard player fields, got %#v", projection)
	}
	for field := range expected {
		if included[field] != 1 {
			t.Fatalf("expected projection to include %q, got %#v", field, projection)
		}
	}
	for _, field := range []string{"auth_token", "qrcode_token", "default_sitone_ids"} {
		if _, ok := included[field]; ok {
			t.Fatalf("projection must not fetch sensitive field %q: %#v", field, projection)
		}
	}
}

func TestBuildDashboardResponseIncludesStaffWithTeamsAndRanksPlayers(t *testing.T) {
	now := time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)
	raw := dashboardRawData{
		Players: []dashboardPlayer{
			{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
			{ID: "player-b", Nickname: "Bob", TeamID: "team-a"},
			{ID: "player-c", Nickname: "Cody"},
			{ID: "staff-a", Nickname: "Staff", TeamID: "team-a", Role: authctx.PlayerRoleStaff},
		},
		Teams: []mongomodel.Team{
			{ID: "team-a", Name: "Alpha"},
		},
		PlayerSitones: []mongomodel.PlayerSitone{
			{PlayerID: "player-a", SitoneID: "stone-a", Quantity: 2},
			{PlayerID: "player-b", SitoneID: "stone-a", Quantity: 1},
			{PlayerID: "staff-a", SitoneID: "stone-a", Quantity: 99},
		},
		PlayerItems: []mongomodel.PlayerItem{
			{PlayerID: "player-b", ItemID: "item-a", Quantity: 3},
			{PlayerID: "staff-a", ItemID: "item-a", Quantity: 99},
		},
		OpenPowerRecords: []mongomodel.OpenPowerRecord{
			{PlayerID: "player-a", Amount: 50, CreatedAt: now.Add(-3 * time.Hour)},
			{PlayerID: "player-b", Amount: 10, CreatedAt: now.Add(-2 * time.Hour)},
			{PlayerID: "staff-a", Amount: 999, CreatedAt: now.Add(-1 * time.Hour)},
		},
		Matches: []mongomodel.Match{
			{
				ID:          "match-a",
				Code:        "123456",
				Mode:        mongomodel.MatchModePVP,
				Status:      mongomodel.MatchStatusCompleted,
				CreatedAt:   now.Add(-5 * time.Hour),
				CompletedAt: now.Add(-4 * time.Hour),
				Players: []mongomodel.MatchPlayer{
					{PlayerID: "player-a", Nickname: "Alice", Score: 100},
					{PlayerID: "player-b", Nickname: "Bob", Score: 80},
				},
			},
			{
				ID:        "match-b",
				Mode:      mongomodel.MatchModeComputer,
				Status:    mongomodel.MatchStatusActive,
				CreatedAt: now.Add(-30 * time.Minute),
				Players: []mongomodel.MatchPlayer{
					{PlayerID: "player-a", Nickname: "Alice"},
					{PlayerID: "computer", Nickname: "Computer", Kind: mongomodel.MatchPlayerKindComputer},
				},
			},
		},
		MatchAnswers: []mongomodel.MatchAnswer{
			{PlayerID: "player-a", Correct: true, Score: 100, ElapsedMillis: 1000, AnsweredAt: now.Add(-4*time.Hour + time.Minute)},
			{PlayerID: "player-b", Correct: false, Score: 20, ElapsedMillis: 3000, AnsweredAt: now.Add(-4*time.Hour + 2*time.Minute)},
			{PlayerID: "staff-a", Correct: true, Score: 999, ElapsedMillis: 1, AnsweredAt: now},
		},
		MatchItemDrops: []mongomodel.MatchItemDrop{
			{PlayerID: "player-a", Dropped: true, CreatedAt: now.Add(-4 * time.Hour)},
			{PlayerID: "player-b", Dropped: false, CreatedAt: now.Add(-4 * time.Hour)},
		},
		ShopPurchases: []mongomodel.ShopPurchase{
			{PlayerID: "player-a", CreatedAt: now.Add(-90 * time.Minute)},
			{PlayerID: "staff-a", CreatedAt: now},
		},
		FusionRecords: []mongomodel.FusionRecord{
			{PlayerID: "player-b", CreatedAt: now.Add(-80 * time.Minute)},
		},
		StaffRewards: []mongomodel.StaffReward{
			{RecipientPlayerID: "player-b", CreatedAt: now.Add(-70 * time.Minute)},
			{RecipientPlayerID: "staff-a", CreatedAt: now},
		},
	}

	response := buildDashboardResponse(now, nil, raw)

	if response.Summary.PlayerCount != 4 || response.Summary.StaffCount != 1 {
		t.Fatalf("unexpected player/staff counts: %#v", response.Summary)
	}
	if response.Summary.TotalSitones != 102 || response.Summary.TotalItems != 102 || response.Summary.TotalOpenPower != 1059 {
		t.Fatalf("expected staff inventory and power to be included, got %#v", response.Summary)
	}
	if response.Summary.AnswerCount != 3 || response.Summary.CorrectAnswerCount != 2 || response.Summary.AnswerAccuracy != 67 {
		t.Fatalf("unexpected answer summary: %#v", response.Summary)
	}
	if response.Players[0].PlayerID != "staff-a" || response.Players[0].Rank != 1 {
		t.Fatalf("expected staff-a to lead sitone ranking, got %#v", response.Players)
	}
	if len(response.Teams) != 1 || response.Teams[0].PlayerCount != 3 || response.Teams[0].SitoneCount != 102 {
		t.Fatalf("unexpected team summary: %#v", response.Teams)
	}
	if len(response.Inventory.Sitones) != 1 || response.Inventory.Sitones[0].Quantity != 102 || response.Inventory.Sitones[0].OwnerCount != 3 {
		t.Fatalf("unexpected sitone inventory summary: %#v", response.Inventory.Sitones)
	}
	if response.Matches.Total != 2 || response.Matches.PVP != 1 || response.Matches.Computer != 1 || response.Matches.DropRate != 50 {
		t.Fatalf("unexpected match summary: %#v", response.Matches)
	}
	if len(response.TopPlayers.ByAccuracy) != 3 || response.TopPlayers.ByAccuracy[1].PlayerID != "staff-a" {
		t.Fatalf("unexpected accuracy ranking: %#v", response.TopPlayers.ByAccuracy)
	}
}
