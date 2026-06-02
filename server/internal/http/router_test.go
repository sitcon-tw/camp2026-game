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
		{method: http.MethodGet, path: "/api/me"},
		{method: http.MethodGet, path: "/api/me/state"},
		{method: http.MethodGet, path: "/api/me/home"},
		{method: http.MethodGet, path: "/api/me/qrcode"},
		{method: http.MethodGet, path: "/api/me/sitones"},
		{method: http.MethodGet, path: "/api/me/sitones/S9K2QA"},
		{method: http.MethodGet, path: "/api/me/items"},
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
		{method: http.MethodGet, path: "/api/matches"},
		{method: http.MethodPost, path: "/api/matches"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/answers"},
		{method: http.MethodPost, path: "/api/matches/M8RXP2/finish"},
		{method: http.MethodGet, path: "/api/matches/M8RXP2/ws"},
		{method: http.MethodGet, path: "/api/shop/items"},
		{method: http.MethodGet, path: "/api/shop/items/item-upgrade-stone"},
		{method: http.MethodPost, path: "/api/shop/purchases"},
		{method: http.MethodGet, path: "/api/storage"},
		{method: http.MethodGet, path: "/api/storage/sitones"},
		{method: http.MethodGet, path: "/api/storage/recipes"},
		{method: http.MethodGet, path: "/api/crafting/recipes"},
		{method: http.MethodGet, path: "/api/crafting/recipes/recipe_engineering_skin"},
		{method: http.MethodPost, path: "/api/crafting"},
		{method: http.MethodGet, path: "/api/catalog/sitones"},
		{method: http.MethodGet, path: "/api/catalog/items"},
		{method: http.MethodGet, path: "/api/catalog/crafting-recipes"},
		{method: http.MethodGet, path: "/api/catalog/recipes"},
		{method: http.MethodPost, path: "/api/staff/rewards"},
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
		"/healthz",
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
		"/me/qrcode",
		"/me/sitones",
		"/me/items",
		"/me/open-power",
		"/me/home",
		"/users/state",
		"/activities",
		"/activities/{activityID}",
		"/activities/{activityID}/claims",
		"/bingo/boards",
		"/match-pairings",
		"/matches",
		"/matches/{matchID}",
		"/matches/{matchID}/answers",
		"/matches/{matchID}/finish",
		"/qrcode/me",
		"/qrcode/scans",
		"/shop/items",
		"/shop/items/{itemID}",
		"/shop/purchases",
		"/world-bosses",
		"/crafting",
		"/crafting/recipes",
		"/crafting/recipes/{recipeID}",
		"/storage",
		"/storage/sitones",
		"/storage/recipes",
		"/catalog/sitones",
		"/catalog/items",
		"/catalog/crafting-recipes",
		"/catalog/recipes",
		"/staff/rewards",
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

func performRequest(handler http.Handler, method, path string, body *strings.Reader) *httptest.ResponseRecorder {
	if body == nil {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}
