package content

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const sitonesFile = "sitones.toml"

var validSitoneTypes = map[string]struct{}{
	"exploration":   {},
	"inspiration":   {},
	"resonance":     {},
	"engineering":   {},
	"entertainment": {},
}

const (
	SitoneAbilityMaterialDropRate     = "material_drop_rate"
	SitoneAbilityAnswerScoreBonus     = "answer_score_bonus"
	SitoneAbilityOpenPowerBonus       = "open_power_bonus"
	SitoneAbilityEliminateWrongChoice = "eliminate_wrong_choice"
)

var validSitoneAbilityKinds = map[string]struct{}{
	SitoneAbilityMaterialDropRate:     {},
	SitoneAbilityAnswerScoreBonus:     {},
	SitoneAbilityOpenPowerBonus:       {},
	SitoneAbilityEliminateWrongChoice: {},
}

const (
	SitoneEffectScoutRecon       = "scout_recon"
	SitoneEffectEvadeMissing     = "evade_missing"
	SitoneEffectFortify          = "fortify"
	SitoneEffectDeathGuard       = "death_guard"
	SitoneEffectStealBoost       = "steal_boost"
	SitoneEffectDefensePierce    = "defense_pierce"
	SitoneEffectFatigueRelief    = "fatigue_relief"
	SitoneEffectRescueBoost      = "rescue_boost"
	SitoneEffectFatigueRecover   = "fatigue_recover"
	SitoneEffectOpRefund         = "op_refund"
	SitoneEffectFiresideEmbers   = "fireside_embers"
	SitoneEffectTurtleDraw       = "turtle_draw"
	SitoneEffectSemanticChain    = "semantic_chain"
	SitoneEffectPromptFirewall   = "prompt_firewall"
	SitoneEffectCommandCalibrate = "command_calibrate"
	SitoneEffectArchitectureRun  = "architecture_run"
	SitoneEffectRetransmit       = "retransmit"
)

var validSitoneEffectKinds = map[string]struct{}{
	SitoneEffectScoutRecon:       {},
	SitoneEffectEvadeMissing:     {},
	SitoneEffectFortify:          {},
	SitoneEffectDeathGuard:       {},
	SitoneEffectStealBoost:       {},
	SitoneEffectDefensePierce:    {},
	SitoneEffectFatigueRelief:    {},
	SitoneEffectRescueBoost:      {},
	SitoneEffectFatigueRecover:   {},
	SitoneEffectOpRefund:         {},
	SitoneEffectFiresideEmbers:   {},
	SitoneEffectTurtleDraw:       {},
	SitoneEffectSemanticChain:    {},
	SitoneEffectPromptFirewall:   {},
	SitoneEffectCommandCalibrate: {},
	SitoneEffectArchitectureRun:  {},
	SitoneEffectRetransmit:       {},
}

type Sitone struct {
	ID                 string `toml:"id"`
	Name               string `toml:"name"`
	Type               string `toml:"type"`
	Rarity             string `toml:"rarity"`
	Style              string `toml:"style"`
	Description        string `toml:"description"`
	IconPath           string `toml:"icon_path"`
	AbilityName        string `toml:"ability_name"`
	AbilityKind        string `toml:"ability_kind"`
	AbilityValue       int    `toml:"ability_value"`
	AbilityCount       int    `toml:"ability_count"`
	AbilityDescription string `toml:"ability_description"`
	Attack             int    `toml:"attack"`
	Defense            int    `toml:"defense"`
	Repeatable         bool   `toml:"repeatable"`
	Unique             bool   `toml:"unique"`
	EffectKind         string `toml:"effect_kind"`
	EffectValue        int    `toml:"effect_value"`
}

type sitonesDocument struct {
	Sitones []Sitone `toml:"sitones"`
}

func (s *Store) ListSitones() []Sitone {
	if s == nil || len(s.sitones) == 0 {
		return nil
	}

	sitones := make([]Sitone, len(s.sitones))
	copy(sitones, s.sitones)
	return sitones
}

func (s *Store) GetSitone(id string) (Sitone, bool) {
	if s == nil {
		return Sitone{}, false
	}

	sitone, ok := s.sitonesByID[id]
	return sitone, ok
}

