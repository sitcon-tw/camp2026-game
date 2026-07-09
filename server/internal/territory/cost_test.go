package territory

import "testing"

func TestEstimateCostRankBands(t *testing.T) {
	config := DefaultCostConfig()
	cases := []struct {
		rank       int
		multiplier float64
	}{
		{1, 1.6},
		{10, 1.6},
		{11, 1.3},
		{25, 1.3},
		{26, 1.0},
		{45, 1.0},
		{46, 0.8},
		{100, 0.8},
	}
	for _, testCase := range cases {
		if got := config.rankMultiplier(testCase.rank); got != testCase.multiplier {
			t.Errorf("rankMultiplier(%d) = %v, want %v", testCase.rank, got, testCase.multiplier)
		}
	}
}

func TestEstimateCostRangeAroundExpected(t *testing.T) {
	config := DefaultCostConfig()

	// attack base 150 at rank 30 (band ×1.0) → expected 150, ±15%.
	minCost, maxCost, err := config.EstimateCost(CostKindAttack, 30, 5, CostExtras{})
	if err != nil {
		t.Fatalf("EstimateCost: %v", err)
	}
	if minCost != 128 || maxCost != 173 {
		t.Errorf("attack rank 30 estimate = (%d, %d), want (128, 173)", minCost, maxCost)
	}

	// attack at rank 5 (band ×1.6) → expected 240.
	minCost, maxCost, err = config.EstimateCost(CostKindAttack, 5, 5, CostExtras{})
	if err != nil {
		t.Fatalf("EstimateCost: %v", err)
	}
	if minCost != 204 || maxCost != 276 {
		t.Errorf("attack rank 5 estimate = (%d, %d), want (204, 276)", minCost, maxCost)
	}
}

func TestEstimateCostAttackCatchUpDiscount(t *testing.T) {
	config := DefaultCostConfig()

	minCost, maxCost, err := config.EstimateCost(
		CostKindAttack,
		30,
		5,
		AttackCostExtras(TierChallenger, TierLeader),
	)
	if err != nil {
		t.Fatalf("EstimateCost: %v", err)
	}
	if minCost != 64 || maxCost != 86 {
		t.Errorf("challenger attack leader estimate = (%d, %d), want (64, 86)", minCost, maxCost)
	}
}

func TestEstimateCostRepeatScaling(t *testing.T) {
	config := DefaultCostConfig()

	// redeem base 120 at rank 30, 2 previous team redeems → 120×1.5=180.
	minCost, maxCost, err := config.EstimateCost(CostKindRedeem, 30, 5, CostExtras{TeamRedeemCount: 2})
	if err != nil {
		t.Fatalf("EstimateCost: %v", err)
	}
	if minCost != 153 || maxCost != 207 {
		t.Errorf("redeem estimate = (%d, %d), want (153, 207)", minCost, maxCost)
	}

	// convert base 130 at rank 30, 1 previous team convert → 130×1.25=162.5.
	minCost, maxCost, err = config.EstimateCost(CostKindConvert, 30, 5, CostExtras{TeamConvertCount: 1})
	if err != nil {
		t.Fatalf("EstimateCost: %v", err)
	}
	if minCost != 138 || maxCost != 187 {
		t.Errorf("convert estimate = (%d, %d), want (138, 187)", minCost, maxCost)
	}
}

func TestEstimateCostUnknownKind(t *testing.T) {
	config := DefaultCostConfig()
	if _, _, err := config.EstimateCost(CostKind("unknown"), 1, 5, CostExtras{}); err == nil {
		t.Error("expected error for unknown cost kind")
	}
}

func TestRollStaysWithinEstimateRangeAndIsReproducible(t *testing.T) {
	config := DefaultCostConfig()
	minCost, maxCost, err := config.EstimateCost(CostKindBoss, 8, 6, CostExtras{})
	if err != nil {
		t.Fatalf("EstimateCost: %v", err)
	}

	seedRef := "seed_cost_test"
	first, err := config.Roll(CostKindBoss, 8, 6, CostExtras{}, NewRoll(seedRef))
	if err != nil {
		t.Fatalf("Roll: %v", err)
	}
	if first < minCost || first > maxCost {
		t.Errorf("rolled cost %d outside estimate range (%d, %d)", first, minCost, maxCost)
	}

	second, err := config.Roll(CostKindBoss, 8, 6, CostExtras{}, NewRoll(seedRef))
	if err != nil {
		t.Fatalf("Roll: %v", err)
	}
	if first != second {
		t.Errorf("roll not reproducible from seed ref: %d vs %d", first, second)
	}
}

func TestFullStrengthStandard(t *testing.T) {
	if got := FullStrengthStandard(5); got != 150 {
		t.Errorf("FullStrengthStandard(5) = %d, want 150", got)
	}
	if got := FullStrengthStandard(6); got != 180 {
		t.Errorf("FullStrengthStandard(6) = %d, want 180", got)
	}
}

func TestVoteThresholdAndDeployCap(t *testing.T) {
	if DeployCap(5) != 10 || DeployCap(6) != 12 {
		t.Error("deploy caps must be 10 for 5 members and 12 for 6 members")
	}
	if VoteThreshold(5) != 3 || VoteThreshold(6) != 4 {
		t.Error("vote thresholds must be 3 for 5 members and 4 for 6 members")
	}
}
