package matches

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

type battleEffects struct {
	MaterialDropBonusPercent int
	AnswerScoreBonusPercent  int
	OpenPowerBonusPercent    int
	EliminateChancePercent   int
	EliminateCount           int
	EliminateSourceNames     []string
}

func (h *Handler) snapshotMatchBattleEffects(ctx context.Context, match *mongomodel.Match) error {
	if match == nil {
		return nil
	}
	for index, player := range match.Players {
		effects, err := h.battleEffects(ctx, player.PlayerID, player.SitoneIDs)
		if err != nil {
			return err
		}
		match.Players[index].BattleEffects = battleEffectsSnapshot(effects)
	}
	return nil
}

func (h *Handler) matchPlayerBattleEffects(ctx context.Context, player mongomodel.MatchPlayer) (battleEffects, error) {
	if effects, ok := battleEffectsFromSnapshot(player.BattleEffects); ok {
		return effects, nil
	}
	return h.battleEffects(ctx, player.PlayerID, player.SitoneIDs)
}

func battleEffectsSnapshot(effects battleEffects) *mongomodel.MatchBattleEffects {
	return &mongomodel.MatchBattleEffects{
		MaterialDropBonusPercent: effects.MaterialDropBonusPercent,
		AnswerScoreBonusPercent:  effects.AnswerScoreBonusPercent,
		OpenPowerBonusPercent:    effects.OpenPowerBonusPercent,
		EliminateChancePercent:   effects.EliminateChancePercent,
		EliminateCount:           effects.EliminateCount,
		EliminateSourceNames:     cloneStrings(effects.EliminateSourceNames),
	}
}

func battleEffectsFromSnapshot(snapshot *mongomodel.MatchBattleEffects) (battleEffects, bool) {
	if snapshot == nil {
		return battleEffects{}, false
	}
	return battleEffects{
		MaterialDropBonusPercent: snapshot.MaterialDropBonusPercent,
		AnswerScoreBonusPercent:  snapshot.AnswerScoreBonusPercent,
		OpenPowerBonusPercent:    snapshot.OpenPowerBonusPercent,
		EliminateChancePercent:   snapshot.EliminateChancePercent,
		EliminateCount:           snapshot.EliminateCount,
		EliminateSourceNames:     cloneStrings(snapshot.EliminateSourceNames),
	}, true
}

func (h *Handler) battleEffects(ctx context.Context, playerID string, sitoneIDs []string) (battleEffects, error) {
	effects := battleEffects{}
	sitoneTypes := make(map[string]struct{}, len(sitoneIDs))

	for _, sitoneID := range sitoneIDs {
		sitone, ok := h.content.GetSitone(sitoneID)
		if !ok {
			continue
		}
		sitoneTypes[sitone.Type] = struct{}{}
		addSitoneAbility(&effects, sitone)
	}

	charmIDs, err := h.ownedCharmIDs(ctx, playerID)
	if err != nil {
		return battleEffects{}, err
	}
	for charmID := range charmIDs {
		addCharmAbility(&effects, charmID, sitoneTypes)
	}

	effects.EliminateSourceNames = uniqueStrings(effects.EliminateSourceNames)
	return effects, nil
}

func addSitoneAbility(effects *battleEffects, sitone content.Sitone) {
	switch sitone.AbilityKind {
	case content.SitoneAbilityMaterialDropRate:
		effects.MaterialDropBonusPercent += sitone.AbilityValue
	case content.SitoneAbilityAnswerScoreBonus:
		effects.AnswerScoreBonusPercent += sitone.AbilityValue
	case content.SitoneAbilityOpenPowerBonus:
		effects.OpenPowerBonusPercent += sitone.AbilityValue
	case content.SitoneAbilityEliminateWrongChoice:
		effects.EliminateChancePercent += sitone.AbilityValue
		if sitone.AbilityCount > effects.EliminateCount {
			effects.EliminateCount = sitone.AbilityCount
		}
		effects.EliminateSourceNames = append(effects.EliminateSourceNames, sitone.Name)
	}
}

func addCharmAbility(effects *battleEffects, charmID string, sitoneTypes map[string]struct{}) {
	switch charmID {
	case "item_charm_connection":
		if hasSitoneType(sitoneTypes, "exploration") {
			effects.MaterialDropBonusPercent += 15
		}
	case "item_charm_debug":
		if hasSitoneType(sitoneTypes, "engineering") {
			effects.AnswerScoreBonusPercent += 10
		}
	case "item_charm_all_nighter":
		if hasSitoneType(sitoneTypes, "inspiration") {
			effects.EliminateChancePercent += 20
			if effects.EliminateCount < 1 {
				effects.EliminateCount = 1
			}
			effects.EliminateSourceNames = append(effects.EliminateSourceNames, "熬夜有成御守")
		}
	case "item_charm_success":
		if hasSitoneType(sitoneTypes, "entertainment") {
			effects.OpenPowerBonusPercent += 20
		}
	case "item_charm_harmony":
		if hasSitoneType(sitoneTypes, "resonance") {
			effects.OpenPowerBonusPercent += 20
		}
	}
}

func hasSitoneType(sitoneTypes map[string]struct{}, sitoneType string) bool {
	_, ok := sitoneTypes[sitoneType]
	return ok
}

func (h *Handler) ownedCharmIDs(ctx context.Context, playerID string) (map[string]struct{}, error) {
	if h.db == nil {
		return nil, nil
	}

	cursor, err := h.db.Collection(mongomodel.PlayerItemsCollection).Find(
		ctx,
		bson.M{
			"player_id": playerID,
			"quantity":  bson.M{"$gt": 0},
			"item_id": bson.M{"$in": []string{
				"item_charm_connection",
				"item_charm_debug",
				"item_charm_all_nighter",
				"item_charm_success",
				"item_charm_harmony",
			}},
		},
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	out := make(map[string]struct{})
	for cursor.Next(ctx) {
		var record mongomodel.PlayerItem
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		out[record.ItemID] = struct{}{}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func applyPercentBonus(value int, bonusPercent int) int {
	if value <= 0 || bonusPercent <= 0 {
		return value
	}
	return value + value*bonusPercent/100
}

func deterministicPercent(parts ...string) int {
	return int(deterministicUint64(parts...) % 100)
}

func deterministicUint64(parts ...string) uint64 {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return binary.BigEndian.Uint64(sum[:8])
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
