package front

import (
	"strings"
	"testing"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
)

func TestNewStateAssignsTeamBasesAndActiveEvents(t *testing.T) {
	state := newTestState(t)

	if state.MapID != "test_front" || state.Status != StatusOpenPlay || state.TickSeconds != 1 {
		t.Fatalf("unexpected state metadata: %#v", state)
	}
	if len(state.Cells) != 5 || len(state.Teams) != 2 {
		t.Fatalf("unexpected state size: cells=%d teams=%d", len(state.Cells), len(state.Teams))
	}
	if cellByID(t, state, "base_a").OwnerTeamID != "team-a" {
		t.Fatalf("expected base_a to be assigned to team-a")
	}
	if cellByID(t, state, "base_b").OwnerTeamID != "team-b" {
		t.Fatalf("expected base_b to be assigned to team-b")
	}
	if cellByID(t, state, "repair_a").EventID != "evt_repair" || len(state.ActiveEvents) != 1 {
		t.Fatalf("expected repair event to be active, got cells=%#v events=%#v", state.Cells, state.ActiveEvents)
	}

	leaderboard := BuildLeaderboard(state)
	if len(leaderboard) != 2 || leaderboard[0].TeamID != "team-a" || leaderboard[0].ControlledCells != 1 {
		t.Fatalf("unexpected initial leaderboard: %#v", leaderboard)
	}
}

func TestValidateCommandRejectsInvalidCommand(t *testing.T) {
	state := newTestState(t)

	err := ValidateCommand(state, Command{
		ID:         "cmd-1",
		PlayerID:   "player-a",
		TeamID:     "team-a",
		Kind:       CommandExpand,
		FromCellID: "base_a",
		ToCellID:   "base_b",
		SitoneID:   "stone-a",
	})
	if err == nil {
		t.Fatal("expected non-adjacent command to be rejected")
	}
	if !strings.Contains(err.Error(), `cell "base_b" is not adjacent to "base_a"`) {
		t.Fatalf("expected adjacency error, got %v", err)
	}

	state.Teams[0].FrontOpenPower = 0
	err = ValidateCommand(state, Command{
		ID:         "cmd-2",
		PlayerID:   "player-a",
		TeamID:     "team-a",
		Kind:       CommandExpand,
		FromCellID: "base_a",
		ToCellID:   "neutral_a",
		SitoneID:   "stone-a",
	})
	if err == nil || !strings.Contains(err.Error(), "needs 10 front open power") {
		t.Fatalf("expected open power error, got %v", err)
	}
}

func TestApplyCommandCapturesNeutralCellAndCopiesState(t *testing.T) {
	state := newTestState(t)

	next, result, err := ApplyCommand(state, Command{
		ID:         "cmd-1",
		PlayerID:   "player-a",
		TeamID:     "team-a",
		Kind:       CommandExpand,
		FromCellID: "base_a",
		ToCellID:   "neutral_a",
		SitoneID:   "stone-a",
	})
	if err != nil {
		t.Fatalf("apply command: %v", err)
	}
	if !result.Accepted || !result.Applied || !result.CapturedCell || result.ScoreDelta != 30 || result.ResourceDelta != 20 {
		t.Fatalf("unexpected command result: %#v", result)
	}
	if cellByID(t, next, "neutral_a").OwnerTeamID != "team-a" {
		t.Fatalf("expected neutral_a to be captured: %#v", cellByID(t, next, "neutral_a"))
	}
	if cellByID(t, state, "neutral_a").OwnerTeamID != "" {
		t.Fatalf("expected original state to remain unchanged: %#v", cellByID(t, state, "neutral_a"))
	}

	team := teamByID(t, next, "team-a")
	if team.Score != 30 || team.FrontOpenPower != 110 || team.ControlledCells != 2 {
		t.Fatalf("unexpected team state after expand: %#v", team)
	}
}

