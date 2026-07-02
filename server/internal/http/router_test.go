package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestHealth(t *testing.T) {
	router := NewRouter(Dependencies{})

	res := performRequest(router, http.MethodGet, "/api/healthz", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if strings.Contains(res.Body.String(), `"database"`) {
		t.Fatalf("expected health response without database check, got %s", res.Body.String())
	}
}

func TestHealthWhenDatabaseUnavailable(t *testing.T) {
	client, err := mongo.Connect(options.Client().
		ApplyURI("mongodb://127.0.0.1:1").
		SetServerSelectionTimeout(time.Millisecond))
	if err != nil {
		t.Fatalf("connect mongo client: %v", err)
	}
	defer func() {
		_ = client.Disconnect(t.Context())
	}()

	router := NewRouter(Dependencies{
		MongoClient: client,
	})

	res := performRequest(router, http.MethodGet, "/api/healthz", nil)
	problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
	if problem.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected problem status %d, got %d", http.StatusServiceUnavailable, problem.Status)
	}
}

func TestAdminLoginUsesConfiguredSecureCookie(t *testing.T) {
	router := NewRouter(Dependencies{
		AdminPassword:     "secret",
		AdminCookieSecure: true,
	})

	res := performRequest(router, http.MethodPost, "/api/admin/login", strings.NewReader(`{"password":"secret"}`))

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %#v", cookies)
	}
	if cookie := cookies[0]; !cookie.Secure {
		t.Fatalf("expected router to pass secure admin cookie setting, got %#v", cookie)
	}
}

func TestRemovedRoutes(t *testing.T) {
	router := NewRouter(Dependencies{})

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/"},
		{method: http.MethodGet, path: "/api/me"},
		{method: http.MethodGet, path: "/api/me/state"},
		{method: http.MethodGet, path: "/api/me/sitones/S9K2QA"},
		{method: http.MethodGet, path: "/api/me/items/I8M4RX"},
		{method: http.MethodGet, path: "/api/me/open-power"},
		{method: http.MethodGet, path: "/api/me/open-power/records"},
		{method: http.MethodGet, path: "/api/users/state"},
		{method: http.MethodPost, path: "/api/qrcode/scans"},
		{method: http.MethodGet, path: "/api/activities"},
		{method: http.MethodGet, path: "/api/activities/booth-linux-101"},
		{method: http.MethodPost, path: "/api/activities/booth-linux-101/claims"},
		{method: http.MethodGet, path: "/api/bingo/boards"},
		{method: http.MethodPost, path: "/api/bingo/missions/mission_daily_match_3/complete"},
		{method: http.MethodGet, path: "/api/qrcode/me"},
		{method: http.MethodGet, path: "/api/world-bosses"},
		{method: http.MethodPost, path: "/api/match-pairings"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/finish"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2/ws"},
		{method: http.MethodGet, path: "/api/storage"},
		{method: http.MethodGet, path: "/api/storage/sitones"},
		{method: http.MethodGet, path: "/api/storage/recipes"},
		{method: http.MethodGet, path: "/api/crafting/recipes"},
		{method: http.MethodGet, path: "/api/crafting/recipes/recipe_engineering_skin"},
		{method: http.MethodPost, path: "/api/crafting"},
		{method: http.MethodGet, path: "/api/catalog/crafting-recipes"},
		{method: http.MethodGet, path: "/api/catalog/recipes"},
		{method: http.MethodPost, path: "/api/staff/activity-verifications"},
		{method: http.MethodGet, path: "/api/readyz"},
		{method: http.MethodGet, path: "/api/ping"},
		{method: http.MethodPost, path: "/api/examples/validation"},
	} {
		res := performRequest(router, route.method, route.path, nil)
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s %s: expected status %d, got %d", route.method, route.path, http.StatusNotFound, res.Code)
		}
	}
}

func TestStaffRoutesRequireAuthentication(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
		MongoDB: fakeDatabase(t),
	})

	for _, route := range []struct {
		method string
		path   string
		body   *strings.Reader
	}{
		{method: http.MethodGet, path: "/api/staff/players?query=Alice"},
		{method: http.MethodPost, path: "/api/staff/rewards", body: strings.NewReader(`{"qrcodeToken":"qr-token-player-a","kind":"sitone","refId":"stone_engineering_base","quantity":1}`)},
		{method: http.MethodPost, path: "/api/qr/resolve", body: strings.NewReader(`{"qrcodeToken":"qr-token-player-a"}`)},
	} {
		res := performRequest(router, route.method, route.path, route.body)
		problem := assertProblem(t, res, http.StatusUnauthorized, "")
		if problem.Status != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected problem status %d, got %d", route.method, route.path, http.StatusUnauthorized, problem.Status)
		}
	}
}

