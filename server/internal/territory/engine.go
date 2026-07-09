package territory

// Mission / process resolution engine (design §7-§11, §14). All math is
// pure and deterministic given a Roll, so staff can replay any resolution
// from the stored random_seed_ref. Every tunable lives in EngineConfig and
// stays server-side only: clients never see chances, rolls or multipliers,
// only outcome labels.

import (
	"math"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
)

// EngineConfig holds every tunable of the resolution engine.
type EngineConfig struct {
	// MissionBaseDuration / MissionExtraDuration: a deployed mission
	// resolves after base + roll(0..extra), capped at MissionMaxDuration.
	MissionBaseDuration  time.Duration
	MissionExtraDuration time.Duration

	// Steal parameters (A > D branch).
	StealChanceMin     float64
	StealChanceMax     float64
	StealPortionJitter float64
	StealMaxCount      int

	// Per-stone risk pass.
	BaseMissingChance    float64
	BaseDeathChance      float64
	DefeatRiskMultiplier float64
	DefeatCaptureFactor  float64
	DefeatCaptureMax     float64
	FatigueGainMin       int
	FatigueGainMax       int
	// FatigueAttackDivisor: effective attack = attack × (1 − fatigue/divisor).
	FatigueAttackDivisor float64
	// FatiguedLabelThreshold: stones gaining at least this much fatigue are
	// listed as "fatigued" in the public result summary.
	FatiguedLabelThreshold int

	// CapturedDefenseFactor: defense stones the team only holds as
	// captives count at this fraction of their value (design §8: 50%).
	CapturedDefenseFactor float64

	// OpRefundMaxFraction caps the total op_refund effect.
	OpRefundMaxFraction float64

	// Defense strength bands revealed by scout_recon.
	DefenseBandWeakMax   float64
	DefenseBandMediumMax float64

	// Convert (captive → new team) process.
	ConvertBaseRate          float64
	ConvertDecayPerPrior     float64
	ConvertFloor             float64
	ConvertBoostPerOpenPower float64 // extra chance per 1 extra open power
	ConvertBoostMax          float64
	ConvertResolveMin        time.Duration
	ConvertResolveMax        time.Duration

	// Redeem (unique stone back from yansan) process.
	RedeemBaseRate      float64
	RedeemDecayPerPrior float64
	RedeemFloor         float64
	RedeemResolveMin    time.Duration
	RedeemResolveMax    time.Duration

	// Rescue (missing stone) process.
	RescueSuccessRate      float64
	RescueStillMissingRate float64
	RescueDeathRate        float64
	RescueResolveMin       time.Duration
	RescueResolveMax       time.Duration
	RescueFatigueGain      int
}

// DefaultEngineConfig returns the baseline engine tunables.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MissionBaseDuration:  20 * time.Minute,
		MissionExtraDuration: 20 * time.Minute,

		StealChanceMin:     0.05,
		StealChanceMax:     0.90,
		StealPortionJitter: 0.25,
		StealMaxCount:      3,

		BaseMissingChance:    0.08,
		BaseDeathChance:      0.03,
		DefeatRiskMultiplier: 2.5,
		DefeatCaptureFactor:  0.5,
		DefeatCaptureMax:     0.6,
		FatigueGainMin:       15,
		FatigueGainMax:       35,
		FatigueAttackDivisor: 200,

		FatiguedLabelThreshold: 25,

		CapturedDefenseFactor: 0.5,

		OpRefundMaxFraction: 0.5,

		DefenseBandWeakMax:   0.4,
		DefenseBandMediumMax: 0.8,

		ConvertBaseRate:          0.60,
		ConvertDecayPerPrior:     0.10,
		ConvertFloor:             0.20,
		ConvertBoostPerOpenPower: 0.001,
		ConvertBoostMax:          0.15,
		ConvertResolveMin:        20 * time.Minute,
		ConvertResolveMax:        35 * time.Minute,

		RedeemBaseRate:      0.85,
		RedeemDecayPerPrior: 0.08,
		RedeemFloor:         0.25,
		RedeemResolveMin:    20 * time.Minute,
		RedeemResolveMax:    35 * time.Minute,

		RescueSuccessRate:      0.55,
		RescueStillMissingRate: 0.25,
		RescueDeathRate:        0.05,
		RescueResolveMin:       10 * time.Minute,
		RescueResolveMax:       25 * time.Minute,
		RescueFatigueGain:      20,
	}
}

