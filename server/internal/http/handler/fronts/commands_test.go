package fronts

import (
	"strings"
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestCampusTerritorySnapshotUsesConfiguredMap(t *testing.T) {
	store := testcontent.Load(t)
	template, ok := store.GetTerritoryMap("campus_territory_v1")
	if !ok {
		t.Fatal("campus territory template not loaded")
	}
	front := frontFromTerritoryTemplate(template)
	if front.MapMode != contentFrontMapModeTerritoryGrid || front.Territory == nil || front.Territory.Width != 64 || front.Territory.Height != 57 {
		t.Fatalf("unexpected territory front: %#v", front)
	}
	if len(front.Teams) != 9 || len(front.Territory.Bases) != 9 {
		t.Fatalf("expected nine teams and bases, got teams=%d bases=%d", len(front.Teams), len(front.Territory.Bases))
	}
	if len(front.ActiveEvents) != 4 {
		t.Fatalf("expected four actionable landmark events, got %#v", front.ActiveEvents)
	}
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	for _, base := range front.Territory.Bases {
		cell := matrix[base.Y][base.X]
		if cell.Owner != base.TeamID || cell.Defense != 100 {
			t.Fatalf("base is not immutable/owned at startup: base=%#v cell=%#v", base, cell)
		}
	}
	if halo := matrix[7][30]; halo.Owner != "team-001" || halo.Defense != 20 {
		t.Fatalf("expected Manhattan base halo at (30,7), got %#v", halo)
	}
	if diagonal := matrix[8][30]; diagonal.Owner != "" {
		t.Fatalf("base halo must not fill diagonals, got %#v", diagonal)
	}
	foundNeutral := false
	for _, row := range front.Territory.Rows {
		for _, run := range row.Runs {
			if run.OwnerTeamID == "" {
				foundNeutral = true
			}
		}
	}
	if !foundNeutral {
		t.Fatal("expected neutral playable cells to be present in territory rows")
	}
}

func TestWithFrontDefaultsReactivatesConfiguredTerritorySession(t *testing.T) {
	store := testcontent.Load(t)
	handler := New(Dependencies{Content: store})
	front := handler.withFrontDefaults(mongomodel.Front{
		ID: "campus_territory_v1", MapID: "campus_territory_v1",
		MapMode: contentFrontMapModeTerritoryGrid, Current: false,
		Territory: &mongomodel.FrontTerritory{Width: 1, Height: 1, Connectivity: 4},
	})
	if !front.Current {
		t.Fatal("configured territory session must be reactivated for rollout")
	}
}

func TestTerritoryRepairLandmarkRequiresOwnershipAndResolvesOnce(t *testing.T) {
	store := testcontent.Load(t)
	template, _ := store.GetTerritoryMap("campus_territory_v1")
	front := frontFromTerritoryTemplate(template)
	landmark, ok := territoryLandmarkAt(front.Territory.Landmarks, 20, 36)
	if !ok || landmark.Kind != "repair" {
		t.Fatalf("repair landmark missing: %#v", landmark)
	}
	x, y := landmark.X, landmark.Y
	command := mongomodel.FrontCommand{
		ID: "repair-1", PlayerID: "player-1", TeamID: "team-001", Kind: "repair",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	}
	if _, _, err := applyCommandToFront(front, command); err == nil || !strings.Contains(err.Error(), "control the landmark") {
		t.Fatalf("expected ownership error, got %v", err)
	}

	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[y][x].Owner = "team-001"
	matrix[y][x].Defense = 95
	front.Territory.Rows = encodeTerritoryRows(matrix)
	next, _, err := applyCommandToFront(front, command)
	if err != nil {
		t.Fatalf("resolve repair: %v", err)
	}
	nextMatrix, _ := decodeTerritoryRows(next.Territory)
	if nextMatrix[y][x].Defense != 100 || next.Teams[0].RepairedEvents != 1 || next.Teams[0].Score != 30 {
		t.Fatalf("repair outcome incorrect: cell=%#v team=%#v", nextMatrix[y][x], next.Teams[0])
	}
	if activeEventKindAtFrontCell(next, landmark.ID) != "" {
		t.Fatal("repair event should be one-shot")
	}
	if _, _, err := applyCommandToFront(next, command); err == nil || !strings.Contains(err.Error(), "no longer active") {
		t.Fatalf("expected resolved event rejection, got %v", err)
	}
}

func TestApplyTerritoryExpandUsesCoordinateAndConnectedPatch(t *testing.T) {
	front := newTerritoryTestFront(12, 3)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	matrix[1][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 11, 1

	next, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "expand-1", PlayerID: "player-1", TeamID: "team-001", Kind: "expand",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply expand: %v", err)
	}
	if len(command.AffectedCells) != territoryExpandLimit {
		t.Fatalf("expected %d affected cells, got %#v", territoryExpandLimit, command.AffectedCells)
	}
	if next.Teams[0].FrontOpenPower != 90 || next.Teams[0].Score != 0 {
		t.Fatalf("unexpected expand accounting: %#v", next.Teams[0])
	}
	nextMatrix, err := decodeTerritoryRows(next.Territory)
	if err != nil {
		t.Fatal(err)
	}
	for _, coordinate := range command.AffectedCells {
		if nextMatrix[coordinate.Y][coordinate.X].Owner != "team-001" {
			t.Fatalf("affected coordinate was not captured: %#v", coordinate)
		}
	}
	if matrix[1][1].Owner != "" {
		t.Fatal("expected source front to remain unchanged")
	}
}

func TestApplyTerritoryAttackNeverCapturesBase(t *testing.T) {
	front := newTerritoryTestFront(4, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	matrix[0][1] = territoryCell{Playable: true, Owner: "team-002", Defense: 20}
	matrix[0][2] = territoryCell{Playable: true, Owner: "team-002", Defense: 100}
	front.Territory.Bases = []mongomodel.FrontTerritoryBase{{TeamID: "team-002", X: 2, Y: 0, CoreDefense: 100}}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 1, 0

	next, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "attack-1", PlayerID: "player-1", TeamID: "team-001", Kind: "attack",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply attack: %v", err)
	}
	nextMatrix, _ := decodeTerritoryRows(next.Territory)
	if nextMatrix[0][1].Owner != "team-001" {
		t.Fatalf("expected non-base boundary to be captured: %#v", nextMatrix[0][1])
	}
	if nextMatrix[0][2].Owner != "team-002" || nextMatrix[0][2].Defense != 100 {
		t.Fatalf("base changed: %#v", nextMatrix[0][2])
	}
	if len(command.AffectedCells) != 1 || command.AffectedCells[0].X != 1 {
		t.Fatalf("unexpected attack patch: %#v", command.AffectedCells)
	}
	baseX := 2
	if _, _, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "attack-base", PlayerID: "player-1", TeamID: "team-001", Kind: "attack",
		TargetX: &baseX, TargetY: &y, CreatedAt: time.Now().UTC(),
	}); err == nil || !strings.Contains(err.Error(), "cannot be attacked") {
		t.Fatalf("expected direct base attack rejection, got %v", err)
	}
}

