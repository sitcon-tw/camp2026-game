package territory

import "testing"

func TestTierForRank(t *testing.T) {
	cases := []struct {
		rank int
		want Tier
	}{
		{1, TierLeader},
		{2, TierLeader},
		{3, TierLeader},
		{4, TierContested},
		{5, TierContested},
		{6, TierContested},
		{7, TierChallenger},
		{8, TierChallenger},
		{9, TierChallenger},
	}
	for _, testCase := range cases {
		if got := TierForRank(testCase.rank); got != testCase.want {
			t.Errorf("TierForRank(%d) = %v, want %v", testCase.rank, got, testCase.want)
		}
	}
}

func TestTierPermissions(t *testing.T) {
	if CanAttack(TierLeader) {
		t.Error("leader tier must not be able to attack")
	}
	if !CanBeAttacked(TierLeader) {
		t.Error("leader tier must be attackable")
	}
	if !CanAttack(TierContested) || !CanBeAttacked(TierContested) {
		t.Error("contested tier must be able to attack and be attacked")
	}
	if !CanAttack(TierChallenger) {
		t.Error("challenger tier must be able to attack")
	}
	if CanBeAttacked(TierChallenger) {
		t.Error("challenger tier must not be attackable")
	}
}

func TestBuildStandingsOrdersAndAssignsTiers(t *testing.T) {
	config := DefaultCombatRankConfig()
	scores := make([]TeamScore, 0, 9)
	for i := 0; i < 9; i++ {
		scores = append(scores, TeamScore{
			TeamID:      teamID(i),
			PlayerCount: 5,
			SitonePower: (9 - i) * 100,
		})
	}

	standings := BuildStandings(scores, config)
	if len(standings) != 9 {
		t.Fatalf("expected 9 standings, got %d", len(standings))
	}
	for i, standing := range standings {
		if standing.TeamID != teamID(i) {
			t.Errorf("rank %d: expected %s, got %s", i+1, teamID(i), standing.TeamID)
		}
		if standing.Rank != i+1 {
			t.Errorf("expected rank %d, got %d", i+1, standing.Rank)
		}
		if want := TierForRank(i + 1); standing.Tier != want {
			t.Errorf("rank %d: expected tier %v, got %v", i+1, want, standing.Tier)
		}
	}
}

func TestBuildStandingsExcludesNothingButIsDeterministicWithSeed(t *testing.T) {
	config := DefaultCombatRankConfig()
	config.SeedRef = "seed_test"
	scores := []TeamScore{
		{TeamID: "team_a", PlayerCount: 5, SitonePower: 500},
		{TeamID: "team_b", PlayerCount: 6, SitonePower: 500},
		{TeamID: "team_c", PlayerCount: 5, SitonePower: 300, OpenPower: 100},
	}

	first := BuildStandings(scores, config)
	second := BuildStandings(scores, config)
	for i := range first {
		if first[i].TeamID != second[i].TeamID || first[i].Score != second[i].Score {
			t.Fatalf("standings not deterministic for same seed: %+v vs %+v", first[i], second[i])
		}
	}
}

func TestBuildStandingsAppliesBalanceCorrection(t *testing.T) {
	config := DefaultCombatRankConfig()
	config.BalanceCorrection = func(id string, score float64) float64 {
		if id == "team_underdog" {
			return score * 10
		}
		return score
	}
	scores := []TeamScore{
		{TeamID: "team_strong", PlayerCount: 5, SitonePower: 500},
		{TeamID: "team_underdog", PlayerCount: 5, SitonePower: 100},
	}

	standings := BuildStandings(scores, config)
	if standings[0].TeamID != "team_underdog" {
		t.Errorf("balance correction not applied: got %s first", standings[0].TeamID)
	}
}

func teamID(index int) string {
	return string(rune('a'+index)) + "_team"
}
