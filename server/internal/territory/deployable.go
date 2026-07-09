package territory

import "github.com/sitcon-tw/camp2026-game/internal/content"

// Limited sitones are staff/event rewards and should not be assigned to
// territory attack or defense slots.
func CanDeploySitone(sitone content.Sitone) bool {
	return sitone.Rarity != "limited" && (sitone.Attack > 0 || sitone.Defense > 0)
}

func CanAttackWithSitone(sitone content.Sitone) bool {
	return CanDeploySitone(sitone) && sitone.Attack > 0
}

func CanDefendWithSitone(sitone content.Sitone) bool {
	return CanDeploySitone(sitone) && sitone.Defense > 0
}
