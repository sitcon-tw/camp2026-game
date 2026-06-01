package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestRemovedRoutes(t *testing.T) {
	router := NewRouter(Dependencies{})

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/"},
		{method: http.MethodGet, path: "/api/me/home"},
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
		"/users/state",
		"/bingo/boards",
		"/matches",
		"/match-pairings",
		"/qrcode/me",
		"/world-bosses",
		"/storage",
		"/storage/sitones",
		"/storage/recipes",
		"/catalog/sitones",
		"/staff/rewards",
		"/staff/activity-verifications",
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
	for _, path := range []string{"/", "/readyz", "/ping", "/examples/validation", "/me/home", "/quiz/pairings", "/quiz/sessions", "/world-bosses/challenges/{challengeID}/answers"} {
		if _, ok := spec.Paths[path]; ok {
			t.Fatalf("expected swagger json not to contain path %q", path)
		}
	}
	assertSwaggerSecurity(t, spec.Paths, "/healthz", http.MethodGet, false)
	assertSwaggerSecurity(t, spec.Paths, "/auth/login", http.MethodPost, false)
	for _, route := range []struct {
		path   string
		method string
	}{
		{path: "/users/state", method: http.MethodGet},
		{path: "/auth/logout", method: http.MethodPost},
		{path: "/staff/rewards", method: http.MethodPost},
		{path: "/staff/activity-verifications", method: http.MethodPost},
	} {
		assertSwaggerSecurity(t, spec.Paths, route.path, route.method, true)
	}
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

func TestGameDesignGetRoutes(t *testing.T) {
	router := NewRouter(Dependencies{})

	for _, path := range []string{
		"/api/users/state",
		"/api/bingo/boards",
		"/api/matches",
		"/api/matches/match_01HR9Z7E2Z2VJ2QZ4P4Z",
		"/api/qrcode/me",
		"/api/world-bosses",
		"/api/world-bosses/boss_layer_1",
		"/api/storage",
		"/api/storage/sitones",
		"/api/storage/items",
		"/api/storage/recipes",
		"/api/catalog/sitones",
		"/api/catalog/items",
		"/api/catalog/recipes",
	} {
		res := performRequest(router, http.MethodGet, path, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("%s: expected status %d, got %d: %s", path, http.StatusOK, res.Code, res.Body.String())
		}
		if contentType := res.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("%s: expected json content type, got %q", path, contentType)
		}
	}
}

func TestRecipeListContractUsesSingularRequirements(t *testing.T) {
	router := NewRouter(Dependencies{})

	for _, path := range []string{"/api/storage/recipes", "/api/catalog/recipes"} {
		res := performRequest(router, http.MethodGet, path, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("%s: expected status %d, got %d: %s", path, http.StatusOK, res.Code, res.Body.String())
		}

		body := res.Body.String()
		for _, want := range []string{
			`"requiredSitoneType":"engineering"`,
			`"requiredItemId":"item-camp-sticker"`,
			`"requiredItemQuantity":1`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("%s: expected body to contain %s: %s", path, want, body)
			}
		}
		for _, removed := range []string{"requiredSitoneTypes", "requiredItemIds"} {
			if strings.Contains(body, removed) {
				t.Fatalf("%s: expected body not to contain %q: %s", path, removed, body)
			}
		}
	}
}

func TestGameDesignPostRoutesAreContractStubs(t *testing.T) {
	router := NewRouter(Dependencies{})

	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "complete bingo mission",
			path: "/api/bingo/missions/mission_daily_match_3/complete",
			body: `{"flag":"CAMP2026-HELLO"}`,
		},
		{
			name: "create match pairing",
			path: "/api/match-pairings",
			body: `{"targetQrCodeToken":"player_qr_token"}`,
		},
		{
			name: "create match",
			path: "/api/matches",
			body: `{"mode":"qr_duel","pairingId":"pair_01HR9Z7E2Z2VJ2QZ4P4Z","sitoneIds":["sitone_01HR9Z7E2Z2VJ2QZ4P4Z"]}`,
		},
		{
			name: "submit match answer",
			path: "/api/matches/match_01HR9Z7E2Z2VJ2QZ4P4Z/answers",
			body: `{"questionId":"question_001","choiceId":"A"}`,
		},
		{
			name: "scan qrcode",
			path: "/api/qrcode/scans",
			body: `{"token":"player_qr_token","context":"match_pairing"}`,
		},
		{
			name: "create world boss match",
			path: "/api/world-bosses/boss_layer_1/matches",
			body: `{"sitoneIds":["sitone_01HR9Z7E2Z2VJ2QZ4P4Z"]}`,
		},
		{
			name: "craft sitone",
			path: "/api/storage/crafting",
			body: `{"recipeId":"recipe_engineering_skin","sitoneId":"sitone_01HR9Z7E2Z2VJ2QZ4P4Z","itemIds":["pit_01HR9Z7E2Z2VJ2QZ4P4Z"]}`,
		},
		{
			name: "claim bingo line reward",
			path: "/api/bingo/line-rewards/line_reward_row_0/claim",
			body: `{}`,
		},
		{
			name: "grant staff reward",
			path: "/api/staff/rewards",
			body: `{"targetQrCodeToken":"player_qr_token","reason":"camp_activity_reward","openPower":100}`,
		},
		{
			name: "verify staff activity",
			path: "/api/staff/activity-verifications",
			body: `{"targetQrCodeToken":"player_qr_token","activityCode":"booth-linux-101","missionId":"mission_activity_linux_101"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := performRequest(router, http.MethodPost, tt.path, strings.NewReader(tt.body))
			assertProblem(t, res, http.StatusNotImplemented, "")
		})
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

func performRequest(handler http.Handler, method, path string, body *strings.Reader) *httptest.ResponseRecorder {
	if body == nil {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}
