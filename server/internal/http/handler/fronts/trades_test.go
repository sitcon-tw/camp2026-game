package fronts

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestBuildFrontRailSegmentsConnectsTwoNearestWithinRange(t *testing.T) {
	garrisons := []mongomodel.FrontGarrison{
		{ID: "a", X: 0, Y: 0},
		{ID: "b", X: 5, Y: 0},
		{ID: "c", X: 15, Y: 0},
		{ID: "d", X: 40, Y: 0},
	}
	segments := buildFrontRailSegments(garrisons)
	if len(segments) != 3 {
		t.Fatalf("expected three deduplicated nearby rails, got %#v", segments)
	}
	got := make(map[string]bool, len(segments))
	for _, segment := range segments {
		if segment.Distance > frontRailMaxDistance {
			t.Fatalf("rail exceeded maximum distance: %#v", segment)
		}
		got[frontRailPairKey(segment.SourceGarrisonID, segment.TargetGarrisonID)] = true
	}
	for _, pair := range [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}} {
		if !got[frontRailPairKey(pair[0], pair[1])] {
			t.Fatalf("missing rail %s-%s: %#v", pair[0], pair[1], segments)
		}
	}
}

func TestFrontRailShortestPathUsesConnectedStations(t *testing.T) {
	segments := []mongomodel.FrontRailSegment{
		{SourceGarrisonID: "a", TargetGarrisonID: "b", Distance: 12},
		{SourceGarrisonID: "b", TargetGarrisonID: "c", Distance: 12},
		{SourceGarrisonID: "a", TargetGarrisonID: "d", Distance: 10},
		{SourceGarrisonID: "d", TargetGarrisonID: "c", Distance: 20},
	}
	path := frontRailShortestPath(segments, "a", "c")
	if len(path) != 3 || path[0] != "a" || path[1] != "b" || path[2] != "c" {
		t.Fatalf("unexpected shortest path: %#v", path)
	}
}

func TestAdvanceFrontTradeStateStartsTrainAfterCooldown(t *testing.T) {
	front := newTerritoryTestFront(25, 1)
	base := time.Date(2026, 7, 10, 4, 5, 0, 0, time.UTC)
	front.Garrisons = []mongomodel.FrontGarrison{
		{ID: "a", PlayerID: "player-a", TeamID: "team-001", X: 0, Y: 0, StationedAt: base},
		{ID: "b", PlayerID: "player-b", TeamID: "team-001", X: 12, Y: 0, StationedAt: base, LastTrainAt: base.Add(time.Hour)},
		{ID: "c", PlayerID: "player-c", TeamID: "team-002", X: 24, Y: 0, StationedAt: base, LastTrainAt: base.Add(time.Hour)},
	}
	now := firstFrontTrainDepartureTime(front.ID, "a", base.Add(frontTrainCooldown))

	next, settlements, advance := advanceFrontTradeState(front, now)
	if !advance.Changed || advance.Started != 1 || len(settlements) != 0 {
		t.Fatalf("train was not started: advance=%#v settlements=%#v", advance, settlements)
	}
	if len(next.RailSegments) != 2 || len(next.TradeRoutes) != 1 {
		t.Fatalf("rail network or train missing: rails=%#v trains=%#v", next.RailSegments, next.TradeRoutes)
	}
	route := next.TradeRoutes[0]
	if len(route.Waypoints) != 3 || route.Waypoints[1].GarrisonID != "b" || route.Waypoints[2].GarrisonID != "c" {
		t.Fatalf("train did not use the connected intermediate station: %#v", route.Waypoints)
	}
	if !next.Garrisons[0].LastTrainAt.Equal(now) {
		t.Fatalf("departure cooldown was not persisted: %#v", next.Garrisons[0])
	}
}

