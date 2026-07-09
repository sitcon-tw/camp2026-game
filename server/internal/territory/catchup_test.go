package territory

import "testing"

func TestAttackCatchUpMultiplier(t *testing.T) {
	cases := []struct {
		name     string
		attacker Tier
		defender Tier
		want     float64
	}{
		{"challenger attacks leader", TierChallenger, TierLeader, 0.50},
		{"challenger attacks contested", TierChallenger, TierContested, 0.65},
		{"contested attacks leader", TierContested, TierLeader, 0.85},
		{"contested attacks contested", TierContested, TierContested, 1},
	}

	for _, testCase := range cases {
		if got := AttackCatchUpMultiplier(testCase.attacker, testCase.defender); got != testCase.want {
			t.Errorf("%s: multiplier = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestCatchUpStealMaxCount(t *testing.T) {
	cases := []struct {
		name     string
		attacker Tier
		defender Tier
		want     int
	}{
		{"challenger attacks leader", TierChallenger, TierLeader, 6},
		{"challenger attacks contested", TierChallenger, TierContested, 5},
		{"contested attacks leader", TierContested, TierLeader, 4},
		{"normal matchup", TierContested, TierContested, 3},
	}

	for _, testCase := range cases {
		if got := CatchUpStealMaxCount(3, testCase.attacker, testCase.defender); got != testCase.want {
			t.Errorf("%s: steal max = %d, want %d", testCase.name, got, testCase.want)
		}
	}
}