func TestApplyTerritoryReinforceRejectsFullAreaWithoutCostOrScore(t *testing.T) {
	front := newTerritoryTestFront(2, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 100}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 0, 0

	next, _, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "reinforce-full", PlayerID: "player-1", TeamID: "team-001", Kind: "reinforce",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "maximum defense") {
		t.Fatalf("expected maximum defense rejection, got %v", err)
	}
	if next.Revision != front.Revision || next.Teams[0].FrontOpenPower != 100 || next.Teams[0].Score != 0 {
		t.Fatalf("rejected reinforce mutated state: %#v", next.Teams[0])
	}
}

func TestApplyTerritoryReinforceCapsDefenseAndDoesNotScore(t *testing.T) {
	front := newTerritoryTestFront(2, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 90}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 0, 0

	next, _, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "reinforce-1", PlayerID: "player-1", TeamID: "team-001", Kind: "reinforce",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply reinforce: %v", err)
	}
	nextMatrix, _ := decodeTerritoryRows(next.Territory)
	if nextMatrix[0][0].Defense != 100 {
		t.Fatalf("expected capped defense, got %#v", nextMatrix[0][0])
	}
	if next.Teams[0].FrontOpenPower != 92 || next.Teams[0].Score != 0 {
		t.Fatalf("unexpected reinforce accounting: %#v", next.Teams[0])
	}
}

func TestTerritoryObserverCannotPlay(t *testing.T) {
	front := newTerritoryTestFront(2, 1)
	response := detailResponse(front, "team-010", nil)
	if response.CanPlay || len(response.AvailableCommands) != 0 {
		t.Fatalf("expected read-only response, got %#v", response)
	}
	x, y := 0, 0
	_, _, err := applyCommandToFront(front, mongomodel.FrontCommand{TeamID: "team-010", Kind: "expand", TargetX: &x, TargetY: &y})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected observer command rejection, got %v", err)
	}
}