// DeployedStone is one attacker-side stone in a mission.
type DeployedStone struct {
	InstanceID     string
	SitoneID       string
	OriginPlayerID string
	Fatigue        int
	Sitone         content.Sitone
}

// DefenseStone is one defender-side stone read at resolve time.
type DefenseStone struct {
	Sitone  content.Sitone
	Captive bool // held only as a captive → counts at CapturedDefenseFactor
}

// StealCandidate is one stealable stone on the defender side.
type StealCandidate struct {
	SitoneID string
	// OwnerPlayerID is the defender member whose quantity is decremented
	// when a quantity-based stone is stolen.
	OwnerPlayerID string
	// CaptiveInstanceID / CaptiveRecordID are set when the candidate is a
	// captive originally belonging to the attacker (原隊搶回 priority).
	CaptiveInstanceID string
	CaptiveRecordID   string
	Recapture         bool
}

// Stone outcomes after the risk pass.
const (
	StoneOutcomeReturned  = "returned"
	StoneOutcomeMissing   = "missing"
	StoneOutcomeDead      = "dead"
	StoneOutcomeProtected = "protected_at_yansan"
	StoneOutcomeCaptured  = "captured"
)

// StoneResult is the resolved outcome of one deployed stone.
type StoneResult struct {
	Stone       DeployedStone
	Outcome     string
	FatigueGain int
}

// MissionInput is everything the engine needs to resolve a mission.
type MissionInput struct {
	Attackers        []DeployedStone
	AttackerTeamSize int
	Defense          []DefenseStone
	DefenderTeamSize int
	StealPool        []StealCandidate
	CostPaid         int
}

// MissionResolution is the full private result of a mission resolution.
type MissionResolution struct {
	A float64
	D float64
	G float64

	AttackerWon   bool
	StealHappened bool
	Stolen        []StealCandidate

	Stones []StoneResult

	DefenseBand   string
	HasScoutRecon bool
	OpRefund      int
}

// ResolveMission resolves a deployed mission deterministically from the
// given roll sequence. Roll order: steal chance → steal count/selection →
// per-stone risk (in input order: capture, missing, death, fatigue).
func ResolveMission(cfg EngineConfig, in MissionInput, roll *Roll) MissionResolution {
	attackerEffects := collectEffects(in.Attackers)

	a := ComputeEffectiveAttack(cfg, in.Attackers, in.AttackerTeamSize)
	d := ComputeEffectiveDefense(cfg, in.Defense, in.DefenderTeamSize, attackerEffects)
	g := a - d

	res := MissionResolution{
		A:             a,
		D:             d,
		G:             g,
		AttackerWon:   g > 0,
		DefenseBand:   defenseBand(cfg, d),
		HasScoutRecon: attackerEffects[content.SitoneEffectScoutRecon] > 0,
	}

	if g > 0 && len(in.StealPool) > 0 {
		stealChance := clampFloat(g+attackerEffects[content.SitoneEffectStealBoost]/100, cfg.StealChanceMin, cfg.StealChanceMax)
		if roll.Float() < stealChance {
			res.StealHappened = true
			res.Stolen = selectStealTargets(cfg, in.StealPool, g, roll)
		}
	}

	res.Stones = riskPass(cfg, in, attackerEffects, g, roll)
	res.OpRefund = opRefund(cfg, in.Attackers, in.CostPaid)
	return res
}

// ComputeEffectiveAttack computes A: fatigue-reduced attack total with
// attacker effect bonuses, normalized against the attacker's
// full-strength standard.
func ComputeEffectiveAttack(cfg EngineConfig, attackers []DeployedStone, teamSize int) float64 {
	divisor := cfg.FatigueAttackDivisor
	if divisor <= 0 {
		divisor = 200
	}
	total := 0.0
	bonusPercent := 0.0
	for _, stone := range attackers {
		factor := 1 - float64(stone.Fatigue)/divisor
		if factor < 0 {
			factor = 0
		}
		total += float64(stone.Sitone.Attack) * factor
		if stone.Sitone.EffectKind == content.SitoneEffectSemanticChain ||
			stone.Sitone.EffectKind == content.SitoneEffectTurtleDraw {
			// semantic_chain / turtle_draw add a squad-wide attack bonus.
			bonusPercent += float64(stone.Sitone.EffectValue)
		}
	}
	total *= 1 + bonusPercent/100
	standard := FullStrengthStandard(teamSize)
	if standard <= 0 {
		return 0
	}
	return total / float64(standard)
}