func TestAdvanceTickProcessesDecayEventsCommandsAndLeaderboard(t *testing.T) {
	state := newTestState(t)
	now := time.Unix(10, 0).UTC()

	next, results := AdvanceTick(state, []Command{
		{
			ID:         "cmd-repair",
			PlayerID:   "player-a",
			TeamID:     "team-a",
			Kind:       CommandRepair,
			FromCellID: "base_a",
			ToCellID:   "repair_a",
			SitoneID:   "stone-a",
		},
		{
			ID:         "cmd-invalid",
			PlayerID:   "player-b",
			TeamID:     "team-b",
			Kind:       CommandAttack,
			FromCellID: "base_b",
			ToCellID:   "neutral_a",
			SitoneID:   "stone-b",
		},
	}, now)

	if next.Tick != 1 || !next.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected tick metadata: tick=%d updated=%s", next.Tick, next.UpdatedAt)
	}
	if len(results) != 2 || !results[0].Applied || results[0].ScoreDelta != 30 {
		t.Fatalf("unexpected repair result: %#v", results)
	}
	if results[1].Accepted || !strings.Contains(results[1].RejectReason, "must be controlled by another team") {
		t.Fatalf("expected invalid attack to be rejected: %#v", results[1])
	}
	if cellByID(t, next, "repair_a").EventID != "" || len(next.ActiveEvents) != 1 {
		t.Fatalf("expected repair event removed and rescue event activated, got cells=%#v events=%#v", next.Cells, next.ActiveEvents)
	}
	if cellByID(t, next, "neutral_a").Resource != 19 {
		t.Fatalf("expected resource decay, got %#v", cellByID(t, next, "neutral_a"))
	}

	leaderboard := BuildLeaderboard(next)
	if leaderboard[0].TeamID != "team-a" || leaderboard[0].Score != 30 || leaderboard[0].RepairedEvents != 1 {
		t.Fatalf("expected team-a to lead after repair, got %#v", leaderboard)
	}
}

func TestAdvanceTickRejectsCooldown(t *testing.T) {
	state := newTestState(t)

	next, results := AdvanceTick(state, []Command{
		{
			ID:         "cmd-1",
			PlayerID:   "player-a",
			TeamID:     "team-a",
			Kind:       CommandScout,
			FromCellID: "base_a",
			ToCellID:   "neutral_a",
			SitoneID:   "stone-a",
		},
		{
			ID:         "cmd-2",
			PlayerID:   "player-a",
			TeamID:     "team-a",
			Kind:       CommandScout,
			FromCellID: "base_a",
			ToCellID:   "neutral_a",
			SitoneID:   "stone-b",
		},
	}, time.Unix(20, 0).UTC())

	if len(results) != 2 || !results[0].Applied {
		t.Fatalf("expected first command to apply, got %#v", results)
	}
	if results[1].Accepted || !strings.Contains(results[1].RejectReason, "command cooldown") {
		t.Fatalf("expected second command to be rejected by cooldown, got %#v", results[1])
	}
	if next.Teams[0].Score != 3 {
		t.Fatalf("expected only first scout to score, got %#v", next.Teams[0])
	}
}

func TestReinforceRejectsMaximumWithoutSpendingOrScoring(t *testing.T) {
	state := newTestState(t)
	cell := cellByID(t, state, "base_a")
	for i := range state.Cells {
		if state.Cells[i].ID == cell.ID {
			state.Cells[i].Control = 100
			state.Cells[i].Defense = 100
		}
		if state.Cells[i].ID == "neutral_a" {
			state.Cells[i].OwnerTeamID = "team-a"
		}
	}

	next, result, err := ApplyCommand(state, Command{
		ID: "cmd-full", PlayerID: "player-a", TeamID: "team-a", Kind: CommandReinforce,
		FromCellID: "neutral_a", ToCellID: "base_a", SitoneID: "stone-a",
	})
	if err == nil || !strings.Contains(err.Error(), "maximum defense") {
		t.Fatalf("expected maximum defense error, got result=%#v err=%v", result, err)
	}
	if teamByID(t, next, "team-a").FrontOpenPower != teamByID(t, state, "team-a").FrontOpenPower || teamByID(t, next, "team-a").Score != teamByID(t, state, "team-a").Score {
		t.Fatalf("rejected reinforce changed team state: before=%#v after=%#v", teamByID(t, state, "team-a"), teamByID(t, next, "team-a"))
	}
}

func TestReinforceAndRepairCapDefenseAtOneHundred(t *testing.T) {
	state := newTestState(t)
	for i := range state.Cells {
		switch state.Cells[i].ID {
		case "base_a", "repair_a":
			state.Cells[i].Defense = 95
		}
		if state.Cells[i].ID == "repair_a" {
			state.Cells[i].OwnerTeamID = "team-a"
		}
	}

	reinforced, _, err := ApplyCommand(state, Command{
		ID: "cmd-reinforce", PlayerID: "player-a", TeamID: "team-a", Kind: CommandReinforce,
		FromCellID: "repair_a", ToCellID: "base_a", SitoneID: "stone-a",
	})
	if err != nil {
		t.Fatalf("reinforce: %v", err)
	}
	if cellByID(t, reinforced, "base_a").Defense != 100 {
		t.Fatalf("reinforce exceeded cap: %#v", cellByID(t, reinforced, "base_a"))
	}

	repaired, _, err := ApplyCommand(state, Command{
		ID: "cmd-repair", PlayerID: "player-b", TeamID: "team-a", Kind: CommandRepair,
		FromCellID: "base_a", ToCellID: "repair_a", SitoneID: "stone-b",
	})
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if cellByID(t, repaired, "repair_a").Defense != 100 {
		t.Fatalf("repair exceeded cap: %#v", cellByID(t, repaired, "repair_a"))
	}
}