func validateSitones(path string, sitones []Sitone) ([]Sitone, map[string]Sitone, error) {
	var errs []error
	seen := make(map[string]struct{}, len(sitones))
	normalized := make([]Sitone, 0, len(sitones))

	for i, sitone := range sitones {
		sitone = normalizeSitone(sitone)
		location := fmt.Sprintf("%s: sitones[%d]", path, i)

		if sitone.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", location))
		} else if _, ok := seen[sitone.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate sitone id %q", path, sitone.ID))
		} else {
			seen[sitone.ID] = struct{}{}
		}
		if sitone.Name == "" {
			errs = append(errs, fmt.Errorf("%s.name is required", location))
		}
		if sitone.Type == "" {
			errs = append(errs, fmt.Errorf("%s.type is required", location))
		} else if _, ok := validSitoneTypes[sitone.Type]; !ok {
			errs = append(errs, fmt.Errorf("%s.type must be one of %s", location, sortedKeys(validSitoneTypes)))
		}
		if sitone.Rarity == "" {
			errs = append(errs, fmt.Errorf("%s.rarity is required", location))
		} else if _, ok := validRarities[sitone.Rarity]; !ok {
			errs = append(errs, fmt.Errorf("%s.rarity must be one of %s", location, sortedKeys(validRarities)))
		}
		if sitone.AbilityName == "" {
			errs = append(errs, fmt.Errorf("%s.ability_name is required", location))
		}
		if sitone.AbilityKind == "" {
			errs = append(errs, fmt.Errorf("%s.ability_kind is required", location))
		} else if _, ok := validSitoneAbilityKinds[sitone.AbilityKind]; !ok {
			errs = append(errs, fmt.Errorf("%s.ability_kind must be one of %s", location, sortedKeys(validSitoneAbilityKinds)))
		}
		if sitone.AbilityValue <= 0 {
			errs = append(errs, fmt.Errorf("%s.ability_value must be greater than 0", location))
		}
		if sitone.AbilityKind == SitoneAbilityEliminateWrongChoice {
			if sitone.AbilityCount <= 0 {
				errs = append(errs, fmt.Errorf("%s.ability_count must be greater than 0 for eliminate_wrong_choice", location))
			}
		} else if sitone.AbilityCount != 0 {
			errs = append(errs, fmt.Errorf("%s.ability_count must be 0 unless ability_kind is eliminate_wrong_choice", location))
		}
		if sitone.AbilityDescription == "" {
			errs = append(errs, fmt.Errorf("%s.ability_description is required", location))
		}
		if sitone.Attack < 0 {
			errs = append(errs, fmt.Errorf("%s.attack must be greater than or equal to 0", location))
		}
		if sitone.Defense < 0 {
			errs = append(errs, fmt.Errorf("%s.defense must be greater than or equal to 0", location))
		}
		if sitone.Repeatable == sitone.Unique {
			errs = append(errs, fmt.Errorf("%s: exactly one of repeatable and unique must be true", location))
		}
		if sitone.EffectKind != "" {
			if _, ok := validSitoneEffectKinds[sitone.EffectKind]; !ok {
				errs = append(errs, fmt.Errorf("%s.effect_kind must be one of %s", location, sortedKeys(validSitoneEffectKinds)))
			}
			if sitone.EffectValue <= 0 {
				errs = append(errs, fmt.Errorf("%s.effect_value must be greater than 0 when effect_kind is set", location))
			}
		} else if sitone.EffectValue != 0 {
			errs = append(errs, fmt.Errorf("%s.effect_value must be 0 unless effect_kind is set", location))
		}

		normalized = append(normalized, sitone)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].ID < normalized[j].ID
	})

	byID := make(map[string]Sitone, len(normalized))
	for _, sitone := range normalized {
		byID[sitone.ID] = sitone
	}

	return normalized, byID, nil
}

func normalizeSitone(sitone Sitone) Sitone {
	sitone.ID = strings.TrimSpace(sitone.ID)
	sitone.Name = strings.TrimSpace(sitone.Name)
	sitone.Type = strings.TrimSpace(sitone.Type)
	sitone.Rarity = strings.TrimSpace(sitone.Rarity)
	sitone.Style = strings.TrimSpace(sitone.Style)
	sitone.Description = strings.TrimSpace(sitone.Description)
	sitone.IconPath = strings.TrimSpace(sitone.IconPath)
	sitone.AbilityName = strings.TrimSpace(sitone.AbilityName)
	sitone.AbilityKind = strings.TrimSpace(sitone.AbilityKind)
	sitone.AbilityDescription = strings.TrimSpace(sitone.AbilityDescription)
	sitone.EffectKind = strings.TrimSpace(sitone.EffectKind)
	return sitone
}