// ComputeEffectiveDefense computes D: defense total (captive-held stones
// at CapturedDefenseFactor) with fortify/architecture_run bonuses, reduced
// by attacker defense_pierce / prompt_firewall, normalized against the
// defender's full-strength standard.
func ComputeEffectiveDefense(cfg EngineConfig, defense []DefenseStone, teamSize int, attackerEffects map[string]float64) float64 {
	base := 0.0
	bonusPercent := 0.0
	for _, stone := range defense {
		value := float64(stone.Sitone.Defense)
		if stone.Captive {
			value *= cfg.CapturedDefenseFactor
		}
		base += value
		if stone.Sitone.EffectKind == content.SitoneEffectFortify ||
			stone.Sitone.EffectKind == content.SitoneEffectArchitectureRun {
			bonusPercent += float64(stone.Sitone.EffectValue)
		}
	}

	// prompt_firewall dampens the defender's effect bonuses.
	if firewall := attackerEffects[content.SitoneEffectPromptFirewall]; firewall > 0 {
		bonusPercent *= clampFloat(1-firewall/100, 0, 1)
	}
	total := base * (1 + bonusPercent/100)

	// defense_pierce shaves a percentage off the whole defense.
	if pierce := attackerEffects[content.SitoneEffectDefensePierce]; pierce > 0 {
		total *= clampFloat(1-pierce/100, 0, 1)
	}

	standard := FullStrengthStandard(teamSize)
	if standard <= 0 {
		return 0
	}
	return total / float64(standard)
}

// selectStealTargets picks the stolen stones: recapture candidates (the
// attacker's own origin-team captives held by the defender, design §8)
// always come first, then regular repeatable stones. Both groups are
// shuffled deterministically by the roll.
func selectStealTargets(cfg EngineConfig, pool []StealCandidate, g float64, roll *Roll) []StealCandidate {
	recapture := make([]StealCandidate, 0, len(pool))
	regular := make([]StealCandidate, 0, len(pool))
	for _, candidate := range pool {
		if candidate.Recapture {
			recapture = append(recapture, candidate)
		} else {
			regular = append(regular, candidate)
		}
	}
	shuffleCandidates(recapture, roll)
	shuffleCandidates(regular, roll)
	ordered := append(recapture, regular...)

	portion := g * roll.Jitter(cfg.StealPortionJitter)
	count := int(math.Ceil(portion * float64(len(ordered))))
	if count < 1 {
		count = 1
	}
	if cfg.StealMaxCount > 0 && count > cfg.StealMaxCount {
		count = cfg.StealMaxCount
	}
	if count > len(ordered) {
		count = len(ordered)
	}
	return ordered[:count]
}

