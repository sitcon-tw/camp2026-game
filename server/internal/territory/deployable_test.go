package territory

import (
	"testing"

	"github.com/sitcon-tw/camp2026-game/internal/content"
)

func TestCanDeploySitoneExcludesLimitedStaffRewards(t *testing.T) {
	sitone := content.Sitone{
		Rarity:  "limited",
		Attack:  17,
		Defense: 9,
		Unique:  true,
	}

	if CanDeploySitone(sitone) || CanAttackWithSitone(sitone) || CanDefendWithSitone(sitone) {
		t.Fatalf("limited staff reward should not be deployable: %#v", sitone)
	}
}

func TestCanDeploySitoneAllowsRegularAttackAndDefense(t *testing.T) {
	sitone := content.Sitone{
		Rarity:     "common",
		Attack:     12,
		Defense:    5,
		Repeatable: true,
	}

	if !CanDeploySitone(sitone) || !CanAttackWithSitone(sitone) || !CanDefendWithSitone(sitone) {
		t.Fatalf("regular combat sitone should be deployable: %#v", sitone)
	}
}
