package territory

// Attack/defense normalization (design §6, plan B). Percentages are
// computed against a full-strength standard derived from team size, so a
// 5-member team is not automatically disadvantaged against a 6-member one.

import "github.com/sitcon-tw/camp2026-game/internal/content"

// averageStatBaseline is the assumed average attack/defense value of a
// deployed sitone used to derive the full-strength standard.
const averageStatBaseline = 15

// FullStrengthStandard returns the standard total attack/defense value a
// team of the given size is normalized against: deploy cap × average stat
// baseline (5 members → 10×15=150, 6 members → 12×15=180).
func FullStrengthStandard(teamSize int) int {
	return DeployCap(teamSize) * averageStatBaseline
}

// ComputeAttackPercent returns A: the deployed sitones' total attack value
// divided by the team's full-strength standard.
func ComputeAttackPercent(sitones []content.Sitone, teamSize int) float64 {
	total := 0
	for _, sitone := range sitones {
		total += sitone.Attack
	}
	return normalizePercent(total, teamSize)
}

// ComputeDefensePercent returns D: the defending sitones' total defense
// value divided by the team's full-strength standard.
func ComputeDefensePercent(sitones []content.Sitone, teamSize int) float64 {
	total := 0
	for _, sitone := range sitones {
		total += sitone.Defense
	}
	return normalizePercent(total, teamSize)
}

func normalizePercent(total int, teamSize int) float64 {
	standard := FullStrengthStandard(teamSize)
	if standard <= 0 {
		return 0
	}
	return float64(total) / float64(standard)
}
