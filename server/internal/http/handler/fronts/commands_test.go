package fronts

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

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
