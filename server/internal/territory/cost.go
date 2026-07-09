package territory

// Open power cost module (design §5, decision 10). Costs are keyed on the
// beneficiary's *rank position* on the personal leaderboard, split into
// rank bands. All coefficients live in CostConfig so admins can retune
// them later; none of this (bands, multipliers, formulas) is exposed to
// the frontend — clients only ever receive the (min, max) estimate range.

import (
	"fmt"
	"math"
)

// CostKind identifies the action being paid for.
type CostKind string

const (
	CostKindAttack  CostKind = "attack"
	CostKindRescue  CostKind = "rescue"
	CostKindRedeem  CostKind = "redeem"
	CostKindConvert CostKind = "convert"
	CostKindBoss    CostKind = "boss"
)

// RankBand maps a personal-leaderboard rank segment to a cost multiplier.
// A band covers ranks up to and including MaxRank; the last band should
// use MaxRank 0 to mean "everything beyond".
type RankBand struct {
	MaxRank    int
	Multiplier float64
}

// CostConfig holds every tunable coefficient of the cost formula.
type CostConfig struct {
	BaseCosts map[CostKind]int
	RankBands []RankBand
	// RepeatScale is the per-previous-action surcharge applied to team
	// redeem/convert counts (+25% each by default).
	RepeatScale float64
	// JitterFraction is the ±% wobble applied to the final rolled cost.
	JitterFraction float64
}

// CostExtras carries action-specific inputs to the cost formula.
type CostExtras struct {
	// TeamRedeemCount is how many redeems the beneficiary's team has
	// already performed (scales redeem cost).
	TeamRedeemCount int
	// TeamConvertCount is how many converts the beneficiary's team has
	// already performed (scales convert cost).
	TeamConvertCount int
}

// DefaultCostConfig returns the baseline coefficients: base costs of
// attack 150 / rescue 80 / redeem 120 / convert 130 / boss 200, rank bands
// 1-10 ×1.6, 11-25 ×1.3, 26-45 ×1.0, 46+ ×0.8, +25% per repeated team
// redeem/convert and ±15% roll jitter.
func DefaultCostConfig() CostConfig {
	return CostConfig{
		BaseCosts: map[CostKind]int{
			CostKindAttack:  150,
			CostKindRescue:  80,
			CostKindRedeem:  120,
			CostKindConvert: 130,
			CostKindBoss:    200,
		},
		RankBands: []RankBand{
			{MaxRank: 10, Multiplier: 1.6},
			{MaxRank: 25, Multiplier: 1.3},
			{MaxRank: 45, Multiplier: 1.0},
			{MaxRank: 0, Multiplier: 0.8},
		},
		RepeatScale:    0.25,
		JitterFraction: 0.15,
	}
}

// EstimateCost returns the (min, max) open power cost range for an action
// whose beneficiary sits at the given personal-leaderboard rank position.
// teamSize is reserved for future size-based corrections and currently
// unused. Only this range may be shown to players.
func (c CostConfig) EstimateCost(kind CostKind, beneficiaryRank int, teamSize int, extras CostExtras) (int, int, error) {
	_ = teamSize
	expected, err := c.expectedCost(kind, beneficiaryRank, extras)
	if err != nil {
		return 0, 0, err
	}
	minCost := roundCost(expected * (1 - c.JitterFraction))
	maxCost := roundCost(expected * (1 + c.JitterFraction))
	return minCost, maxCost, nil
}

// Roll produces the final cost using the mission's deterministic roll
// sequence, applying ±JitterFraction around the expected cost. Staff can
// replay the roll from the stored seed ref to audit the charged amount.
func (c CostConfig) Roll(kind CostKind, beneficiaryRank int, teamSize int, extras CostExtras, roll *Roll) (int, error) {
	_ = teamSize
	expected, err := c.expectedCost(kind, beneficiaryRank, extras)
	if err != nil {
		return 0, err
	}
	return roundCost(expected * roll.Jitter(c.JitterFraction)), nil
}

func (c CostConfig) expectedCost(kind CostKind, beneficiaryRank int, extras CostExtras) (float64, error) {
	base, ok := c.BaseCosts[kind]
	if !ok {
		return 0, fmt.Errorf("unsupported cost kind %q", kind)
	}

	expected := float64(base)
	switch kind {
	case CostKindRedeem:
		expected *= 1 + c.RepeatScale*float64(maxInt(extras.TeamRedeemCount, 0))
	case CostKindConvert:
		expected *= 1 + c.RepeatScale*float64(maxInt(extras.TeamConvertCount, 0))
	}
	return expected * c.rankMultiplier(beneficiaryRank), nil
}

func (c CostConfig) rankMultiplier(rank int) float64 {
	if rank < 1 {
		rank = 1
	}
	for _, band := range c.RankBands {
		if band.MaxRank <= 0 || rank <= band.MaxRank {
			return band.Multiplier
		}
	}
	return 1
}

func roundCost(value float64) int {
	if value < 0 {
		return 0
	}
	return int(math.Round(value))
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
