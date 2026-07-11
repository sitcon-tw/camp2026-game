package fronts

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestAdvanceFrontHoldingRewardsEveryThirtyMinutes(t *testing.T) {
	occupiedAt := time.Date(2026, time.July, 11, 4, 0, 0, 0, time.UTC)
	front := newTerritoryTestFront(1, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20, OccupiedAt: occupiedAt}
	front.Territory.Rows = encodeTerritoryRows(matrix)

	unchanged, settlements, changed := advanceFrontHoldingRewardState(front, occupiedAt.Add(29*time.Minute))
	if changed || len(settlements) != 0 || unchanged.Territory.Rows[0].Runs[0].HoldingRewardPeriods != 0 {
		t.Fatalf("reward settled before 30 minutes: changed=%v settlements=%#v", changed, settlements)
	}

	next, settlements, changed := advanceFrontHoldingRewardState(front, occupiedAt.Add(30*time.Minute))
	if !changed || len(settlements) != 1 || settlements[0].Period != 1 {
		t.Fatalf("first reward was not settled: changed=%v settlements=%#v", changed, settlements)
	}
	if settlements[0].Amount < 2 || settlements[0].Amount > 4 {
		t.Fatalf("level 1 reward %d is outside expected variance", settlements[0].Amount)
	}
	if next.Territory.Rows[0].Runs[0].HoldingRewardPeriods != 1 {
		t.Fatalf("settled period was not persisted: %#v", next.Territory.Rows[0].Runs[0])
	}
}

func TestHoldingRewardLevelProgressionAndVariance(t *testing.T) {
	occupiedAt := time.Date(2026, time.July, 11, 4, 0, 0, 0, time.UTC)
	front := newTerritoryTestFront(1, 1)
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20, OccupiedAt: occupiedAt}
	front.Territory.Rows = encodeTerritoryRows(matrix)

	_, settlements, changed := advanceFrontHoldingRewardState(front, occupiedAt.Add(270*time.Minute))
	if !changed || len(settlements) != 9 {
		t.Fatalf("expected nine catch-up settlements, got changed=%v settlements=%d", changed, len(settlements))
	}
	wants := []struct {
		period int
		level  int
		min    int
		max    int
	}{
		{period: 1, level: 1, min: 2, max: 4},
		{period: 3, level: 2, min: 5, max: 9},
		{period: 6, level: 3, min: 8, max: 14},
		{period: 9, level: 4, min: 11, max: 20},
	}
	for _, want := range wants {
		settlement := settlements[want.period-1]
		if got := territoryHoldingLevel(occupiedAt, settlement.SettledAt); got != want.level {
			t.Fatalf("period %d level = %d, want %d", want.period, got, want.level)
		}
		if settlement.Amount < want.min || settlement.Amount > want.max {
			t.Fatalf("period %d reward %d outside [%d,%d]", want.period, settlement.Amount, want.min, want.max)
		}
		if again := frontHoldingRewardAmount(front.ID, occupiedAt, 0, 0, want.period); again != settlement.Amount {
			t.Fatalf("period %d reward changed from %d to %d", want.period, settlement.Amount, again)
		}
	}
}

func TestHoldingRewardsExcludePermanentBasesAndLegacyTerritory(t *testing.T) {
	now := time.Date(2026, time.July, 11, 8, 0, 0, 0, time.UTC)
	front := newTerritoryTestFront(2, 1)
	front.Territory.Bases = []mongomodel.FrontTerritoryBase{{TeamID: "team-001", X: 0, Y: 0}}
	matrix, _ := decodeTerritoryRows(front.Territory)
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 100, OccupiedAt: now.Add(-time.Hour)}
	matrix[0][1] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)

	_, settlements, changed := advanceFrontHoldingRewardState(front, now)
	if changed || len(settlements) != 0 {
		t.Fatalf("base or legacy territory generated rewards: changed=%v settlements=%#v", changed, settlements)
	}
}
