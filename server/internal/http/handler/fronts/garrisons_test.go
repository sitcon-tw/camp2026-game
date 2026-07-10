package fronts

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestStationAndWithdrawGarrison(t *testing.T) {
	front := newTerritoryTestFront(3, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][1] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 1, 0
	now := time.Now().UTC()

	stationed, command, err := applyTestCommandToFront(front, mongomodel.FrontCommand{
		ID: "station-1", PlayerID: "player-1", TeamID: "team-001", Kind: "station",
		TargetX: &x, TargetY: &y, SitoneIDs: []string{"stone-a", "stone-b"},
		SitoneEffect: mongomodel.FrontSitoneEffect{TotalBonusPercent: 20}, CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("station garrison: %v", err)
	}
	if len(stationed.Garrisons) != 1 || command.GarrisonID == "" || stationed.Garrisons[0].ID != command.GarrisonID {
		t.Fatalf("garrison was not created: front=%#v command=%#v", stationed.Garrisons, command)
	}
	stationedMatrix, _ := decodeTerritoryRows(stationed.Territory)
	if command.SitoneEffect.DefenseBonus != 12 || stationedMatrix[y][x].Defense != 32 {
		t.Fatalf("station defense bonus not applied: defense=%d command=%#v", stationedMatrix[y][x].Defense, command)
	}

	withdrawn, withdraw, err := applyTestCommandToFront(stationed, mongomodel.FrontCommand{
		ID: "withdraw-1", PlayerID: "player-1", TeamID: "team-001", Kind: "withdraw",
		TargetX: &x, TargetY: &y, CreatedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("withdraw garrison: %v", err)
	}
	if len(withdrawn.Garrisons) != 0 || len(withdraw.SitoneIDs) != 2 || withdraw.GarrisonID != command.GarrisonID {
		t.Fatalf("garrison was not returned: front=%#v command=%#v", withdrawn.Garrisons, withdraw)
	}
	withdrawnMatrix, _ := decodeTerritoryRows(withdrawn.Territory)
	if withdrawnMatrix[y][x].Defense != 20 {
		t.Fatalf("withdraw must remove the stationed defense, got %d", withdrawnMatrix[y][x].Defense)
	}
}

func TestAttackCapturesGarrisonSitones(t *testing.T) {
	front := newTerritoryTestFront(3, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	matrix[0][1] = territoryCell{Playable: true, Owner: "team-002", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Garrisons = []mongomodel.FrontGarrison{{
		ID: "garrison-2", PlayerID: "defender", TeamID: "team-002", X: 1, Y: 0,
		SitoneIDs: []string{"stone-a", "stone-a", "stone-b"}, StationedAt: time.Now().UTC(),
	}}
	x, y := 1, 0

	next, command, err := applyTestCommandToFront(front, mongomodel.FrontCommand{
		ID: "attack-garrison", PlayerID: "attacker", TeamID: "team-001", Kind: "attack",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("attack garrison: %v", err)
	}
	if len(next.Garrisons) != 0 || len(command.CapturedGarrisons) != 1 {
		t.Fatalf("captured garrison was not removed: front=%#v command=%#v", next.Garrisons, command)
	}
	if got := command.CapturedGarrisons[0].SitoneIDs; len(got) != 3 {
		t.Fatalf("captured sitones missing: %#v", got)
	}
}
