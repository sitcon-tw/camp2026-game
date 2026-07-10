package fronts

import (
	"strings"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	frontSquadBonusPerExtraSitone = 5
	frontAffinityBonusCap         = 40
	frontTotalSitoneBonusCap      = 60
)

func frontAffinityCommands(sitoneType string) []string {
	switch strings.TrimSpace(sitoneType) {
	case "exploration":
		return []string{"expand"}
	case "engineering":
		return []string{"attack", "reinforce", "repair"}
	case "resonance":
		return []string{"rescue", "support"}
	case "inspiration":
		return []string{"answer_challenge"}
	case "entertainment":
		return []string{"repair", "rescue", "support", "answer_challenge"}
	default:
		return []string{}
	}
}

func frontSitoneEffect(store *content.Store, kind string, sitoneIDs []string) mongomodel.FrontSitoneEffect {
	selectedCount := len(sitoneIDs)
	squadBonus := maxFrontInt(0, selectedCount-1) * frontSquadBonusPerExtraSitone
	affinityBonus := 0
	if store != nil {
		for _, sitoneID := range sitoneIDs {
			sitone, ok := store.GetSitone(sitoneID)
			if ok && stringSliceContains(frontAffinityCommands(sitone.Type), kind) {
				affinityBonus += sitone.AbilityValue
			}
		}
	}
	affinityBonus = minFrontInt(affinityBonus, frontAffinityBonusCap)
	return mongomodel.FrontSitoneEffect{
		SelectedCount:        selectedCount,
		SquadBonusPercent:    squadBonus,
		AffinityBonusPercent: affinityBonus,
		TotalBonusPercent:    minFrontInt(squadBonus+affinityBonus, frontTotalSitoneBonusCap),
	}
}

func scaledFrontSitoneBonus(base int, percent int) int {
	if base <= 0 || percent <= 0 {
		return 0
	}
	return (base*percent + 99) / 100
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func frontSitoneEffectResponse(effect mongomodel.FrontSitoneEffect) FrontSitoneEffectResponse {
	return FrontSitoneEffectResponse{
		SelectedCount:        effect.SelectedCount,
		SquadBonusPercent:    effect.SquadBonusPercent,
		AffinityBonusPercent: effect.AffinityBonusPercent,
		TotalBonusPercent:    effect.TotalBonusPercent,
		AffectedCellBonus:    effect.AffectedCellBonus,
		DefenseBonus:         effect.DefenseBonus,
		ScoreBonus:           effect.ScoreBonus,
	}
}
