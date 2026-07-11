package fronts

import (
	"strings"
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

func TestAttackDisplacesGarrisonSitones(t *testing.T) {
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
	if len(next.Garrisons) != 0 || len(command.DisplacedGarrisons) != 1 {
		t.Fatalf("displaced garrison was not removed: front=%#v command=%#v", next.Garrisons, command)
	}
	if displaced := command.DisplacedGarrisons[0]; displaced.PlayerID != "defender" || len(displaced.SitoneIDs) != 3 {
		t.Fatalf("displaced sitones missing their original owner: %#v", displaced)
	}
}

func TestGarrisonAttackOpenPowerCostScalesPerSitone(t *testing.T) {
	tests := []struct {
		sitoneCount int
		want        int
	}{
		{sitoneCount: 0, want: 15},
		{sitoneCount: 1, want: 17},
		{sitoneCount: 2, want: 18},
		{sitoneCount: 3, want: 20},
		{sitoneCount: 4, want: 21},
		{sitoneCount: 5, want: 23},
	}
	for _, test := range tests {
		if got := frontAttackOpenPowerCost(1, test.sitoneCount); got != test.want {
			t.Fatalf("%d sitones: got cost %d, want %d", test.sitoneCount, got, test.want)
		}
	}
}

func TestTerritoryAttackCostUsesOnlyTargetGarrison(t *testing.T) {
	front := newTerritoryTestFront(3, 1)
	now := time.Now().UTC()
	target := territoryCell{OccupiedAt: now}
	front.Garrisons = []mongomodel.FrontGarrison{
		{X: 1, Y: 0, SitoneIDs: []string{"stone-a", "stone-b"}},
		{X: 2, Y: 0, SitoneIDs: []string{"stone-a", "stone-b", "stone-c", "stone-d", "stone-e"}},
	}
	if got := territoryAttackOpenPowerCost(front, target, 1, 0, now); got != 18 {
		t.Fatalf("target garrison cost = %d, want 18", got)
	}
	if got := territoryAttackOpenPowerCost(front, target, 0, 0, now); got != 15 {
		t.Fatalf("empty target cost = %d, want 15", got)
	}
}

func TestTerritoryLevelIncreasesAttackOpenPowerCost(t *testing.T) {
	wants := []int{15, 18, 21, 24}
	for index, want := range wants {
		level := index + 1
		if got := frontAttackOpenPowerCost(level, 0); got != want {
			t.Fatalf("level %d cost = %d, want %d", level, got, want)
		}
	}
	if got := frontAttackOpenPowerCost(4, 5); got != 32 {
		t.Fatalf("level 4 with five sitones cost = %d, want 32", got)
	}
}

func TestTerritoryAttackRejectsBalanceBelowGarrisonCost(t *testing.T) {
	front := newTerritoryTestFront(2, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	matrix[0][1] = territoryCell{Playable: true, Owner: "team-002", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Garrisons = []mongomodel.FrontGarrison{{
		X: 1, Y: 0, SitoneIDs: []string{"stone-a", "stone-b", "stone-c", "stone-d", "stone-e"},
	}}
	x, y := 1, 0
	_, _, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "attack-garrison", TeamID: "team-001", Kind: "attack", TargetX: &x, TargetY: &y,
	}, 22)
	if err == nil || !strings.Contains(err.Error(), "insufficient open power") {
		t.Fatalf("expected 23-point garrison attack to reject 22 balance, got %v", err)
	}
}

func TestTerritoryCaptureOpenPowerRewardIsDeterministicAndBounded(t *testing.T) {
	captured := make([]mongomodel.FrontCoordinate, 20)
	for index := range captured {
		captured[index] = mongomodel.FrontCoordinate{X: index, Y: index % 3}
	}
	first := territoryCaptureOpenPowerReward("front-1", "command-1", captured)
	second := territoryCaptureOpenPowerReward("front-1", "command-1", captured)
	if first != second {
		t.Fatalf("reward changed across identical rolls: %d != %d", first, second)
	}
	if first < 1 || first > frontCaptureRewardMaximum {
		t.Fatalf("reward %d is outside 1..%d", first, frontCaptureRewardMaximum)
	}
	if got := territoryCaptureOpenPowerReward("front-1", "command-1", nil); got != 0 {
		t.Fatalf("empty capture reward = %d, want 0", got)
	}
}