func TestStaffRoutesRequireDatabase(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
	})

	for _, route := range []struct {
		method string
		path   string
		body   *strings.Reader
	}{
		{method: http.MethodGet, path: "/api/staff/players?query=Alice"},
		{method: http.MethodPost, path: "/api/staff/rewards", body: strings.NewReader(`{"qrcodeToken":"qr-token-player-a","kind":"sitone","refId":"stone_engineering_base","quantity":1}`)},
		{method: http.MethodPost, path: "/api/qr/resolve", body: strings.NewReader(`{"qrcodeToken":"qr-token-player-a"}`)},
	} {
		res := performRequestWithCookie(router, route.method, route.path, route.body, "staff_token_2026")
		problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
		if problem.Status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected problem status %d, got %d", route.method, route.path, http.StatusServiceUnavailable, problem.Status)
		}
	}
}

func TestShopRoutesRequireAuthentication(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
		MongoDB: fakeDatabase(t),
	})

	for _, route := range []struct {
		method string
		path   string
		body   *strings.Reader
	}{
		{method: http.MethodGet, path: "/api/shop/items"},
		{method: http.MethodGet, path: "/api/shop/items/item_adventure_backpack"},
		{method: http.MethodPost, path: "/api/shop/purchases", body: strings.NewReader(`{"itemId":"item_adventure_backpack"}`)},
	} {
		res := performRequest(router, route.method, route.path, route.body)
		problem := assertProblem(t, res, http.StatusUnauthorized, "")
		if problem.Status != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected problem status %d, got %d", route.method, route.path, http.StatusUnauthorized, problem.Status)
		}
	}
}

func TestShopRoutesRequireDatabase(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
	})

	for _, route := range []struct {
		method string
		path   string
		body   *strings.Reader
	}{
		{method: http.MethodGet, path: "/api/shop/items"},
		{method: http.MethodGet, path: "/api/shop/items/item_adventure_backpack"},
		{method: http.MethodPost, path: "/api/shop/purchases", body: strings.NewReader(`{"itemId":"item_adventure_backpack"}`)},
	} {
		res := performRequestWithCookie(router, route.method, route.path, route.body, "auth_token_123456")
		problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
		if problem.Status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected problem status %d, got %d", route.method, route.path, http.StatusServiceUnavailable, problem.Status)
		}
	}
}

func TestMatchRoutesRequireAuthentication(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
		MongoDB: fakeDatabase(t),
	})

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/matches"},
		{method: http.MethodGet, path: "/api/matches/computer/settings"},
		{method: http.MethodPost, path: "/api/matches/computer"},
		{method: http.MethodPost, path: "/api/matches/join"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2"},
		{method: http.MethodPut, path: "/api/matches/M8RXP2/loadout"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/ready"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/answers"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2/events"},
	} {
		res := performRequest(router, route.method, route.path, nil)
		problem := assertProblem(t, res, http.StatusUnauthorized, "")
		if problem.Status != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected problem status %d, got %d", route.method, route.path, http.StatusUnauthorized, problem.Status)
		}
	}
}

func TestMatchRoutesRequireDatabase(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
	})

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/matches"},
		{method: http.MethodGet, path: "/api/matches/computer/settings"},
		{method: http.MethodPost, path: "/api/matches/computer"},
		{method: http.MethodPost, path: "/api/matches/join"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2"},
		{method: http.MethodPut, path: "/api/matches/M8RXP2/loadout"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/ready"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/answers"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2/events"},
	} {
		res := performRequestWithCookie(router, route.method, route.path, nil, "auth_token_123456")
		problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
		if problem.Status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected problem status %d, got %d", route.method, route.path, http.StatusServiceUnavailable, problem.Status)
		}
	}
}

func TestMeRoutesRequireAuthentication(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
		MongoDB: fakeDatabase(t),
	})

	for _, path := range []string{
		"/api/me/home",
		"/api/me/status",
		"/api/me/qrcode",
		"/api/me/sitones",
		"/api/me/sitone-loadout",
		"/api/me/items",
		"/api/me/matches",
	} {
		res := performRequest(router, http.MethodGet, path, nil)
		problem := assertProblem(t, res, http.StatusUnauthorized, "")
		if problem.Status != http.StatusUnauthorized {
			t.Fatalf("%s: expected problem status %d, got %d", path, http.StatusUnauthorized, problem.Status)
		}
	}
}

func TestMeRoutesRequireDatabase(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
	})

	for _, path := range []string{
		"/api/me/home",
		"/api/me/status",
		"/api/me/qrcode",
		"/api/me/sitones",
		"/api/me/sitone-loadout",
		"/api/me/items",
		"/api/me/matches",
	} {
		res := performRequestWithCookie(router, http.MethodGet, path, nil, "auth_token_123456")
		problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
		if problem.Status != http.StatusServiceUnavailable {
			t.Fatalf("%s: expected problem status %d, got %d", path, http.StatusServiceUnavailable, problem.Status)
		}
	}
}

