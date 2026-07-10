package fronts

import (
	"testing"

	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestFrontSitoneEffectUsesOnlyMatchingAffinity(t *testing.T) {
	store := testcontent.Load(t)
	effect := frontSitoneEffect(store, "expand", []string{
		"stone_explorer_base",
		"stone_engineering_base",
	})
	if effect.SelectedCount != 2 || effect.SquadBonusPercent != 5 {
		t.Fatalf("unexpected squad bonus: %#v", effect)
	}
	if effect.AffinityBonusPercent != 5 || effect.TotalBonusPercent != 10 {
		t.Fatalf("only the exploration sitone should match expand: %#v", effect)
	}
}

func TestFrontSitoneEffectCapsLargeAndRepeatedAbilities(t *testing.T) {
	store := testcontent.Load(t)
	effect := frontSitoneEffect(store, "rescue", []string{
		"stone_fireside",
		"stone_fireside",
		"stone_fireside",
		"stone_fireside",
		"stone_fireside",
	})
	if effect.SquadBonusPercent != 20 || effect.AffinityBonusPercent != 40 || effect.TotalBonusPercent != 60 {
		t.Fatalf("front sitone caps were not applied: %#v", effect)
	}
}

func TestValidateTerritoryCommandRequiresOneToFiveSitones(t *testing.T) {
	x, y := 1, 1
	base := CreateCommandRequest{
		ClientCommandID: "command-1",
		Kind:            "expand",
		TargetX:         &x,
		TargetY:         &y,
	}
	if err := validateCommandRequest(base); err == nil {
		t.Fatal("command without sitones must fail")
	}
	base.SitoneIDs = []string{"one", "two", "three", "four", "five", "six"}
	if err := validateCommandRequest(base); err == nil {
		t.Fatal("command with more than five sitones must fail")
	}
	base.SitoneIDs = []string{"same", "same", "same", "same", "same"}
	if err := validateCommandRequest(base); err != nil {
		t.Fatalf("five repeated sitones should pass request validation: %v", err)
	}
}
