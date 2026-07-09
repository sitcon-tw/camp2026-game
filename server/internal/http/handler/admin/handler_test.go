package admin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
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
	if cookie.MaxAge != 365*24*60*60 {
		t.Fatalf("expected admin cookie max age to be one year, got %d", cookie.MaxAge)
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

func TestSortPlayerSitonesForBalanceTrimPrefersQuantityThenBaseThenID(t *testing.T) {
	handler := New(Dependencies{Content: testcontent.Load(t)})
	sitones := []mongomodel.PlayerSitone{
		{SitoneID: "stone_gpg", Quantity: 4},
		{SitoneID: "stone_explorer_base", Quantity: 5},
		{SitoneID: "stone_espresso", Quantity: 6},
		{SitoneID: "stone_engineering_base", Quantity: 5},
		{SitoneID: "stone_docker", Quantity: 5},
	}

	handler.sortPlayerSitonesForBalanceTrim(sitones)

	got := make([]string, 0, len(sitones))
	for _, sitone := range sitones {
		got = append(got, sitone.SitoneID)
	}
	want := []string{
		"stone_espresso",
		"stone_engineering_base",
		"stone_explorer_base",
		"stone_docker",
		"stone_gpg",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected trim order: got %#v want %#v", got, want)
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

func TestSettingsResponseIncludesSameTeamBattleToggle(t *testing.T) {
	got := settingsResponse(gamecontrol.Settings{})
	if !got.SameTeamBattlesEnabled {
		t.Fatal("expected same-team battles to be enabled by default")
	}

	got = settingsResponse(gamecontrol.Settings{SameTeamBattlesDisabled: true})
	if got.SameTeamBattlesEnabled {
		t.Fatal("expected same-team battles to be disabled when stored flag is set")
	}
}

func TestSettingsResponseIncludesMaintenanceMode(t *testing.T) {
	got := settingsResponse(gamecontrol.Settings{
		MaintenanceMode:    gamecontrol.MaintenanceModeDraining,
		MaintenanceMessage: "Deploying",
	})

	if !got.MaintenanceActive || got.MaintenanceMode != gamecontrol.MaintenanceModeDraining {
		t.Fatalf("expected draining maintenance response, got %#v", got)
	}
	if got.MaintenanceMessage != "Deploying" || !got.BattleOpeningLocked {
		t.Fatalf("expected maintenance message and battle lock, got %#v", got)
	}
}

func TestSettingsResponseIncludesClassTimeBattleLockSessions(t *testing.T) {
	got := settingsResponse(gamecontrol.Settings{
		ClassTimeBattleLockSessions: []gamecontrol.ClockTimeRange{
			{Start: "09:00", End: "10:30"},
			{Start: "13:00", End: "14:30"},
		},
	})

	if len(got.ClassTimeBattleLockSessions) != 2 {
		t.Fatalf("expected two class time lock sessions, got %#v", got.ClassTimeBattleLockSessions)
	}
	if got.ClassTimeBattleLockStart != "09:00" || got.ClassTimeBattleLockEnd != "10:30" {
		t.Fatalf("expected legacy response fields to mirror first session, got %#v", got)
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

func TestHistoryRequiresAdminCookie(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/history", nil)
	res := httptest.NewRecorder()

	handler.History(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestHistoryRejectsUnsupportedBucket(t *testing.T) {
	handler := New(Dependencies{AdminPassword: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/history?bucket=minute", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: adminSessionValue("secret")})
	res := httptest.NewRecorder()

	handler.History(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestNormalizeUpdateTeamRequestTrimsFields(t *testing.T) {
	got := normalizeUpdateTeamRequest(UpdateTeamRequest{
		Name:      "  Blue Team  ",
		AvatarURL: "  /game-icons/teams/blue.png  ",
	})

	if got.Name != "Blue Team" || got.AvatarURL != "/game-icons/teams/blue.png" {
		t.Fatalf("unexpected normalized update team request: %#v", got)
	}
}

func TestValidateUpdateTeamRequest(t *testing.T) {
	valid := UpdateTeamRequest{Name: "Blue Team", AvatarURL: "https://example.test/avatar/blue.png"}
	if details := validateUpdateTeamRequest("T1", valid); len(details) != 0 {
		t.Fatalf("expected valid team update request, got %#v", details)
	}

	invalid := UpdateTeamRequest{Name: strings.Repeat("x", teamNameMaxLen+1), AvatarURL: "ftp://example.test/avatar.png"}
	details := validateUpdateTeamRequest("", invalid)
	if len(details) != 3 {
		t.Fatalf("expected team id, name, and avatar validation errors, got %#v", details)
	}
}

func TestValidTeamAvatarURLAllowsEmptyHTTPAndRootRelative(t *testing.T) {
	for _, value := range []string{"", "https://example.test/avatar.png", "http://example.test/avatar.png", "/game-icons/teams/blue.png"} {
		if !validTeamAvatarURL(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}

	for _, value := range []string{"ftp://example.test/avatar.png", "//example.test/avatar.png", "avatar.png"} {
		if validTeamAvatarURL(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestUpdateTeamResponseIncludesAvatarURL(t *testing.T) {
	got := updateTeamResponse(mongomodel.Team{ID: "T1", Name: "Blue Team", AvatarURL: "/avatar.png"})
	if got.TeamID != "T1" || got.Name != "Blue Team" || got.AvatarURL != "/avatar.png" {
		t.Fatalf("unexpected update team response: %#v", got)
	}
}

func TestNormalizeUpdateCommunityStandRequestTrimsFields(t *testing.T) {
	enabled := true
	got := normalizeUpdateCommunityStandRequest(UpdateCommunityStandRequest{
		Name:        "  社群攤位  ",
		Description: "  介紹社群  ",
		LogoURL:     "  /game-icons/features/team.png  ",
		WebsiteURL:  "  https://sitcon.org  ",
		Enabled:     &enabled,
		Reward: CommunityStandRewardRequest{
			Kind:  "  item  ",
			RefID: "  item_booth_sticker  ",
		},
	})

	if got.Name != "社群攤位" || got.Description != "介紹社群" || got.LogoURL != "/game-icons/features/team.png" || got.WebsiteURL != "https://sitcon.org" {
		t.Fatalf("unexpected normalized community stand request: %#v", got)
	}
	if got.Reward.Kind != "item" || got.Reward.RefID != "item_booth_sticker" {
		t.Fatalf("unexpected normalized reward request: %#v", got.Reward)
	}
}

func TestNormalizeCreateCommunityStandRequestTrimsFields(t *testing.T) {
	enabled := true
	got := normalizeCreateCommunityStandRequest(CreateCommunityStandRequest{
		UpdateCommunityStandRequest: UpdateCommunityStandRequest{
			Name:        "  社群攤位  ",
			Description: "  介紹社群  ",
			LogoURL:     "  /game-icons/features/team.png  ",
			WebsiteURL:  "  https://sitcon.org  ",
			Enabled:     &enabled,
			Reward: CommunityStandRewardRequest{
				Kind:  "  item  ",
				RefID: "  item_booth_sticker  ",
			},
		},
	})

	if got.Name != "社群攤位" || got.Description != "介紹社群" || got.LogoURL != "/game-icons/features/team.png" || got.WebsiteURL != "https://sitcon.org" {
		t.Fatalf("unexpected normalized community stand create request: %#v", got)
	}
	if got.Reward.Kind != "item" || got.Reward.RefID != "item_booth_sticker" {
		t.Fatalf("unexpected normalized reward request: %#v", got.Reward)
	}
}

func TestValidateCreateCommunityStandRequest(t *testing.T) {
	handler := New(Dependencies{Content: loadAdminTestContent(t)})
	enabled := true
	valid := CreateCommunityStandRequest{
		UpdateCommunityStandRequest: UpdateCommunityStandRequest{
			Name:        "社群攤位",
			Description: "介紹學生社群。",
			Enabled:     &enabled,
			Reward: CommunityStandRewardRequest{
				Kind:     standRewardKindItem,
				RefID:    "item_booth_sticker",
				Quantity: 1,
			},
		},
	}
	if details := handler.validateCreateCommunityStandRequest(valid); len(details) != 0 {
		t.Fatalf("expected valid community stand create request, got %#v", details)
	}
}

func TestValidateUpdateCommunityStandRequest(t *testing.T) {
	handler := New(Dependencies{Content: loadAdminTestContent(t)})
	enabled := true
	valid := UpdateCommunityStandRequest{
		Name:        "社群攤位",
		Description: "介紹學生社群。",
		LogoURL:     "/game-icons/features/team.png",
		WebsiteURL:  "https://sitcon.org",
		Enabled:     &enabled,
		Reward: CommunityStandRewardRequest{
			Kind:     standRewardKindItem,
			RefID:    "item_booth_sticker",
			Quantity: 1,
		},
	}
	if details := handler.validateUpdateCommunityStandRequest("stand-a", valid); len(details) != 0 {
		t.Fatalf("expected valid community stand update request, got %#v", details)
	}

	invalid := UpdateCommunityStandRequest{
		Name:        strings.Repeat("x", communityStandNameMaxLen+1),
		Description: "",
		LogoURL:     "logo.png",
		WebsiteURL:  "/local/path",
		Reward: CommunityStandRewardRequest{
			Kind:     standRewardKindItem,
			RefID:    "missing-item",
			Quantity: 0,
		},
	}
	details := handler.validateUpdateCommunityStandRequest("", invalid)
	if len(details) != 8 {
		t.Fatalf("expected stand id, enabled, name, description, urls, item ref, and quantity errors, got %#v", details)
	}
}

func TestValidateUpdateCommunityStandOpenPowerReward(t *testing.T) {
	handler := New(Dependencies{Content: loadAdminTestContent(t)})
	enabled := false
	valid := UpdateCommunityStandRequest{
		Name:        "開源力攤位",
		Description: "完成任務後取得開源力。",
		Enabled:     &enabled,
		Reward: CommunityStandRewardRequest{
			Kind:   standRewardKindOpenPower,
			Amount: 50,
		},
	}
	if details := handler.validateUpdateCommunityStandRequest("stand-a", valid); len(details) != 0 {
		t.Fatalf("expected valid open power reward, got %#v", details)
	}

	valid.Reward.Amount = 0
	if details := handler.validateUpdateCommunityStandRequest("stand-a", valid); len(details) != 1 {
		t.Fatalf("expected amount validation error, got %#v", details)
	}
}

func TestCommunityStandRewardResponseIncludesContentMetadata(t *testing.T) {
	handler := New(Dependencies{Content: loadAdminTestContent(t)})

	got, ok := handler.communityStandRewardResponse(mongomodel.StandReward{
		Kind:     standRewardKindItem,
		RefID:    "item_booth_sticker",
		Quantity: 1,
	})
	if !ok || got.Name != "攤位貼紙" || got.IconPath != "/game-icons/items/item_booth_sticker.png" {
		t.Fatalf("unexpected item reward response: %#v, ok=%v", got, ok)
	}

	got, ok = handler.communityStandRewardResponse(mongomodel.StandReward{
		Kind:   standRewardKindOpenPower,
		Amount: 50,
	})
	if !ok || got.Name != "開源力" || got.Amount != 50 {
		t.Fatalf("unexpected open power reward response: %#v, ok=%v", got, ok)
	}
}

func TestCommunityStandClaimResponseIncludesClaimDetails(t *testing.T) {
	handler := New(Dependencies{Content: loadAdminTestContent(t)})
	createdAt := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	claim := mongomodel.CommunityStandClaim{
		ID:        "community_claim_1",
		StandID:   "stand-a",
		PlayerID:  "player-a",
		RewardID:  "community_claim_1",
		CreatedAt: createdAt,
		Reward: mongomodel.StandReward{
			Kind:     standRewardKindItem,
			RefID:    "item_booth_sticker",
			Quantity: 2,
		},
	}

	got := handler.communityStandClaimResponse(
		claim,
		map[string]mongomodel.Player{"player-a": {ID: "player-a", Nickname: "Alice"}},
		map[string]mongomodel.CommunityStand{"stand-a": {ID: "stand-a", Name: "社群攤位"}},
	)

	if got.ClaimID != "community_claim_1" || got.StandID != "stand-a" || got.StandName != "社群攤位" {
		t.Fatalf("unexpected stand claim identifiers: %#v", got)
	}
	if got.PlayerID != "player-a" || got.PlayerNickname != "Alice" || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected stand claim player/time: %#v", got)
	}
	if got.Reward.Kind != standRewardKindItem || got.Reward.RefID != "item_booth_sticker" || got.Reward.Quantity != 2 || got.Reward.Name != "攤位貼紙" {
		t.Fatalf("unexpected stand claim reward: %#v", got.Reward)
	}
}

func loadAdminTestContent(t *testing.T) *content.Store {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"sitones.toml": strings.TrimSpace(`
[[sitones]]
id = "stone_booth"
name = "攤位小石"
type = "exploration"
rarity = "common"
style = "default"
description = "完成攤位任務取得的小石。"
ability_name = "攤位探索"
ability_kind = "material_drop_rate"
ability_value = 5
ability_count = 0
ability_description = "素材掉落率提高 5%。"
attack = 12
defense = 5
repeatable = true
unique = false
effect_kind = "scout_recon"
effect_value = 12
`),
		"items.toml": strings.TrimSpace(`
[[items]]
id = "item_booth_sticker"
name = "攤位貼紙"
type = "material"
rarity = "common"
description = "攤位貼紙，可用於小石合成。"
icon_path = "/game-icons/items/item_booth_sticker.png"
source = "shop"
purchasable = true
enabled = true
price_open_power = 150
`),
		"fusion_recipes.toml": strings.TrimSpace(`
[[fusion_recipes]]
id = "recipe_booth_stone"
name = "Booth Stone"
description = "Fixture recipe."
enabled = true

[[fusion_recipes.inputs]]
kind = "item"
id = "item_booth_sticker"
quantity = 1

[[fusion_recipes.outputs]]
kind = "sitone"
id = "stone_booth"
quantity = 1
`),
		"quiz_questions.csv": strings.TrimSpace(`
id,prompt,choice_a,choice_b,choice_c,choice_d,correct_choice,explanation
quiz-001,Question 1,A,B,C,D,A,Explanation 1
quiz-002,Question 2,A,B,C,D,B,Explanation 2
quiz-003,Question 3,A,B,C,D,C,Explanation 3
quiz-004,Question 4,A,B,C,D,D,Explanation 4
quiz-005,Question 5,A,B,C,D,A,Explanation 5
quiz-006,Question 6,A,B,C,D,B,Explanation 6
quiz-007,Question 7,A,B,C,D,C,Explanation 7
quiz-008,Question 8,A,B,C,D,D,Explanation 8
quiz-009,Question 9,A,B,C,D,A,Explanation 9
quiz-010,Question 10,A,B,C,D,B,Explanation 10
`),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	store, err := content.Load(dir)
	if err != nil {
		t.Fatalf("load test content: %v", err)
	}
	return store
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

func TestDashboardMatchAnswerStatsPipelineAggregatesKnownPlayers(t *testing.T) {
	playerIDs := []string{"player-a", "player-b"}

	got := dashboardMatchAnswerStatsPipeline(playerIDs)
	want := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "answer_count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "correct_answer_count", Value: dashboardConditionalSum(dashboardFieldEquals("correct", true))},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$score"}}},
			{Key: "elapsed_ms", Value: bson.D{{Key: "$sum", Value: "$elapsed_ms"}}},
			{Key: "last_activity_at", Value: bson.D{{Key: "$max", Value: "$answered_at"}}},
		}}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected match answer stats pipeline:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestBuildDashboardHistoryResponseUsesBaselineAndCumulativeDeltas(t *testing.T) {
	now := time.Date(2026, 7, 6, 10, 45, 0, 0, time.UTC)
	firstHour := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	secondHour := time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC)
	currentHour := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)

	got := buildDashboardHistoryResponse(now, dashboardHistoryBucketHour, dashboardHistoryRawData{
		CurrentSitoneTotal: 10,
		SitoneDeltas: []dashboardHistoryDeltaStat{
			{Timestamp: firstHour, Amount: 2},
			{Timestamp: secondHour, Amount: -1},
			{Timestamp: secondHour, Amount: 3},
		},
		OpenPowerDeltas: []dashboardHistoryDeltaStat{
			{Timestamp: firstHour, Amount: 100},
			{Timestamp: secondHour, Amount: -30},
		},
	})

	if got.Bucket != dashboardHistoryBucketHour || got.SitoneBaseline != 6 || got.OpenPowerStart != 0 {
		t.Fatalf("unexpected history metadata: %#v", got)
	}
	want := []DashboardHistoryPointResponse{
		{Timestamp: firstHour, SitoneCount: 8, OpenPower: 100, SitoneDelta: 2, OpenPowerDelta: 100},
		{Timestamp: secondHour, SitoneCount: 10, OpenPower: 70, SitoneDelta: 2, OpenPowerDelta: -30},
		{Timestamp: currentHour, SitoneCount: 10, OpenPower: 70},
	}
	if !reflect.DeepEqual(got.Points, want) {
		t.Fatalf("unexpected history points:\ngot  %#v\nwant %#v", got.Points, want)
	}
}

func TestBuildDashboardHistoryResponseIncludesInventoryTrimSitoneDeltas(t *testing.T) {
	now := time.Date(2026, 7, 7, 17, 30, 0, 0, time.UTC)
	grantHour := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)
	trimHour := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	currentHour := time.Date(2026, 7, 7, 17, 0, 0, 0, time.UTC)

	got := buildDashboardHistoryResponse(now, dashboardHistoryBucketHour, dashboardHistoryRawData{
		CurrentSitoneTotal: 882,
		SitoneDeltas: []dashboardHistoryDeltaStat{
			{Timestamp: grantHour, Amount: 1200},
			{Timestamp: trimHour, Amount: -500},
		},
	})

	if got.SitoneBaseline != 182 {
		t.Fatalf("expected positive baseline after accounting for trim deltas, got %#v", got)
	}
	want := []DashboardHistoryPointResponse{
		{Timestamp: grantHour, SitoneCount: 1382, SitoneDelta: 1200},
		{Timestamp: trimHour, SitoneCount: 882, SitoneDelta: -500},
		{Timestamp: currentHour, SitoneCount: 882},
	}
	if !reflect.DeepEqual(got.Points, want) {
		t.Fatalf("unexpected history points:\ngot  %#v\nwant %#v", got.Points, want)
	}
}

func TestDashboardHistoryInventoryTrimSitoneDeltaPipeline(t *testing.T) {
	playerIDs := []string{"player-a", "player-b"}

	got := dashboardHistoryInventoryTrimSitoneDeltaPipeline(dashboardHistoryBucketHour, playerIDs)
	want := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}},
			{Key: "sitone_count", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", dashboardHistoryBucketHour)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{"$sitone_count", -1}}}}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected inventory trim sitone history pipeline:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestDashboardHistoryOpenPowerDeltaPipeline(t *testing.T) {
	playerIDs := []string{"player-a", "player-b"}

	got := dashboardHistoryOpenPowerDeltaPipeline(dashboardHistoryBucketDay, playerIDs)
	want := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", dashboardHistoryBucketDay)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected open power history pipeline:\ngot  %#v\nwant %#v", got, want)
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
		PlayerSitones: []dashboardPlayerQuantityStat{
			{PlayerID: "player-a", Quantity: 2},
			{PlayerID: "player-b", Quantity: 1},
			{PlayerID: "staff-a", Quantity: 99},
		},
		PlayerItems: []dashboardPlayerQuantityStat{
			{PlayerID: "player-b", Quantity: 3},
			{PlayerID: "staff-a", Quantity: 99},
		},
		SitoneInventory: []dashboardInventoryStat{
			{ID: "stone-a", Quantity: 102, OwnerCount: 3},
		},
		ItemInventory: []dashboardInventoryStat{
			{ID: "item-a", Quantity: 102, OwnerCount: 2},
		},
		OpenPower: []dashboardPlayerOpenPowerStat{
			{PlayerID: "player-a", Amount: 50, LastActivityAt: now.Add(-3 * time.Hour)},
			{PlayerID: "player-b", Amount: 10, LastActivityAt: now.Add(-2 * time.Hour)},
			{PlayerID: "staff-a", Amount: 999, LastActivityAt: now.Add(-1 * time.Hour)},
		},
		MatchSummary: dashboardMatchSummaryStat{
			Total:     2,
			Active:    1,
			Completed: 1,
			PVP:       1,
			Computer:  1,
		},
		MatchPlayers: []dashboardMatchPlayerStat{
			{PlayerID: "player-a", MatchCount: 2, CompletedMatchCount: 1, LastActivityAt: now.Add(-30 * time.Minute)},
			{PlayerID: "player-b", MatchCount: 1, CompletedMatchCount: 1, LastActivityAt: now.Add(-4 * time.Hour)},
		},
		RecentMatches: []mongomodel.Match{
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
		},
		MatchAnswers: []dashboardMatchAnswerStat{
			{PlayerID: "player-a", AnswerCount: 1, CorrectAnswerCount: 1, Score: 100, ElapsedMillis: 1000, LastActivityAt: now.Add(-4*time.Hour + time.Minute)},
			{PlayerID: "player-b", AnswerCount: 1, Score: 20, ElapsedMillis: 3000, LastActivityAt: now.Add(-4*time.Hour + 2*time.Minute)},
			{PlayerID: "staff-a", AnswerCount: 1, CorrectAnswerCount: 1, Score: 999, ElapsedMillis: 1, LastActivityAt: now},
		},
		MatchItemDrops: []dashboardMatchItemDropStat{
			{PlayerID: "player-a", DropAttempts: 1, DropSuccesses: 1, LastActivityAt: now.Add(-4 * time.Hour)},
			{PlayerID: "player-b", DropAttempts: 1, LastActivityAt: now.Add(-4 * time.Hour)},
		},
		ShopPurchases: []dashboardPlayerActivityStat{
			{PlayerID: "player-a", Count: 1, LastActivityAt: now.Add(-90 * time.Minute)},
			{PlayerID: "staff-a", Count: 1, LastActivityAt: now},
		},
		FusionRecords: []dashboardPlayerActivityStat{
			{PlayerID: "player-b", Count: 1, LastActivityAt: now.Add(-80 * time.Minute)},
		},
		StaffRewards: []dashboardPlayerActivityStat{
			{PlayerID: "player-b", Count: 1, LastActivityAt: now.Add(-70 * time.Minute)},
			{PlayerID: "staff-a", Count: 1, LastActivityAt: now},
		},
	}

	response := buildDashboardResponse(now, nil, raw)

	if response.Summary.PlayerCount != 4 || response.Summary.StaffCount != 1 {
		t.Fatalf("unexpected player/staff counts: %#v", response.Summary)
	}
	if response.Summary.TotalSitones != 102 || response.Summary.TotalItems != 102 || response.Summary.TotalOpenPower != 1059 {
		t.Fatalf("expected staff inventory and power to be included, got %#v", response.Summary)
	}
	if response.Summary.MedianOpenPower != 30 {
		t.Fatalf("unexpected median open power: %#v", response.Summary)
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