func TestLegacyReinforceRejectsFullyDefendedTargetWithoutCost(t *testing.T) {
	front := mongomodel.Front{
		ID: "legacy", Status: mongomodel.FrontStatusOpenPlay, Revision: 1,
		Cells: []mongomodel.FrontCell{
			{ID: "from", OwnerTeamID: "team-a", NeighborIDs: []string{"to"}},
			{ID: "to", OwnerTeamID: "team-a", Control: 100, Defense: 100, NeighborIDs: []string{"from"}},
		},
		Teams: []mongomodel.FrontTeam{{TeamID: "team-a", FrontOpenPower: 100}},
	}
	_, _, err := applyCommandToFront(front, mongomodel.FrontCommand{
		TeamID: "team-a", Kind: "reinforce", FromCellID: "from", ToCellID: "to", CreatedAt: time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "maximum defense") {
		t.Fatalf("expected full defense rejection, got %v", err)
	}
	if front.Teams[0].FrontOpenPower != 100 || front.Teams[0].Score != 0 {
		t.Fatalf("rejected command changed legacy state: %#v", front.Teams[0])
	}
}

func newTerritoryTestFront(width int, height int) mongomodel.Front {
	matrix := makeTerritoryMatrix(width, height)
	for y := range matrix {
		for x := range matrix[y] {
			matrix[y][x].Playable = true
		}
	}
	return mongomodel.Front{
		ID: "territory-test", MapID: "territory-test", MapMode: contentFrontMapModeTerritoryGrid,
		Status: mongomodel.FrontStatusOpenPlay, Revision: 1,
		Territory: &mongomodel.FrontTerritory{Width: width, Height: height, Connectivity: 4, Rows: encodeTerritoryRows(matrix)},
		Teams:     territoryTeams(100),
	}
}

func TestApplyCommandToFrontCapturesNeutralCell(t *testing.T) {
	createdAt := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	front := mongomodel.Front{
		ID:       "front-test",
		MapID:    "front-test",
		Status:   mongomodel.FrontStatusOpenPlay,
		Revision: 1,
		Cells: []mongomodel.FrontCell{
			{
				ID:          "base",
				Terrain:     "base",
				Zone:        "base",
				OwnerTeamID: "team-a",
				Control:     100,
				Defense:     20,
				NeighborIDs: []string{"frontier"},
			},
			{
				ID:          "frontier",
				Terrain:     "neutral",
				Zone:        "frontier",
				Resource:    20,
				NeighborIDs: []string{"base"},
			},
		},
		Teams: []mongomodel.FrontTeam{
			{TeamID: "team-a", Name: "Alpha", FrontOpenPower: 100},
			{TeamID: "team-b", Name: "Beta", FrontOpenPower: 100},
		},
	}
	command := mongomodel.FrontCommand{
		ID:         "command-a",
		PlayerID:   "player-a",
		TeamID:     "team-a",
		Kind:       "expand",
		FromCellID: "base",
		ToCellID:   "frontier",
		SitoneID:   "stone_engineering_base",
		CreatedAt:  createdAt,
	}

	next, appliedCommand, err := applyCommandToFront(front, command)
	if err != nil {
		t.Fatalf("apply command: %v", err)
	}

	if !appliedCommand.Accepted || !appliedCommand.Applied {
		t.Fatalf("expected accepted command, got %#v", appliedCommand)
	}
	if next.Cells[1].OwnerTeamID != "team-a" || next.Cells[1].Resource != 0 {
		t.Fatalf("expected captured frontier with collected resource, got %#v", next.Cells[1])
	}
	if next.Teams[0].FrontOpenPower != 110 {
		t.Fatalf("expected open power 110 after cost and resource, got %d", next.Teams[0].FrontOpenPower)
	}
	if next.Tick != 1 || next.Revision != 2 || !next.UpdatedAt.Equal(createdAt) {
		t.Fatalf("expected tick/revision/update time to advance, got tick=%d revision=%d updated=%s", next.Tick, next.Revision, next.UpdatedAt)
	}
	if len(next.Leaderboard) == 0 || next.Leaderboard[0].TeamID != "team-a" {
		t.Fatalf("expected team-a to lead after capture, got %#v", next.Leaderboard)
	}
}

func TestWithCurrentPlayerTeamDoesNotRefillExistingZeroOpenPower(t *testing.T) {
	front := mongomodel.Front{
		ID:     "front-test",
		Status: mongomodel.FrontStatusOpenPlay,
		Cells: []mongomodel.FrontCell{
			{ID: "base", Terrain: "base", Zone: "base", OwnerTeamID: "team-a", Control: 100},
		},
		Teams: []mongomodel.FrontTeam{
			{TeamID: "team-a", Name: "Alpha", FrontOpenPower: 0},
		},
	}

	next := withCurrentPlayerTeam(front, mongomodel.Player{ID: "player-a", TeamID: "team-a"})

	if next.Teams[0].FrontOpenPower != 0 {
		t.Fatalf("expected existing zero open power to stay zero, got %d", next.Teams[0].FrontOpenPower)
	}
}
