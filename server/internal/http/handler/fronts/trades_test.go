package fronts

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestAdvanceFrontTradeStateStartsAndSettlesRoutes(t *testing.T) {
	front := newTerritoryTestFront(10, 1)
	front.Garrisons = []mongomodel.FrontGarrison{
		{ID: "garrison-a", PlayerID: "player-a", TeamID: "team-001", X: 0, Y: 0, SitoneIDs: []string{"stone-a"}},
		{ID: "garrison-b", PlayerID: "player-b", TeamID: "team-002", X: 9, Y: 0, SitoneIDs: []string{"stone-b", "stone-b"}, TradeBonusPercent: 20},
	}
	now := time.Date(2026, 7, 10, 4, 5, 0, 0, time.UTC)

	startedFront, settlements, started, changed := advanceFrontTradeState(front, now)
	if !changed || started != 2 || len(settlements) != 0 || len(startedFront.TradeRoutes) != 2 {
		t.Fatalf("trade routes not started: changed=%v started=%d settlements=%#v routes=%#v", changed, started, settlements, startedFront.TradeRoutes)
	}
	for _, route := range startedFront.TradeRoutes {
		duration := route.ArrivesAt.Sub(route.StartedAt)
		if duration < frontTradeMinDuration || duration > frontTradeMaxDuration || route.PotentialReward <= 0 {
			t.Fatalf("invalid trade route timing or reward: %#v", route)
		}
	}

	settledFront, settlements, _, changed := advanceFrontTradeState(startedFront, now.Add(frontTradeMaxDuration+time.Second))
	if !changed || len(settlements) != 2 {
		t.Fatalf("trade routes not settled: changed=%v settlements=%#v", changed, settlements)
	}
	if settledFront.Teams[0].TradeHourlyEarned <= 0 || settledFront.Teams[1].TradeHourlyEarned <= 0 {
		t.Fatalf("both teams must earn from completed trade: %#v", settledFront.Teams[:2])
	}
	if settledFront.Teams[0].TradeHourlyEarned > frontTradeHourlyLimit || settledFront.Teams[1].TradeHourlyEarned > frontTradeHourlyLimit {
		t.Fatalf("hourly limit exceeded: %#v", settledFront.Teams[:2])
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
		{ID: "garrison-a", PlayerID: "player-a", TeamID: "team-001", X: 0, Y: 0, SitoneIDs: []string{"stone-a"}},
		{ID: "garrison-b", PlayerID: "player-b", TeamID: "team-002", X: 9, Y: 0, SitoneIDs: []string{"stone-b"}},
	}
	front.TradeRoutes = []mongomodel.FrontTradeRoute{{
		ID: "trade-1", SourceGarrisonID: "garrison-a", TargetGarrisonID: "garrison-b",
		SourcePlayerID: "player-a", TargetPlayerID: "player-b",
		SourceTeamID: "team-001", TargetTeamID: "team-002",
		PotentialReward: 8, Status: frontTradeRouteActive,
		StartedAt: now.Add(-time.Minute), ArrivesAt: now.Add(-time.Second),
	}}

	next, settlements, started, changed := advanceFrontTradeState(front, now)
	if !changed || len(settlements) != 1 || settlements[0].SourceReward != 2 {
		t.Fatalf("remaining hourly reward was not applied: %#v", settlements)
	}
	if next.Teams[0].TradeHourlyEarned != 10 || next.Teams[1].TradeHourlyEarned != 8 {
		t.Fatalf("unexpected hourly totals: %#v", next.Teams[:2])
	}
	if started != 0 {
		t.Fatalf("capped team must not start another route, got %d", started)
	}
}
