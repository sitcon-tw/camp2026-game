package territory

import (
	"math"
	"testing"

	"github.com/sitcon-tw/camp2026-game/internal/content"
)

func almostEqual(a float64, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func attackStone(id string, attack int, fatigue int) DeployedStone {
	return DeployedStone{
		InstanceID: "instance_" + id,
		SitoneID:   id,
		Fatigue:    fatigue,
		Sitone:     content.Sitone{ID: id, Attack: attack, Repeatable: true},
	}
}

func TestComputeEffectiveAttack(t *testing.T) {
	cfg := DefaultEngineConfig()

	// Full-strength squad of a 5-member team: 10 stones × 15 attack = 150
	// against a standard of 150 → A = 1.0.
	attackers := make([]DeployedStone, 0, 10)
	for i := 0; i < 10; i++ {
		attackers = append(attackers, attackStone("stone", 15, 0))
	}
	if a := ComputeEffectiveAttack(cfg, attackers, 5); !almostEqual(a, 1.0) {
		t.Fatalf("expected A=1.0, got %f", a)
	}

	// Fatigue 100 halves attack with the default divisor of 200.
	fatigued := []DeployedStone{attackStone("stone", 100, 100)}
	expected := 100.0 * 0.5 / float64(FullStrengthStandard(5))
	if a := ComputeEffectiveAttack(cfg, fatigued, 5); !almostEqual(a, expected) {
		t.Fatalf("expected A=%f with fatigue, got %f", expected, a)
	}

	// semantic_chain adds a squad-wide percentage bonus.
	boosted := []DeployedStone{
		attackStone("stone", 100, 0),
		{
			SitoneID: "chain",
			Sitone: content.Sitone{
				ID:          "chain",
				Attack:      0,
				EffectKind:  content.SitoneEffectSemanticChain,
				EffectValue: 10,
			},
		},
	}
	expected = 100.0 * 1.1 / float64(FullStrengthStandard(5))
	if a := ComputeEffectiveAttack(cfg, boosted, 5); !almostEqual(a, expected) {
		t.Fatalf("expected A=%f with semantic_chain, got %f", expected, a)
	}
}

func TestComputeEffectiveDefense(t *testing.T) {
	cfg := DefaultEngineConfig()

	defense := []DefenseStone{
		{Sitone: content.Sitone{Defense: 100}},
		{Sitone: content.Sitone{Defense: 100}, Captive: true}, // 50% value
	}
	expected := (100.0 + 50.0) / float64(FullStrengthStandard(5))
	if d := ComputeEffectiveDefense(cfg, defense, 5, map[string]float64{}); !almostEqual(d, expected) {
		t.Fatalf("expected D=%f, got %f", expected, d)
	}

	// fortify adds a percentage bonus on the defense total.
	fortified := []DefenseStone{
		{Sitone: content.Sitone{Defense: 100, EffectKind: content.SitoneEffectFortify, EffectValue: 20}},
	}
	expected = 100.0 * 1.2 / float64(FullStrengthStandard(5))
	if d := ComputeEffectiveDefense(cfg, fortified, 5, map[string]float64{}); !almostEqual(d, expected) {
		t.Fatalf("expected D=%f with fortify, got %f", expected, d)
	}

	// defense_pierce on the attacker side shaves the total.
	pierced := ComputeEffectiveDefense(cfg, fortified, 5, map[string]float64{
		content.SitoneEffectDefensePierce: 50,
	})
	expected = 100.0 * 1.2 * 0.5 / float64(FullStrengthStandard(5))
	if !almostEqual(pierced, expected) {
		t.Fatalf("expected D=%f with defense_pierce, got %f", expected, pierced)
	}

	// prompt_firewall dampens the fortify bonus (100% removes it fully).
	firewalled := ComputeEffectiveDefense(cfg, fortified, 5, map[string]float64{
		content.SitoneEffectPromptFirewall: 100,
	})
	expected = 100.0 / float64(FullStrengthStandard(5))
	if !almostEqual(firewalled, expected) {
		t.Fatalf("expected D=%f with prompt_firewall, got %f", expected, firewalled)
	}
}

func TestSelectStealTargetsRecapturePriority(t *testing.T) {
	cfg := DefaultEngineConfig()
	pool := []StealCandidate{
		{SitoneID: "regular_a", OwnerPlayerID: "p1"},
		{SitoneID: "regular_b", OwnerPlayerID: "p2"},
		{SitoneID: "captive_home", CaptiveRecordID: "cap1", CaptiveInstanceID: "inst1", Recapture: true},
		{SitoneID: "regular_c", OwnerPlayerID: "p3"},
	}

	// Any seed: the recapture candidate must always be selected first.
	for _, seed := range []string{"seed_a", "seed_b", "seed_c"} {
		roll := NewRoll(seed)
		stolen := selectStealTargets(cfg, pool, 0.10, roll)
		if len(stolen) == 0 {
			t.Fatalf("expected at least one stolen stone")
		}
		if !stolen[0].Recapture || stolen[0].SitoneID != "captive_home" {
			t.Fatalf("seed %s: expected recapture candidate first, got %+v", seed, stolen[0])
		}
	}

	// Steal count is capped by StealMaxCount even with huge G.
	roll := NewRoll("seed_cap")
	stolen := selectStealTargets(cfg, pool, 5.0, roll)
	if len(stolen) > cfg.StealMaxCount {
		t.Fatalf("expected at most %d stolen, got %d", cfg.StealMaxCount, len(stolen))
	}
}

func TestRiskPassUniqueGoesToYansan(t *testing.T) {
	cfg := DefaultEngineConfig()
	// Force certain missing and death.
	cfg.BaseMissingChance = 1.0
	cfg.BaseDeathChance = 1.0

	uniqueStone := DeployedStone{
		InstanceID: "inst_unique",
		SitoneID:   "unique_stone",
		Sitone:     content.Sitone{ID: "unique_stone", Attack: 10, Unique: true},
	}
	repeatable := DeployedStone{
		InstanceID: "inst_repeat",
		SitoneID:   "repeat_stone",
		Sitone:     content.Sitone{ID: "repeat_stone", Attack: 10, Repeatable: true},
	}

	in := MissionInput{
		Attackers:        []DeployedStone{uniqueStone, repeatable},
		AttackerTeamSize: 5,
		DefenderTeamSize: 5,
	}
	roll := NewRoll("seed_unique")
	results := riskPass(cfg, in, map[string]float64{}, 0, roll)

	if results[0].Outcome != StoneOutcomeProtected {
		t.Fatalf("expected unique stone protected_at_yansan, got %s", results[0].Outcome)
	}
	if results[1].Outcome != StoneOutcomeMissing {
		t.Fatalf("expected repeatable stone missing (missing rolls before death), got %s", results[1].Outcome)
	}
}

func TestRiskPassDefeatCanCaptureRepeatable(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.DefeatCaptureFactor = 10 // force capture chance to the max
	cfg.DefeatCaptureMax = 1.0
	cfg.BaseMissingChance = 0
	cfg.BaseDeathChance = 0

	repeatable := DeployedStone{
		InstanceID: "inst_repeat",
		SitoneID:   "repeat_stone",
		Sitone:     content.Sitone{ID: "repeat_stone", Attack: 10, Repeatable: true},
	}
	uniqueStone := DeployedStone{
		InstanceID: "inst_unique",
		SitoneID:   "unique_stone",
		Sitone:     content.Sitone{ID: "unique_stone", Attack: 10, Unique: true},
	}

	in := MissionInput{
		Attackers:        []DeployedStone{repeatable, uniqueStone},
		AttackerTeamSize: 5,
		DefenderTeamSize: 5,
	}
	roll := NewRoll("seed_defeat")
	results := riskPass(cfg, in, map[string]float64{}, -0.5, roll)

	if results[0].Outcome != StoneOutcomeCaptured {
		t.Fatalf("expected repeatable stone captured on defeat, got %s", results[0].Outcome)
	}
	if results[1].Outcome == StoneOutcomeCaptured {
		t.Fatalf("unique stones must never be captured")
	}
}

func TestResolveMissionDeterministic(t *testing.T) {
	cfg := DefaultEngineConfig()
	in := MissionInput{
		Attackers: []DeployedStone{
			attackStone("a", 30, 0),
			attackStone("b", 25, 10),
		},
		AttackerTeamSize: 5,
		Defense: []DefenseStone{
			{Sitone: content.Sitone{Defense: 10}},
		},
		DefenderTeamSize: 5,
		StealPool: []StealCandidate{
			{SitoneID: "loot", OwnerPlayerID: "p1"},
		},
		CostPaid: 150,
	}

	first := ResolveMission(cfg, in, NewRoll("seed_fixed"))
	second := ResolveMission(cfg, in, NewRoll("seed_fixed"))

	if first.A != second.A || first.D != second.D || first.StealHappened != second.StealHappened {
		t.Fatalf("resolution must be deterministic for the same seed")
	}
	if len(first.Stones) != len(second.Stones) {
		t.Fatalf("stone results must be deterministic")
	}
	for i := range first.Stones {
		if first.Stones[i].Outcome != second.Stones[i].Outcome || first.Stones[i].FatigueGain != second.Stones[i].FatigueGain {
			t.Fatalf("stone %d differs across replays", i)
		}
	}
	if first.A <= first.D {
		t.Fatalf("expected attacker advantage in this fixture, A=%f D=%f", first.A, first.D)
	}
}

func TestOpRefund(t *testing.T) {
	cfg := DefaultEngineConfig()
	attackers := []DeployedStone{
		{Sitone: content.Sitone{EffectKind: content.SitoneEffectOpRefund, EffectValue: 20}},
	}
	if refund := opRefund(cfg, attackers, 100); refund != 20 {
		t.Fatalf("expected refund 20, got %d", refund)
	}
	// Cap at OpRefundMaxFraction.
	heavy := []DeployedStone{
		{Sitone: content.Sitone{EffectKind: content.SitoneEffectOpRefund, EffectValue: 40}},
		{Sitone: content.Sitone{EffectKind: content.SitoneEffectOpRefund, EffectValue: 40}},
	}
	if refund := opRefund(cfg, heavy, 100); refund != 50 {
		t.Fatalf("expected capped refund 50, got %d", refund)
	}
}

func TestConvertAndRedeemChances(t *testing.T) {
	cfg := DefaultEngineConfig()

	if chance := ConvertSuccessChance(cfg, 0, 0); chance != 0.60 {
		t.Fatalf("expected 0.60, got %f", chance)
	}
	if chance := ConvertSuccessChance(cfg, 10, 0); chance != 0.20 {
		t.Fatalf("expected floor 0.20, got %f", chance)
	}
	boosted := ConvertSuccessChance(cfg, 0, 1000000)
	if boosted != 0.60+cfg.ConvertBoostMax {
		t.Fatalf("expected boost capped at %f, got %f", 0.60+cfg.ConvertBoostMax, boosted)
	}

	if chance := RedeemSuccessChance(cfg, 0); chance != 0.85 {
		t.Fatalf("expected 0.85, got %f", chance)
	}
	if chance := RedeemSuccessChance(cfg, 100); chance != 0.25 {
		t.Fatalf("expected floor 0.25, got %f", chance)
	}
}

func TestDefenseBand(t *testing.T) {
	cfg := DefaultEngineConfig()
	if band := defenseBand(cfg, 0.1); band != "weak" {
		t.Fatalf("expected weak, got %s", band)
	}
	if band := defenseBand(cfg, 0.6); band != "medium" {
		t.Fatalf("expected medium, got %s", band)
	}
	if band := defenseBand(cfg, 1.2); band != "strong" {
		t.Fatalf("expected strong, got %s", band)
	}
}

func TestRollMissionDurationCapped(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.MissionBaseDuration = MissionMaxDuration
	cfg.MissionExtraDuration = MissionMaxDuration
	if duration := RollMissionDuration(cfg, NewRoll("seed_duration")); duration > MissionMaxDuration {
		t.Fatalf("mission duration must be capped at %s, got %s", MissionMaxDuration, duration)
	}
}
