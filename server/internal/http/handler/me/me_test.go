package me

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestQRCodeResponse(t *testing.T) {
	handler := New(Dependencies{})
	req := authenticatedRequest(mongomodel.Player{
		ID:          "7H9K2Q",
		AuthToken:   "auth_token_123456",
		QRCodeToken: "qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok",
	})
	res := httptest.NewRecorder()

	handler.QRCode(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["qrcodeToken"] != "qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok" {
		t.Fatalf("expected qr code identifier, got %#v", body)
	}
	if _, ok := body["authToken"]; ok {
		t.Fatalf("expected auth token to be omitted, got %#v", body)
	}
}

func TestQRCodeRequiresPlayerContext(t *testing.T) {
	handler := New(Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/api/me/qrcode", nil)
	res := httptest.NewRecorder()

	handler.QRCode(res, req)

	assertProblem(t, res, http.StatusUnauthorized)
}

func TestQRCodeRequiresToken(t *testing.T) {
	handler := New(Dependencies{})
	req := authenticatedRequest(mongomodel.Player{ID: "7H9K2Q"})
	res := httptest.NewRecorder()

	handler.QRCode(res, req)

	assertProblem(t, res, http.StatusInternalServerError)
}

func TestStatusResponse(t *testing.T) {
	team := mongomodel.Team{
		ID:   "8M4RXP",
		Name: "Blue Team",
	}
	response := statusResponse(
		mongomodel.Player{
			ID:        "7H9K2Q",
			Nickname:  "Alice",
			TeamID:    "8M4RXP",
			AvatarURL: "https://example.test/avatar/alice.png",
		},
		&team,
		1280,
		[]mongomodel.Player{
			{
				ID:        "7H9K2Q",
				Nickname:  "Alice",
				TeamID:    "8M4RXP",
				AvatarURL: "https://example.test/avatar/alice.png",
			},
			{
				ID:        "2QK9H7",
				Nickname:  "Bob",
				TeamID:    "8M4RXP",
				AvatarURL: "https://example.test/avatar/bob.png",
			},
			{
				ID:       "staff-token-1",
				Nickname: "Staff",
				TeamID:   "8M4RXP",
				Role:     "staff",
			},
		},
	)

	if response.PlayerID != "7H9K2Q" {
		t.Fatalf("expected player id, got %q", response.PlayerID)
	}
	if response.Team == nil {
		t.Fatal("expected team")
	}
	if response.Team.TeamID != "8M4RXP" {
		t.Fatalf("expected team id, got %q", response.Team.TeamID)
	}
	if response.OpenPower != 1280 {
		t.Fatalf("expected open power 1280, got %d", response.OpenPower)
	}
	if len(response.TeamMembers) != 3 {
		t.Fatalf("expected 3 team members, got %#v", response.TeamMembers)
	}
	if response.TeamMembers[0].PlayerID != "7H9K2Q" || response.TeamMembers[0].Nickname != "Alice" {
		t.Fatalf("unexpected first team member: %#v", response.TeamMembers[0])
	}
	if response.TeamMembers[1].PlayerID != "2QK9H7" || response.TeamMembers[1].Nickname != "Bob" {
		t.Fatalf("unexpected second team member: %#v", response.TeamMembers[1])
	}
	if response.TeamMembers[2].PlayerID != "staff-token-1" || response.TeamMembers[2].Role != "staff" {
		t.Fatalf("unexpected third team member: %#v", response.TeamMembers[2])
	}
	if response.AvatarURL == "" {
		t.Fatalf("expected avatar url")
	}
	if response.Role != "" {
		t.Fatalf("expected empty role, got %q", response.Role)
	}
}

func TestStaffStatusResponseKeepsStaffRoleWhenNoTeamIsLoaded(t *testing.T) {
	response := statusResponse(
		mongomodel.Player{
			ID:       "staff-token-1",
			Nickname: "Staff",
			TeamID:   "team-001",
			Role:     "staff",
		},
		nil,
		0,
		nil,
	)

	if response.Team != nil {
		t.Fatalf("expected staff team to be omitted, got %#v", response.Team)
	}
	if len(response.TeamMembers) != 0 {
		t.Fatalf("expected staff team members to be empty, got %#v", response.TeamMembers)
	}
	if response.Role != "staff" {
		t.Fatalf("expected staff role, got %q", response.Role)
	}
}

func TestTeamMemberResponsesSkipsInvalidPlayers(t *testing.T) {
	members := teamMemberResponses([]mongomodel.Player{
		{ID: "7H9K2Q", Nickname: "Alice", AuthToken: "secret", QRCodeToken: "qr-secret"},
		{ID: "", Nickname: "Missing ID"},
		{ID: "2QK9H7"},
		{ID: "staff-token-1", Nickname: "Staff", Role: "staff"},
	})

	if len(members) != 2 {
		t.Fatalf("expected 2 team members, got %#v", members)
	}
	if members[0].PlayerID != "7H9K2Q" || members[0].Nickname != "Alice" {
		t.Fatalf("unexpected team member: %#v", members[0])
	}
	if members[1].PlayerID != "staff-token-1" || members[1].Role != "staff" {
		t.Fatalf("unexpected staff team member: %#v", members[1])
	}
}

func TestHomeActions(t *testing.T) {
	actions := homeActions()
	if len(actions) != 8 {
		t.Fatalf("expected 8 home actions, got %#v", actions)
	}
	for _, action := range actions {
		if action.ID == "" || action.Label == "" || !action.Enabled {
			t.Fatalf("expected enabled action with id and label, got %#v", action)
		}
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

func TestPlayerSitoneCountsPipeline(t *testing.T) {
	pipeline := playerSitoneCountsPipeline()
	if len(pipeline) != 2 {
		t.Fatalf("expected 2 pipeline stages, got %#v", pipeline)
	}
}

func TestOpenPowerScoresByPlayerPipeline(t *testing.T) {
	pipeline := openPowerScoresByPlayerPipeline()
	if len(pipeline) != 1 {
		t.Fatalf("expected 1 pipeline stage, got %#v", pipeline)
	}
}

func TestTeamRankEntriesRankBySitoneThenOpenPower(t *testing.T) {
	teams := []mongomodel.Team{
		{ID: "team-a", Name: "Alpha"},
		{ID: "team-b", Name: "Beta"},
		{ID: "team-c", Name: "Gamma"},
	}
	players := []mongomodel.Player{
		{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		{ID: "player-b", Nickname: "Bob", TeamID: "team-b"},
		{ID: "player-c", Nickname: "Cody", TeamID: "team-c"},
		{ID: "staff-a", Nickname: "Staff", TeamID: "team-a", Role: authctx.PlayerRoleStaff},
	}
	stats := map[string]teamRankStats{
		"player-a": {SitoneCount: 2, OpenPower: 500},
		"player-b": {SitoneCount: 3, OpenPower: 10},
		"player-c": {SitoneCount: 2, OpenPower: 700},
		"staff-a":  {SitoneCount: 99, OpenPower: 99},
	}

	rows := teamRankEntries(teams, players, stats)
	current := currentTeamRank(rows, "team-a")

	wantIDs := []string{"team-a", "team-b", "team-c"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("expected %d rows, got %#v", len(wantIDs), rows)
	}
	for index, wantID := range wantIDs {
		if rows[index].TeamID != wantID || rows[index].Rank != index+1 {
			t.Fatalf("unexpected row at %d: got %#v want id %q rank %d", index, rows[index], wantID, index+1)
		}
	}
	if current == nil || current.TeamID != "team-a" {
		t.Fatalf("expected current team-a rank, got %#v", current)
	}
	if current.SitoneCount != 101 || current.OpenPower != 599 {
		t.Fatalf("expected staff stats to be included, got %#v", current)
	}
	if current.GapToPrevious != 0 {
		t.Fatalf("expected same sitone-count gap 0, got %d", current.GapToPrevious)
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

func TestMapPlayerSitones(t *testing.T) {
	sitones, err := mapPlayerSitones(loadTestContent(t), []mongomodel.PlayerSitone{
		{
			ID:       "owned-sitone-001",
			PlayerID: "7H9K2Q",
			SitoneID: "stone_engineering_base",
			Quantity: 1,
		},
	})
	if err != nil {
		t.Fatalf("map sitones: %v", err)
	}
	if len(sitones) != 1 {
		t.Fatalf("expected 1 sitone, got %d", len(sitones))
	}
	if sitones[0].Sitone.Name != "工程型小石" {
		t.Fatalf("expected catalog sitone name, got %#v", sitones[0])
	}
}

func TestMapPlayerSitonesSkipsMissingCatalogDefinition(t *testing.T) {
	sitones, err := mapPlayerSitones(loadTestContent(t), []mongomodel.PlayerSitone{
		{ID: "owned-sitone-001", SitoneID: "sitone-missing", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("map sitones: %v", err)
	}
	if len(sitones) != 0 {
		t.Fatalf("expected missing catalog sitone to be skipped, got %#v", sitones)
	}
}

func TestNormalizeSitoneLoadoutAllowsDuplicateSlots(t *testing.T) {
	got, err := normalizeSitoneLoadout([]string{
		" stone_engineering_base ",
		"stone_engineering_base",
		"",
	})
	if err != nil {
		t.Fatalf("normalize sitone loadout: %v", err)
	}

	want := []string{"stone_engineering_base", "stone_engineering_base"}
	if len(got) != len(want) {
		t.Fatalf("expected %d sitones, got %#v", len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected sitone at index %d: got %q want %q", index, got[index], want[index])
		}
	}
}

func TestMapPlayerItems(t *testing.T) {
	items, err := mapPlayerItems(loadTestContent(t), []mongomodel.PlayerItem{
		{
			ID:       "owned-item-001",
			PlayerID: "7H9K2Q",
			ItemID:   "item_adventure_backpack",
			Quantity: 3,
		},
	})
	if err != nil {
		t.Fatalf("map items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Item.Name != "冒險背包" {
		t.Fatalf("expected catalog item name, got %#v", items[0])
	}
}

func TestMapPlayerItemsSkipsMissingCatalogDefinition(t *testing.T) {
	items, err := mapPlayerItems(loadTestContent(t), []mongomodel.PlayerItem{
		{ID: "owned-item-001", ItemID: "item-missing", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("map items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected missing catalog item to be skipped, got %#v", items)
	}
}

func TestMapPlayerItemsReturnsEmptySlice(t *testing.T) {
	items, err := mapPlayerItems(loadTestContent(t), nil)
	if err != nil {
		t.Fatalf("map items: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestCompletedMatchesFilterOnlyReturnsCurrentPlayerCompletedMatches(t *testing.T) {
	got := completedMatchesFilter("7H9K2Q")
	want := bson.D{
		{Key: "status", Value: mongomodel.MatchStatusCompleted},
		{Key: "players.player_id", Value: "7H9K2Q"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected completed matches filter: %#v", got)
	}
}

func TestMapCompletedMatches(t *testing.T) {
	completedAt := testTime(t, "2026-06-12T06:30:00Z")
	records := []mongomodel.Match{
		{
			ID:           "match_123",
			Status:       mongomodel.MatchStatusCompleted,
			HostPlayerID: "P1",
			Players: []mongomodel.MatchPlayer{
				{
					PlayerID:  "P1",
					Nickname:  "Alice",
					Score:     850,
					SitoneIDs: []string{"stone_engineering_base"},
				},
				{
					PlayerID: "P2",
					Nickname: "Bob",
					Score:    700,
				},
			},
			QuestionIDs: []string{"quiz-001", "quiz-002"},
			CompletedAt: completedAt,
		},
	}

	matches := mapCompletedMatches(records)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %#v", matches)
	}
	if matches[0].MatchID != "match_123" ||
		matches[0].Status != mongomodel.MatchStatusCompleted ||
		matches[0].QuestionCount != 2 ||
		matches[0].CompletedAt == nil {
		t.Fatalf("unexpected completed match response: %#v", matches[0])
	}
	if len(matches[0].Players) != 2 || matches[0].Players[0].Score != 850 {
		t.Fatalf("unexpected completed match players: %#v", matches[0].Players)
	}
}

func authenticatedRequest(player mongomodel.Player) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/me/qrcode", strings.NewReader(""))
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

func testTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse test time: %v", err)
	}
	return parsed
}
