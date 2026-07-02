package staff

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestCurrentStaffRequiresStaffRole(t *testing.T) {
	req := authenticatedRequest(mongomodel.Player{ID: "player-a"})
	res := httptest.NewRecorder()

	_, ok := currentStaff(res, req)
	if ok {
		t.Fatal("expected non-staff player to be rejected")
	}
	assertProblem(t, res, http.StatusForbidden)
}

func TestCurrentStaffAcceptsStaffRole(t *testing.T) {
	req := authenticatedRequest(mongomodel.Player{ID: "staff-a", Role: authctx.PlayerRoleStaff})
	res := httptest.NewRecorder()

	player, ok := currentStaff(res, req)
	if !ok {
		t.Fatalf("expected staff player to be accepted: %s", res.Body.String())
	}
	if player.ID != "staff-a" {
		t.Fatalf("expected staff player id, got %q", player.ID)
	}
}

func TestRewardDefinitionFindsSitone(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})

	reward, ok := handler.rewardDefinition(rewardKindSitone, "stone_engineering_base")
	if !ok {
		t.Fatal("expected sitone reward definition")
	}
	if reward.id != "stone_engineering_base" || reward.name != "工程型小石" {
		t.Fatalf("unexpected reward definition: %#v", reward)
	}
}

func TestRewardDefinitionFindsEnabledItem(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})

	reward, ok := handler.rewardDefinition(rewardKindItem, "item_adventure_backpack")
	if !ok {
		t.Fatal("expected item reward definition")
	}
	if reward.id != "item_adventure_backpack" || reward.name != "冒險背包" {
		t.Fatalf("unexpected reward definition: %#v", reward)
	}
}

func TestRewardDefinitionRejectsMissingContent(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})

	if _, ok := handler.rewardDefinition(rewardKindSitone, "missing"); ok {
		t.Fatal("expected missing sitone to be rejected")
	}
	if _, ok := handler.rewardDefinition(rewardKindItem, "missing"); ok {
		t.Fatal("expected missing item to be rejected")
	}
}

func TestPlayerSearchFilterMatchesNicknameAndID(t *testing.T) {
	filter := playerSearchFilter("Alice.1")

	if got := filter["role"]; got != nil {
		t.Fatalf("expected no role filter, got %#v", got)
	}
	branches, ok := filter["$or"].(bson.A)
	if !ok || len(branches) != 2 {
		t.Fatalf("expected nickname and id search branches, got %#v", filter["$or"])
	}
	for _, branch := range branches {
		condition, ok := branch.(bson.M)
		if !ok {
			t.Fatalf("expected branch to be bson.M, got %#v", branch)
		}
		for _, value := range condition {
			regex, ok := value.(bson.Regex)
			if !ok {
				t.Fatalf("expected regex condition, got %#v", value)
			}
			if regex.Pattern != regexp.QuoteMeta("Alice.1") || regex.Options != "i" {
				t.Fatalf("unexpected regex: %#v", regex)
			}
		}
	}
}

func TestStaffPlayerResponsesIncludesStaffTargets(t *testing.T) {
	responses := staffPlayerResponses(
		[]mongomodel.Player{
			{ID: "P1", Nickname: "Alice", TeamID: "T1"},
			{ID: "S1", Nickname: "Staff", Role: authctx.PlayerRoleStaff},
			{ID: "", Nickname: "Missing"},
		},
		map[string]mongomodel.Team{
			"T1": {ID: "T1", Name: "Blue Team"},
		},
	)

	if len(responses) != 2 {
		t.Fatalf("expected two player responses, got %#v", responses)
	}
	if responses[0].PlayerID != "P1" || responses[0].Nickname != "Alice" {
		t.Fatalf("unexpected player response: %#v", responses[0])
	}
	if responses[0].Team == nil || responses[0].Team.TeamID != "T1" || responses[0].Team.Name != "Blue Team" {
		t.Fatalf("expected team in response, got %#v", responses[0].Team)
	}
	if responses[1].PlayerID != "S1" || responses[1].Nickname != "Staff" {
		t.Fatalf("unexpected staff response: %#v", responses[1])
	}
}

func TestInventoryCollection(t *testing.T) {
	collection, field, err := inventoryCollection(rewardKindSitone)
	if err != nil {
		t.Fatalf("sitone inventory collection: %v", err)
	}
	if collection != mongomodel.PlayerSitonesCollection || field != "sitone_id" {
		t.Fatalf("unexpected sitone inventory collection: %s %s", collection, field)
	}

	collection, field, err = inventoryCollection(rewardKindItem)
	if err != nil {
		t.Fatalf("item inventory collection: %v", err)
	}
	if collection != mongomodel.PlayerItemsCollection || field != "item_id" {
		t.Fatalf("unexpected item inventory collection: %s %s", collection, field)
	}
}

func authenticatedRequest(player mongomodel.Player) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/staff/rewards", strings.NewReader(""))
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