func shuffleCandidates(candidates []StealCandidate, roll *Roll) {
	for i := len(candidates) - 1; i > 0; i-- {
		j := roll.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
}

// riskPass rolls the fate of every deployed attacker stone. On defeat
// (D > A) risks are amplified and repeatable stones may be captured by the
// defender. Unique stones never go missing/die/get captured: any bad event
// redirects them to yansan protection (design §9).
func riskPass(cfg EngineConfig, in MissionInput, effects map[string]float64, g float64, roll *Roll) []StoneResult {
	defeat := g < 0
	riskMultiplier := 1.0
	captureChance := 0.0
	if defeat {
		riskMultiplier = cfg.DefeatRiskMultiplier
		captureChance = clampFloat((-g)*cfg.DefeatCaptureFactor-effects[content.SitoneEffectTurtleDraw]/100, 0, cfg.DefeatCaptureMax)
	}

	calibrate := effects[content.SitoneEffectCommandCalibrate]
	firesideReduction := effects[content.SitoneEffectFiresideEmbers]

	results := make([]StoneResult, 0, len(in.Attackers))
	for _, stone := range in.Attackers {
		result := StoneResult{Stone: stone, Outcome: StoneOutcomeReturned}

		missingChance := cfg.BaseMissingChance * riskMultiplier
		if stone.Sitone.EffectKind == content.SitoneEffectEvadeMissing {
			missingChance *= clampFloat(1-float64(stone.Sitone.EffectValue)/100, 0, 1)
		}
		missingChance *= clampFloat(1-calibrate/100, 0, 1)

		deathChance := cfg.BaseDeathChance * riskMultiplier
		if stone.Sitone.EffectKind == content.SitoneEffectDeathGuard {
			deathChance *= clampFloat(1-float64(stone.Sitone.EffectValue)/100, 0, 1)
		}

		unique := stone.Sitone.Unique

		captured := defeat && !unique && stone.Sitone.Repeatable && roll.Float() < captureChance
		missing := roll.Float() < missingChance
		died := roll.Float() < deathChance
		fatigueGain := rollFatigueGain(cfg, stone, firesideReduction, roll)

		switch {
		case captured:
			result.Outcome = StoneOutcomeCaptured
		case unique && (missing || died):
			// Unique stones never get lost or die: sent to yansan (§9).
			result.Outcome = StoneOutcomeProtected
		case missing:
			result.Outcome = StoneOutcomeMissing
		case died && stone.Sitone.Repeatable:
			result.Outcome = StoneOutcomeDead
		default:
			result.FatigueGain = fatigueGain
		}
		results = append(results, result)
	}
	return results
}

func rollFatigueGain(cfg EngineConfig, stone DeployedStone, firesideReduction float64, roll *Roll) int {
	span := cfg.FatigueGainMax - cfg.FatigueGainMin
	gain := float64(cfg.FatigueGainMin)
	if span > 0 {
		gain += float64(roll.IntN(span + 1))
	}
	// fireside_embers reduces the whole squad's fatigue gain.
	gain *= clampFloat(1-firesideReduction/100, 0, 1)
	// fatigue_relief reduces the carrying stone's own gain.
	if stone.Sitone.EffectKind == content.SitoneEffectFatigueRelief {
		gain *= clampFloat(1-float64(stone.Sitone.EffectValue)/100, 0, 1)
	}
	if gain < 0 {
		gain = 0
	}
	return int(math.Round(gain))
}

func opRefund(cfg EngineConfig, attackers []DeployedStone, costPaid int) int {
	if costPaid <= 0 {
		return 0
	}
	fraction := 0.0
	for _, stone := range attackers {
		if stone.Sitone.EffectKind == content.SitoneEffectOpRefund {
			fraction += float64(stone.Sitone.EffectValue) / 100
		}
	}
	fraction = clampFloat(fraction, 0, cfg.OpRefundMaxFraction)
	if fraction <= 0 {
		return 0
	}
	return int(math.Round(float64(costPaid) * fraction))
}

func defenseBand(cfg EngineConfig, d float64) string {
	switch {
	case d <= cfg.DefenseBandWeakMax:
		return "weak"
	case d <= cfg.DefenseBandMediumMax:
		return "medium"
	default:
		return "strong"
	}
}

// collectEffects sums attacker effect values (percent points) by kind.
func collectEffects(attackers []DeployedStone) map[string]float64 {
	effects := make(map[string]float64)
	for _, stone := range attackers {
		if stone.Sitone.EffectKind == "" {
			continue
		}
		effects[stone.Sitone.EffectKind] += float64(stone.Sitone.EffectValue)
	}
	return effects
}

// RollMissionDuration returns the deploy → resolve duration: base + random
// extra, capped at MissionMaxDuration.
func RollMissionDuration(cfg EngineConfig, roll *Roll) time.Duration {
	duration := cfg.MissionBaseDuration
	if cfg.MissionExtraDuration > 0 {
		duration += time.Duration(roll.IntN(int(cfg.MissionExtraDuration/time.Second)+1)) * time.Second
	}
	if duration > MissionMaxDuration {
		duration = MissionMaxDuration
	}
	return duration
}

// RollProcessDuration returns a uniform duration in [min, max].
func RollProcessDuration(minDuration time.Duration, maxDuration time.Duration, roll *Roll) time.Duration {
	if maxDuration <= minDuration {
		return minDuration
	}
	span := int(maxDuration/time.Second) - int(minDuration/time.Second)
	return minDuration + time.Duration(roll.IntN(span+1))*time.Second
}

// ConvertSuccessChance: base 60% − 10% per prior successful convert by the
// team (floor 20%), plus a capped boost from extra invested open power.
func ConvertSuccessChance(cfg EngineConfig, priorSuccessfulConverts int, extraOpenPower int) float64 {
	chance := cfg.ConvertBaseRate - cfg.ConvertDecayPerPrior*float64(maxInt(priorSuccessfulConverts, 0))
	if chance < cfg.ConvertFloor {
		chance = cfg.ConvertFloor
	}
	boost := cfg.ConvertBoostPerOpenPower * float64(maxInt(extraOpenPower, 0))
	if boost > cfg.ConvertBoostMax {
		boost = cfg.ConvertBoostMax
	}
	return clampFloat(chance+boost, 0, 0.99)
}

// RedeemSuccessChance: base 85% − 8% per prior team redeem (floor 25%).
func RedeemSuccessChance(cfg EngineConfig, priorTeamRedeems int) float64 {
	chance := cfg.RedeemBaseRate - cfg.RedeemDecayPerPrior*float64(maxInt(priorTeamRedeems, 0))
	if chance < cfg.RedeemFloor {
		chance = cfg.RedeemFloor
	}
	return chance
}

// Rescue outcomes.
const (
	RescueOutcomeSuccess      = "success"
	RescueOutcomeFail         = "fail"
	RescueOutcomeStillMissing = "still_missing"
	RescueOutcomeDead         = "dead"
)

// RollRescueOutcome resolves a rescue process. rescue_boost / retransmit
// effects on the missing stone improve the success rate. The dead outcome
// only applies to repeatable stones; unique stones redirect to yansan
// protection at the call site.
func RollRescueOutcome(cfg EngineConfig, stone content.Sitone, roll *Roll) string {
	successRate := cfg.RescueSuccessRate
	if stone.EffectKind == content.SitoneEffectRescueBoost || stone.EffectKind == content.SitoneEffectRetransmit {
		successRate += float64(stone.EffectValue) / 100
	}
	successRate = clampFloat(successRate, 0, 0.95)

	value := roll.Float()
	switch {
	case value < successRate:
		return RescueOutcomeSuccess
	case value < successRate+cfg.RescueStillMissingRate:
		return RescueOutcomeStillMissing
	case value < successRate+cfg.RescueStillMissingRate+cfg.RescueDeathRate:
		return RescueOutcomeDead
	default:
		return RescueOutcomeFail
	}
}

// Risk labels exposed to players (design §14: labels only, never numbers).
const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"
)

