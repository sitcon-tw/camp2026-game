package leaderboards

import (
	"testing"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestNormalizeScopeDefaultsToTeams(t *testing.T) {
	if got := normalizeScope(""); got != ScopeTeams {
		t.Fatalf("expected default scope %q, got %q", ScopeTeams, got)
	}
}

func TestIsValidScope(t *testing.T) {
	for _, value := range []string{ScopeTeams, ScopePlayers} {
		if !isValidScope(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	if isValidScope("unknown") {
		t.Fatal("expected unknown scope to be invalid")
	}
}

func TestTeamEntriesRankBySitoneThenOpenPowerThenName(t *testing.T) {
	teams := []mongomodel.Team{
		{ID: "team-a", Name: "Alpha"},
		{ID: "team-b", Name: "Beta"},
		{ID: "team-c", Name: "Gamma"},
		{ID: "team-d", Name: "Delta"},
	}
	players := []mongomodel.Player{
		{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		{ID: "player-b", Nickname: "Bob", TeamID: "team-b"},
		{ID: "player-c", Nickname: "Cody", TeamID: "team-c"},
		{ID: "player-d", Nickname: "Dana", TeamID: "team-d"},
		{ID: "staff-a", Nickname: "Staff", TeamID: "team-d", Role: authctx.PlayerRoleStaff},
	}
	stats := map[string]rankStats{
		"player-a": {SitoneCount: 2, OpenPower: 100},
		"player-b": {SitoneCount: 3, OpenPower: 10},
		"player-c": {SitoneCount: 2, OpenPower: 150},
		"player-d": {SitoneCount: 2, OpenPower: 150},
		"staff-a":  {SitoneCount: 99, OpenPower: 99},
	}

	entries := teamEntries(teams, players, stats, "team-c")
	current, gap := currentEntryAndGap(entries)

	wantIDs := []string{"team-d", "team-b", "team-c", "team-a"}
	if len(entries) != len(wantIDs) {
		t.Fatalf("expected %d entries, got %#v", len(wantIDs), entries)
	}
	for index, wantID := range wantIDs {
		if entries[index].ID != wantID || entries[index].Rank != index+1 {
			t.Fatalf("unexpected entry at %d: got %#v want id %q rank %d", index, entries[index], wantID, index+1)
		}
	}
	if entries[0].SitoneCount != 101 || entries[0].OpenPower != 249 {
		t.Fatalf("expected staff stats to be included in team-d, got %#v", entries[0])
	}
	if current == nil || current.ID != "team-c" || !current.Current {
		t.Fatalf("expected current team-c entry, got %#v", current)
	}
	if gap != 1 {
		t.Fatalf("expected sitone-count gap 1, got %d", gap)
	}
}

func TestPlayerEntriesRankBySitoneThenOpenPowerThenName(t *testing.T) {
	teams := []mongomodel.Team{
		{ID: "team-a", Name: "Alpha"},
		{ID: "team-b", Name: "Beta"},
	}
	players := []mongomodel.Player{
		{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		{ID: "player-b", Nickname: "Bob", TeamID: "team-b"},
		{ID: "player-c", Nickname: "Cody", TeamID: "team-a"},
		{ID: "staff-a", Nickname: "Staff", TeamID: "team-a", Role: authctx.PlayerRoleStaff},
		{ID: "ungrouped", Nickname: "No Team"},
	}
	stats := map[string]rankStats{
		"player-a":  {SitoneCount: 3, OpenPower: 10},
		"player-b":  {SitoneCount: 3, OpenPower: 20},
		"player-c":  {SitoneCount: 2, OpenPower: 999},
		"staff-a":   {SitoneCount: 99, OpenPower: 99},
		"ungrouped": {SitoneCount: 99, OpenPower: 99},
	}

	entries := playerEntries(players, teams, stats, "player-a")
	current, gap := currentEntryAndGap(entries)

	wantIDs := []string{"staff-a", "player-b", "player-a", "player-c"}
	if len(entries) != len(wantIDs) {
		t.Fatalf("expected %d entries, got %#v", len(wantIDs), entries)
	}
	for index, wantID := range wantIDs {
		if entries[index].ID != wantID || entries[index].Rank != index+1 {
			t.Fatalf("unexpected entry at %d: got %#v want id %q rank %d", index, entries[index], wantID, index+1)
		}
	}
	if entries[1].TeamName != "Beta" {
		t.Fatalf("expected team name on player entry, got %#v", entries[1])
	}
	if current == nil || current.ID != "player-a" || !current.Current {
		t.Fatalf("expected current player-a entry, got %#v", current)
	}
	if gap != 0 {
		t.Fatalf("expected same sitone-count gap 0, got %d", gap)
	}
}

func TestCurrentEntryGapUsesSitoneCount(t *testing.T) {
	entries := []RankEntryResponse{
		{ID: "first", SitoneCount: 7, Rank: 1},
		{ID: "current", SitoneCount: 4, Rank: 2, Current: true},
	}

	current, gap := currentEntryAndGap(entries)
	if current == nil || current.ID != "current" {
		t.Fatalf("expected current entry, got %#v", current)
	}
	if gap != 3 {
		t.Fatalf("expected gap 3, got %d", gap)
	}
}

func TestLeaderboardPipelines(t *testing.T) {
	if got := len(playerSitoneCountsPipeline()); got != 2 {
		t.Fatalf("expected 2 sitone count stages, got %d", got)
	}
	if got := len(openPowerScoresByPlayerPipeline()); got != 1 {
		t.Fatalf("expected 1 open power stage, got %d", got)
	}
	if got := len(playerItemCountsPipeline()); got != 2 {
		t.Fatalf("expected 2 item count stages, got %d", got)
	}
}

func TestTeamPlayerSummariesRankBySitoneThenOpenPower(t *testing.T) {
	players := []mongomodel.Player{
		{ID: "player-a", Nickname: "Alice", TeamID: "team-a"},
		{ID: "player-b", Nickname: "Bob", TeamID: "team-a"},
		{ID: "staff-a", Nickname: "Staff", TeamID: "team-a", Role: authctx.PlayerRoleStaff},
	}
	stats := map[string]rankStats{
		"player-a": {SitoneCount: 3, OpenPower: 10},
		"player-b": {SitoneCount: 3, OpenPower: 20},
		"staff-a":  {SitoneCount: 99, OpenPower: 99},
	}

	responses := teamPlayerSummaries(players, stats, map[string]int{"player-b": 4}, "player-a")
	if len(responses) != 3 {
		t.Fatalf("expected 3 player responses, got %#v", responses)
	}
	if responses[0].PlayerID != "staff-a" {
		t.Fatalf("expected staff-a first, got %#v", responses[0])
	}
	if responses[1].PlayerID != "player-b" || responses[1].ItemCount != 4 {
		t.Fatalf("expected player-b second with item count, got %#v", responses[1])
	}
	if responses[2].PlayerID != "player-a" || !responses[2].Current {
		t.Fatalf("expected player-a current third, got %#v", responses[2])
	}
}

func TestInventoryItemResponsesSkipMissingCatalogDefinition(t *testing.T) {
	items := inventoryItemResponses(loadTestContent(t), []mongomodel.PlayerItem{
		{ID: "owned-item-001", ItemID: "item_adventure_backpack", Quantity: 3},
		{ID: "owned-item-missing", ItemID: "item-missing", Quantity: 1},
	})
	if len(items) != 1 {
		t.Fatalf("expected one mapped item, got %#v", items)
	}
	if items[0].Item.Name != "冒險背包" || items[0].Quantity != 3 {
		t.Fatalf("unexpected mapped item: %#v", items[0])
	}
}

func TestInventorySitoneResponsesSkipMissingCatalogDefinition(t *testing.T) {
	sitones := inventorySitoneResponses(loadTestContent(t), []mongomodel.PlayerSitone{
		{ID: "owned-sitone-001", SitoneID: "stone_engineering_base", Quantity: 2},
		{ID: "owned-sitone-missing", SitoneID: "stone-missing", Quantity: 1},
	})
	if len(sitones) != 1 {
		t.Fatalf("expected one mapped sitone, got %#v", sitones)
	}
	if sitones[0].Sitone.Name != "工程型小石" || sitones[0].Quantity != 2 {
		t.Fatalf("unexpected mapped sitone: %#v", sitones[0])
	}
}

func TestQuantityTotalsIgnoreNonPositiveQuantities(t *testing.T) {
	if got := quantityTotalItems([]mongomodel.PlayerItem{{Quantity: 3}, {Quantity: 0}, {Quantity: -1}}); got != 3 {
		t.Fatalf("expected item total 3, got %d", got)
	}
	if got := quantityTotalSitones([]mongomodel.PlayerSitone{{Quantity: 2}, {Quantity: 0}, {Quantity: -1}}); got != 2 {
		t.Fatalf("expected sitone total 2, got %d", got)
	}
}

func loadTestContent(t *testing.T) *content.Store {
	t.Helper()

	store := testcontent.Load(t)
	if len(store.ListItems()) == 0 {
		t.Fatalf("expected loaded item catalog")
	}
	return store
}
