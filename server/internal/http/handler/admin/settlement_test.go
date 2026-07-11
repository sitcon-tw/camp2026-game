package admin

import (
	"testing"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestBuildSettlementResponse(t *testing.T) {
	teamA := &DashboardTeamSummaryResponse{TeamID: "a", Name: "A"}
	teamB := &DashboardTeamSummaryResponse{TeamID: "b", Name: "B"}
	players := []DashboardPlayerResponse{
		{PlayerID: "alice", Nickname: "Alice", Team: teamA, SitoneCount: 9, ItemCount: 2, OpenPower: 100, AnswerCount: 10, AnswerAccuracy: 80},
		{PlayerID: "bob", Nickname: "Bob", Team: teamA, SitoneCount: 3, ItemCount: 8, OpenPower: 0, AnswerCount: 5, AnswerAccuracy: 20},
		{PlayerID: "carol", Nickname: "Carol", Team: teamB, SitoneCount: 4, ItemCount: 1, OpenPower: 50},
		{PlayerID: "staff", Nickname: "Staff", Team: teamB, Role: authctx.PlayerRoleStaff, SitoneCount: 99, ItemCount: 99},
		{PlayerID: "worker", Nickname: "Worker", Team: &DashboardTeamSummaryResponse{TeamID: settlementExcludedTeamID, Name: "Team 010"}, SitoneCount: 999, ItemCount: 999},
	}
	completedAt := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	data := settlementData{
		Dashboard: DashboardResponse{
			Players: players,
			Teams:   []DashboardTeamResponse{{TeamID: "a", Name: "A", PlayerCount: 2}, {TeamID: "b", Name: "B", PlayerCount: 2}},
		},
		TShirts: []settlementPlayerQuantityStat{{PlayerID: "alice", Quantity: 2}, {PlayerID: "bob", Quantity: 1}, {PlayerID: "worker", Quantity: 99}},
		Front:   &mongomodel.Front{Teams: []mongomodel.FrontTeam{{TeamID: "a", Name: "A", ControlledCells: 10}, {TeamID: "b", Name: "B", ControlledCells: 20}, {TeamID: settlementExcludedTeamID, Name: "Team 010", ControlledCells: 999}}},
		Matches: []mongomodel.Match{{
			ID: "latest", Mode: mongomodel.MatchModePVP, Status: mongomodel.MatchStatusCompleted, CompletedAt: completedAt,
			Players: []mongomodel.MatchPlayer{{PlayerID: "alice", Kind: mongomodel.MatchPlayerKindHuman}, {PlayerID: "bob", Kind: mongomodel.MatchPlayerKindHuman}},
		}},
	}

	got := buildSettlementResponse(completedAt, data)
	if len(got.TShirts) != 2 || got.TShirts[0].PlayerID != "alice" || got.TShirts[0].Quantity != 2 {
		t.Fatalf("unexpected tshirt holders: %#v", got.TShirts)
	}
	byKey := map[string]SettlementAwardResponse{}
	for _, award := range got.Awards {
		byKey[award.Key] = award
	}
	if got := byKey["most_sitones"].Players[0].PlayerID; got != "alice" {
		t.Fatalf("most sitones = %q", got)
	}
	if got := byKey["most_items"].Players[0].PlayerID; got != "bob" {
		t.Fatalf("most items = %q", got)
	}
	if got := byKey["lowest_accuracy"].Players[0].PlayerID; got != "bob" {
		t.Fatalf("lowest accuracy = %q", got)
	}
	if got := byKey["largest_island"].Team.TeamID; got != "b" {
		t.Fatalf("largest island = %q", got)
	}
	if got := byKey["lowest_team_average_sitones"].Team.TeamID; got != "b" {
		t.Fatalf("lowest team average = %q", got)
	}
	if got := len(byKey["team_first_places"].Players); got != 2 {
		t.Fatalf("team first places = %d", got)
	}
	for _, winner := range byKey["team_first_places"].Players {
		if winner.PlayerID == "staff" {
			t.Fatal("staff included in settlement")
		}
	}
	for _, award := range got.Awards {
		if award.Team != nil && award.Team.TeamID == settlementExcludedTeamID {
			t.Fatalf("team 010 won award %q", award.Key)
		}
		for _, winner := range award.Players {
			if winner.PlayerID == "worker" {
				t.Fatalf("team 010 player won award %q", award.Key)
			}
		}
	}
}

func TestSettlementEmptyData(t *testing.T) {
	got := buildSettlementResponse(time.Time{}, settlementData{})
	if len(got.Awards) != 9 {
		t.Fatalf("awards = %d, want 9", len(got.Awards))
	}
	for _, award := range got.Awards {
		if award.Players == nil {
			t.Fatalf("award %q players must be an empty array", award.Key)
		}
	}
}