func TestAdvanceFrontTradeStateSettlesEveryForeignStop(t *testing.T) {
	front := newTerritoryTestFront(25, 1)
	now := time.Date(2026, 7, 10, 4, 5, 0, 0, time.UTC)
	front.Garrisons = []mongomodel.FrontGarrison{
		{ID: "a", PlayerID: "player-a", TeamID: "team-001", X: 0, Y: 0, LastTrainAt: now},
		{ID: "b", PlayerID: "player-b", TeamID: "team-002", X: 12, Y: 0, LastTrainAt: now},
		{ID: "c", PlayerID: "player-c", TeamID: "team-003", X: 24, Y: 0, LastTrainAt: now},
	}
	front.RailSegments = buildFrontRailSegments(front.Garrisons)
	route, ok := newFrontTrainRoute(front, front.Garrisons[0], front.Garrisons[2], []string{"a", "b", "c"}, now.Add(-2*time.Minute))
	if !ok {
		t.Fatal("failed to create test train")
	}
	front.TradeRoutes = []mongomodel.FrontTradeRoute{route}

	next, settlements, advance := advanceFrontTradeState(front, now)
	if !advance.Changed || advance.Arrived != 2 || len(settlements) != 2 {
		t.Fatalf("foreign stops were not settled: advance=%#v settlements=%#v", advance, settlements)
	}
	if next.TradeRoutes[0].Status != frontTradeRouteCompleted || next.Teams[0].TradeHourlyEarned <= 0 || next.Teams[1].TradeHourlyEarned <= 0 || next.Teams[2].TradeHourlyEarned <= 0 {
		t.Fatalf("unexpected completed train state: route=%#v teams=%#v", next.TradeRoutes[0], next.Teams[:3])
	}
}

func TestAdvanceFrontTradeStateCapsHourlyRewards(t *testing.T) {
	now := time.Date(2026, 7, 10, 4, 5, 0, 0, time.UTC)
	front := newTerritoryTestFront(10, 1)
	front.Teams[0].TradeHourlyLimit = 10
	front.Teams[0].TradeHourlyEarned = 8
	front.Teams[0].TradeWindowStartedAt = now.Truncate(time.Hour)
	front.Teams[1].TradeHourlyLimit = 10
	front.Teams[1].TradeWindowStartedAt = now.Truncate(time.Hour)
	front.Garrisons = []mongomodel.FrontGarrison{
		{ID: "a", PlayerID: "player-a", TeamID: "team-001", X: 0, Y: 0, LastTrainAt: now},
		{ID: "b", PlayerID: "player-b", TeamID: "team-002", X: 9, Y: 0, LastTrainAt: now},
	}
	front.RailSegments = buildFrontRailSegments(front.Garrisons)
	front.TradeRoutes = []mongomodel.FrontTradeRoute{{
		ID: "train-1", SourceGarrisonID: "a", TargetGarrisonID: "b",
		SourcePlayerID: "player-a", TargetPlayerID: "player-b",
		SourceTeamID: "team-001", TargetTeamID: "team-002",
		PotentialReward: 8, Status: frontTradeRouteActive,
		StartedAt: now.Add(-time.Minute), ArrivesAt: now.Add(-time.Second), NextStopIndex: 1,
		Waypoints: []mongomodel.FrontTradeWaypoint{
			{GarrisonID: "a", TeamID: "team-001", X: 0, Y: 0, ArrivesAt: now.Add(-time.Minute)},
			{GarrisonID: "b", TeamID: "team-002", X: 9, Y: 0, ArrivesAt: now.Add(-time.Second), PotentialReward: 8},
		},
	}}

	next, settlements, advance := advanceFrontTradeState(front, now)
	if !advance.Changed || len(settlements) != 1 || settlements[0].SourceReward != 2 {
		t.Fatalf("remaining hourly reward was not applied: %#v", settlements)
	}
	if next.Teams[0].TradeHourlyEarned != 10 || next.Teams[1].TradeHourlyEarned != 8 {
		t.Fatalf("unexpected hourly totals: %#v", next.Teams[:2])
	}
}

func firstFrontTrainDepartureTime(frontID string, garrisonID string, start time.Time) time.Time {
	for now := start; ; now = now.Add(time.Second) {
		if frontTrainShouldDepart(frontID, garrisonID, now) {
			return now
		}
	}
}