func TestReinforceChargesProportionallyForPartialDefense(t *testing.T) {
	state := newTestState(t)
	for i := range state.Cells {
		switch state.Cells[i].ID {
		case "base_a":
			state.Cells[i].Control = 100
			state.Cells[i].Defense = 95
		case "repair_a":
			state.Cells[i].OwnerTeamID = "team-a"
		}
	}

	next, _, err := ApplyCommand(state, Command{
		ID: "cmd-partial-reinforce", PlayerID: "player-a", TeamID: "team-a", Kind: CommandReinforce,
		FromCellID: "repair_a", ToCellID: "base_a", SitoneID: "stone-a",
	})
	if err != nil {
		t.Fatalf("partial reinforce: %v", err)
	}
	if cellByID(t, next, "base_a").Defense != 100 {
		t.Fatalf("partial reinforce did not cap at 100: %#v", cellByID(t, next, "base_a"))
	}
	if teamByID(t, next, "team-a").FrontOpenPower != 98 {
		t.Fatalf("expected ceil(8*5/37)=2 front open power cost, got %#v", teamByID(t, next, "team-a"))
	}
}

func newTestState(t *testing.T) State {
	t.Helper()

	template := content.FrontMapTemplate{
		ID:                    "test_front",
		Name:                  "Test Front",
		Enabled:               true,
		TickSeconds:           1,
		InitialFrontOpenPower: 100,
		CommandCooldownTicks:  2,
		CommandResolveTicks:   1,
		ResourceDecayPerTick:  1,
		Cells: []content.FrontMapCell{
			{
				ID:        "base_a",
				Terrain:   content.FrontTerrainBase,
				Zone:      content.FrontZoneBase,
				Neighbors: []string{"neutral_a", "repair_a"},
			},
			{
				ID:        "base_b",
				X:         2,
				Terrain:   content.FrontTerrainBase,
				Zone:      content.FrontZoneBase,
				Neighbors: []string{"attack_a"},
			},
			{
				ID:              "neutral_a",
				X:               1,
				Terrain:         content.FrontTerrainNeutral,
				Zone:            content.FrontZoneFrontier,
				Neighbors:       []string{"base_a", "attack_a"},
				InitialResource: 20,
			},
			{
				ID:        "attack_a",
				X:         2,
				Y:         1,
				Terrain:   content.FrontTerrainNeutral,
				Zone:      content.FrontZoneFrontier,
				Neighbors: []string{"base_b", "neutral_a"},
			},
			{
				ID:        "repair_a",
				Y:         1,
				Terrain:   content.FrontTerrainSystem,
				Zone:      content.FrontZoneSystem,
				Neighbors: []string{"base_a"},
			},
		},
		Events: []content.FrontMapEventTemplate{
			{
				ID:                "evt_repair",
				CellID:            "repair_a",
				Kind:              content.FrontEventRepair,
				Title:             "Repair",
				StartsAtTick:      0,
				ExpiresAfterTicks: 10,
				Severity:          1,
			},
			{
				ID:                "evt_rescue",
				CellID:            "neutral_a",
				Kind:              content.FrontEventRescue,
				Title:             "Rescue",
				StartsAtTick:      1,
				ExpiresAfterTicks: 10,
				Severity:          1,
			},
		},
	}

	state, err := NewState(template, []TeamSeed{
		{ID: "team-a", Color: "#2f80ed"},
		{ID: "team-b", Color: "#eb5757"},
	})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	return state
}

func cellByID(t *testing.T, state State, id string) Cell {
	t.Helper()

	for _, cell := range state.Cells {
		if cell.ID == id {
			return cell
		}
	}
	t.Fatalf("cell %q not found in %#v", id, state.Cells)
	return Cell{}
}

func teamByID(t *testing.T, state State, id string) TeamState {
	t.Helper()

	for _, team := range state.Teams {
		if team.TeamID == id {
			return team
		}
	}
	t.Fatalf("team %q not found in %#v", id, state.Teams)
	return TeamState{}
}