// AttackRiskLevel derives the overall risk label from the attacker's own
// strength estimate (the defender side stays an unknown bucket).
func AttackRiskLevel(attackPercent float64) string {
	switch {
	case attackPercent >= 0.9:
		return RiskLow
	case attackPercent >= 0.6:
		return RiskMedium
	default:
		return RiskHigh
	}
}

// MissingRiskLevel labels the missing risk from average fatigue and squad
// effects.
func MissingRiskLevel(cfg EngineConfig, attackers []DeployedStone) string {
	effects := collectEffects(attackers)
	chance := cfg.BaseMissingChance
	chance *= clampFloat(1-effects[content.SitoneEffectCommandCalibrate]/100, 0, 1)
	chance += averageFatigue(attackers) / 1000
	switch {
	case chance <= 0.05:
		return RiskLow
	case chance <= 0.10:
		return RiskMedium
	default:
		return RiskHigh
	}
}

// DeathRiskLevel labels the death risk.
func DeathRiskLevel(cfg EngineConfig, attackers []DeployedStone) string {
	guarded := false
	for _, stone := range attackers {
		if stone.Sitone.EffectKind == content.SitoneEffectDeathGuard {
			guarded = true
		}
	}
	chance := cfg.BaseDeathChance
	if guarded {
		chance *= 0.5
	}
	chance += averageFatigue(attackers) / 2000
	switch {
	case chance <= 0.02:
		return RiskLow
	case chance <= 0.05:
		return RiskMedium
	default:
		return RiskHigh
	}
}

func averageFatigue(attackers []DeployedStone) float64 {
	if len(attackers) == 0 {
		return 0
	}
	total := 0
	for _, stone := range attackers {
		total += stone.Fatigue
	}
	return float64(total) / float64(len(attackers))
}

func clampFloat(value float64, low float64, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
