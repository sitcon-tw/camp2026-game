package territory

// AttackCatchUpMultiplier returns the hidden attack cost multiplier for a
// tier matchup. It gives lower-ranked teams a real path to recover without
// exposing the formula to clients.
func AttackCatchUpMultiplier(attackerTier Tier, defenderTier Tier) float64 {
	switch {
	case attackerTier == TierChallenger && defenderTier == TierLeader:
		return 0.50
	case attackerTier == TierChallenger && defenderTier == TierContested:
		return 0.65
	case attackerTier == TierContested && defenderTier == TierLeader:
		return 0.85
	default:
		return 1
	}
}

// CatchUpStealMaxCount increases the maximum number of stolen stones for
// successful attacks made from below. The base cap still applies to normal
// matchups.
func CatchUpStealMaxCount(base int, attackerTier Tier, defenderTier Tier) int {
	switch {
	case attackerTier == TierChallenger && defenderTier == TierLeader:
		return maxInt(base, 6)
	case attackerTier == TierChallenger && defenderTier == TierContested:
		return maxInt(base, 5)
	case attackerTier == TierContested && defenderTier == TierLeader:
		return maxInt(base, 4)
	default:
		return base
	}
}

func AttackCostExtras(attackerTier Tier, defenderTier Tier) CostExtras {
	return CostExtras{
		AttackCatchUpMultiplier: AttackCatchUpMultiplier(attackerTier, defenderTier),
	}
}