func TestListSitoneCatalog(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
	})

	res := performRequest(router, http.MethodGet, "/api/catalog/sitones", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	var body struct {
		Sitones []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Type        string `json:"type"`
			Rarity      string `json:"rarity"`
			Style       string `json:"style"`
			Description string `json:"description"`
		} `json:"sitones"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	sitonesByID := make(map[string]struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Rarity      string `json:"rarity"`
		Style       string `json:"style"`
		Description string `json:"description"`
	}, len(body.Sitones))
	for _, sitone := range body.Sitones {
		sitonesByID[sitone.ID] = sitone
	}

	if _, ok := sitonesByID["stone_engineering_base"]; !ok {
		t.Fatalf("expected imported stone_engineering_base in catalog, got %#v", body.Sitones)
	}
	got, ok := sitonesByID["stone_2026_camp_explorer"]
	if !ok ||
		got.Name != "2026 營地探險小石" ||
		got.Type != "exploration" ||
		got.Rarity != "rare" ||
		got.Description == "" {
		t.Fatalf("expected imported 2026 camp explorer sitone, got %#v", got)
	}
}

func TestListItemCatalog(t *testing.T) {
	router := NewRouter(Dependencies{
		Content: loadTestContent(t),
	})

	res := performRequest(router, http.MethodGet, "/api/catalog/items", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	var body struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Type        string `json:"type"`
			Rarity      string `json:"rarity"`
			Description string `json:"description"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	itemsByID := make(map[string]struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Rarity      string `json:"rarity"`
		Description string `json:"description"`
	}, len(body.Items))
	for _, item := range body.Items {
		itemsByID[item.ID] = item
	}

	got, ok := itemsByID["item_adventure_backpack"]
	if !ok ||
		got.Name != "冒險背包" ||
		got.Type != "material" ||
		got.Rarity != "common" ||
		got.Description == "" {
		t.Fatalf("expected imported adventure backpack item, got %#v", got)
	}
}

func TestListItemCatalogWhenContentUnavailable(t *testing.T) {
	router := NewRouter(Dependencies{})

	res := performRequest(router, http.MethodGet, "/api/catalog/items", nil)
	problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
	if problem.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected problem status %d, got %d", http.StatusServiceUnavailable, problem.Status)
	}
}

func TestListSitoneCatalogWhenContentUnavailable(t *testing.T) {
	router := NewRouter(Dependencies{})

	res := performRequest(router, http.MethodGet, "/api/catalog/sitones", nil)
	problem := assertProblem(t, res, http.StatusServiceUnavailable, "")
	if problem.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected problem status %d, got %d", http.StatusServiceUnavailable, problem.Status)
	}
}

