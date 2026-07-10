package fronts

import (
	"strings"
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

const testPlayerOpenPower = 1000

func applyTestCommandToFront(front mongomodel.Front, command mongomodel.FrontCommand) (mongomodel.Front, mongomodel.FrontCommand, error) {
	return applyCommandToFront(front, command, testPlayerOpenPower)
}

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
}

func TestWithFrontDefaultsReactivatesConfiguredTerritorySession(t *testing.T) {
	store := testcontent.Load(t)
	handler := New(Dependencies{Content: store})
	front := handler.withFrontDefaults(mongomodel.Front{
		ID: "campus_territory_v1", MapID: "campus_territory_v1",
		Territory: &mongomodel.FrontTerritory{Width: 1, Height: 1, Connectivity: 4},
	})
	if !front.Current || front.MapMode != contentFrontMapModeTerritoryGrid {
		t.Fatal("configured territory session must be active")
	}
}

func TestTerritoryRepairUsesSitoneBonus(t *testing.T) {
	store := testcontent.Load(t)
	template, _ := store.GetTerritoryMap("campus_territory_v1")
	front := frontFromTerritoryTemplate(template)
	landmark, ok := territoryLandmarkAt(front.Territory.Landmarks, 20, 36)
	if !ok || landmark.Kind != "repair" {
		t.Fatalf("repair landmark missing: %#v", landmark)
	}
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[landmark.Y][landmark.X].Owner = "team-001"
	matrix[landmark.Y][landmark.X].Defense = 50
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := landmark.X, landmark.Y
	effect := frontSitoneEffect(store, "repair", []string{"stone_engineering_base", "stone_entertainment_base"})
	next, command, err := applyTestCommandToFront(front, mongomodel.FrontCommand{
		ID: "repair-1", PlayerID: "player-1", TeamID: "team-001", Kind: "repair",
		TargetX: &x, TargetY: &y, SitoneEffect: effect, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("resolve repair: %v", err)
	}
	nextMatrix, _ := decodeTerritoryRows(next.Territory)
	if command.SitoneEffect.DefenseBonus <= 0 || command.SitoneEffect.ScoreBonus <= 0 || nextMatrix[y][x].Defense <= 75 {
		t.Fatalf("expected visible sitone repair bonuses: command=%#v defense=%d", command, nextMatrix[y][x].Defense)
	}
}

func TestApplyTerritoryExpandUsesCoordinateAndConnectedPatch(t *testing.T) {
	front := newTerritoryTestFront(20, 3)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[1][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 19, 1
	effect := mongomodel.FrontSitoneEffect{SelectedCount: 5, SquadBonusPercent: 20, AffinityBonusPercent: 40, TotalBonusPercent: 60}
	next, command, err := applyTestCommandToFront(front, mongomodel.FrontCommand{
		ID: "expand-1", PlayerID: "player-1", TeamID: "team-001", Kind: "expand",
		TargetX: &x, TargetY: &y, SitoneEffect: effect, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply expand: %v", err)
	}
	want := territoryExpandLimit + scaledFrontSitoneBonus(territoryExpandLimit, 60)
	if len(command.AffectedCells) != want || command.SitoneEffect.AffectedCellBonus != want-territoryExpandLimit {
		t.Fatalf("expected %d affected cells with bonus, got %#v", want, command)
	}
	nextMatrix, _ := decodeTerritoryRows(next.Territory)
	for _, coordinate := range command.AffectedCells {
		if nextMatrix[coordinate.Y][coordinate.X].Owner != "team-001" {
			t.Fatalf("affected coordinate was not captured: %#v", coordinate)
		}
	}
}

func TestTerritoryObserverCannotPlay(t *testing.T) {
	front := newTerritoryTestFront(2, 1)
	response := detailResponse(front, "team-010", frontSitoneInventory{}, testPlayerOpenPower)
	if response.CanPlay || len(response.AvailableCommands) != 0 {
		t.Fatalf("expected read-only response, got %#v", response)
	}
	x, y := 0, 0
	_, _, err := applyTestCommandToFront(front, mongomodel.FrontCommand{TeamID: "team-010", Kind: "expand", TargetX: &x, TargetY: &y})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected observer command rejection, got %v", err)
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
		Teams:     territoryTeams(),
	}
}