func TestSwaggerJSON(t *testing.T) {
	router := NewRouter(Dependencies{})

	res := performRequest(router, http.MethodGet, "/api/swagger.json", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	body := res.Body.String()
	for _, want := range []string{
		"/auth/login",
		"/auth/logout",
		"/catalog/items",
		"/catalog/sitones",
		"/healthz",
		"/fusions",
		"/fusions/recipes",
		"/leaderboards",
		"/me/items",
		"/me/matches",
		"/me/home",
		"/me/qrcode",
		"/me/sitones",
		"/me/status",
		"/matches",
		"/matches/join",
		"/matches/{matchID}",
		"/matches/{matchID}/answers",
		"/matches/{matchID}/events",
		"/matches/{matchID}/ready",
		"/shop/items",
		"/shop/items/{itemID}",
		"/shop/purchases",
		"/staff/players",
		"/staff/rewards",
		"/qr/resolve",
		"AuthCookieAuth",
		"camp2026_auth",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected swagger json to contain %q", want)
		}
	}

	var spec struct {
		Paths map[string]map[string]struct {
			Security []map[string][]string `json:"security,omitempty"`
		} `json:"paths"`
	}
	if err := json.Unmarshal([]byte(body), &spec); err != nil {
		t.Fatalf("decode swagger json: %v", err)
	}
	for _, path := range []string{
		"/",
		"/readyz",
		"/ping",
		"/examples/validation",
		"/me",
		"/me/state",
		"/me/open-power",
		"/users/state",
		"/activities",
		"/activities/{activityID}",
		"/activities/{activityID}/claims",
		"/bingo/boards",
		"/match-pairings",
		"/matches/{matchID}/finish",
		"/qrcode/me",
		"/qrcode/scans",
		"/world-bosses",
		"/crafting",
		"/crafting/recipes",
		"/crafting/recipes/{recipeID}",
		"/storage",
		"/storage/sitones",
		"/storage/recipes",
		"/catalog/crafting-recipes",
		"/catalog/recipes",
		"/staff/activity-verifications",
		"/quiz/pairings",
		"/quiz/sessions",
		"/world-bosses/challenges/{challengeID}/answers",
	} {
		if _, ok := spec.Paths[path]; ok {
			t.Fatalf("expected swagger json not to contain path %q", path)
		}
	}
	assertSwaggerSecurity(t, spec.Paths, "/healthz", http.MethodGet, false)
	assertSwaggerSecurity(t, spec.Paths, "/auth/login", http.MethodPost, false)
	assertSwaggerSecurity(t, spec.Paths, "/auth/logout", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/catalog/items", http.MethodGet, false)
	assertSwaggerSecurity(t, spec.Paths, "/catalog/sitones", http.MethodGet, false)
	assertSwaggerSecurity(t, spec.Paths, "/qr/resolve", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/fusions", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/fusions/recipes", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/leaderboards", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/me/home", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/me/status", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/me/qrcode", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/me/sitones", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/me/items", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/me/matches", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/matches", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/matches/join", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/matches/{matchID}", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/matches/{matchID}/ready", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/matches/{matchID}/answers", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/matches/{matchID}/events", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/shop/items", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/shop/items/{itemID}", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/shop/purchases", http.MethodPost, true)
	assertSwaggerSecurity(t, spec.Paths, "/staff/players", http.MethodGet, true)
	assertSwaggerSecurity(t, spec.Paths, "/staff/rewards", http.MethodPost, true)
}

func TestScalarDocs(t *testing.T) {
	router := NewRouter(Dependencies{})

	for _, path := range []string{"/api/docs", "/api/docs/index.html"} {
		res := performRequest(router, http.MethodGet, path, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("%s: expected status %d, got %d", path, http.StatusOK, res.Code)
		}
		if contentType := res.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Fatalf("%s: expected html content type, got %q", path, contentType)
		}

		body := res.Body.String()
		for _, want := range []string{
			"@scalar/api-reference",
			"Scalar.createApiReference",
			"/api/swagger.json",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("%s: expected docs html to contain %q", path, want)
			}
		}
	}
}

func TestNotFound(t *testing.T) {
	router := NewRouter(Dependencies{})

	res := performRequest(router, http.MethodGet, "/missing", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}
}

type problemResponse struct {
	Status int `json:"status"`
	Errors []struct {
		Location string `json:"location"`
		Message  string `json:"message"`
	} `json:"errors"`
}

func assertSwaggerSecurity(
	t *testing.T,
	paths map[string]map[string]struct {
		Security []map[string][]string `json:"security,omitempty"`
	},
	path string,
	method string,
	wantSecurity bool,
) {
	t.Helper()

	operations, ok := paths[path]
	if !ok {
		t.Fatalf("expected swagger path %q", path)
	}
	operation, ok := operations[strings.ToLower(method)]
	if !ok {
		t.Fatalf("expected swagger operation %s %s", method, path)
	}

	hasSecurity := false
	for _, security := range operation.Security {
		if _, ok := security["AuthCookieAuth"]; ok {
			hasSecurity = true
			break
		}
	}
	if wantSecurity && !hasSecurity {
		t.Fatalf("expected %s %s to require AuthCookieAuth", method, path)
	}
	if !wantSecurity && len(operation.Security) > 0 {
		t.Fatalf("expected %s %s to have no security, got %v", method, path, operation.Security)
	}
}

func assertProblem(t *testing.T, res *httptest.ResponseRecorder, status int, location string) problemResponse {
	t.Helper()

	if res.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, res.Code, res.Body.String())
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("expected problem content type, got %q", contentType)
	}

	var problem problemResponse
	if err := json.NewDecoder(res.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Status != status {
		t.Fatalf("expected problem status %d, got %d", status, problem.Status)
	}
	if location != "" {
		if len(problem.Errors) == 0 {
			t.Fatalf("expected validation errors")
		}
		if got := problem.Errors[0].Location; got != location {
			t.Fatalf("expected error location %q, got %q", location, got)
		}
	}
	return problem
}

func loadTestContent(t *testing.T) *content.Store {
	t.Helper()

	return testcontent.Load(t)
}

func performRequest(handler http.Handler, method, path string, body *strings.Reader) *httptest.ResponseRecorder {
	if body == nil {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func performRequestWithCookie(handler http.Handler, method, path string, body *strings.Reader, token string) *httptest.ResponseRecorder {
	if body == nil {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	req.AddCookie(&http.Cookie{Name: "camp2026_auth", Value: token})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func fakeDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	client, err := mongo.Connect(options.Client().
		ApplyURI("mongodb://127.0.0.1:1").
		SetServerSelectionTimeout(time.Millisecond))
	if err != nil {
		t.Fatalf("connect mongo client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(t.Context())
	})
	return client.Database("camp2026_game_test")
}
