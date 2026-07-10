package fronts

import (
	"context"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	maxFrontSitoneInventory = 200
	maxFrontSitoneLoadout   = 5
)

type frontSitoneInventory struct {
	Selected  []FrontSitoneResponse
	Available []FrontSitoneResponse
}

func withCurrentPlayerTeam(front mongomodel.Front, player mongomodel.Player) mongomodel.Front {
	_ = player
	return front
}

func (h *Handler) playerFrontSitones(ctx context.Context, player mongomodel.Player) (frontSitoneInventory, error) {
	defaults := normalizedSitoneLoadout(player.DefaultSitoneIDs)
	ownedCounts := sitoneCounts(defaults)
	if h.db != nil {
		var err error
		ownedCounts, err = h.playerOwnedSitoneCounts(ctx, player.ID)
		if err != nil {
			return frontSitoneInventory{}, err
		}
	}
	if h.db == nil && len(ownedCounts) == 0 && h.content != nil {
		for _, sitone := range h.content.ListSitones() {
			ownedCounts[sitone.ID] = 1
			if len(ownedCounts) >= 3 {
				break
			}
		}
	}
	owned := sortedSitoneIDs(ownedCounts)
	ordered := prioritizedOwnedSitoneIDs(defaults, owned)
	selectedIDs := selectedOwnedSitoneIDs(defaults, ownedCounts)
	if len(selectedIDs) == 0 && len(ordered) > 0 {
		selectedIDs = []string{ordered[0]}
	}
	return frontSitoneInventory{
		Selected:  h.frontSitoneResponses(selectedIDs, ownedCounts),
		Available: h.frontSitoneResponses(ordered, ownedCounts),
	}, nil
}

func prioritizedOwnedSitoneIDs(defaults []string, owned []string) []string {
	owned = normalizedSitoneIDs(owned)
	ownedSet := make(map[string]struct{}, len(owned))
	for _, id := range owned {
		ownedSet[id] = struct{}{}
	}
	out := make([]string, 0, len(owned))
	seen := make(map[string]struct{}, len(owned))
	for _, id := range normalizedSitoneIDs(defaults) {
		if _, ok := ownedSet[id]; !ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range owned {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, id)
	}
	return out
}

func selectedOwnedSitoneIDs(defaults []string, ownedCounts map[string]int) []string {
	out := make([]string, 0, len(defaults))
	used := make(map[string]int)
	for _, id := range normalizedSitoneLoadout(defaults) {
		if len(out) >= maxFrontSitoneLoadout {
			break
		}
		used[id]++
		if used[id] <= ownedCounts[id] {
			out = append(out, id)
		}
	}
	return out
}

func (h *Handler) playerOwnedSitoneCounts(ctx context.Context, playerID string) (map[string]int, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "sitone_id", Value: 1}}).SetLimit(maxFrontSitoneInventory),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var record mongomodel.PlayerSitone
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		if record.SitoneID != "" && record.Quantity > 0 {
			counts[strings.TrimSpace(record.SitoneID)] += record.Quantity
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func (h *Handler) frontSitoneResponses(ids []string, ownedCounts map[string]int) []FrontSitoneResponse {
	out := make([]FrontSitoneResponse, 0, len(ids))
	for _, sitoneID := range ids {
		if h.content != nil {
			if _, ok := h.content.GetSitone(sitoneID); !ok {
				continue
			}
		}
		out = append(out, h.frontSitoneResponse(sitoneID, ownedCounts[sitoneID]))
	}
	return out
}

func (h *Handler) frontSitoneResponse(sitoneID string, ownedQuantity int) FrontSitoneResponse {
	response := FrontSitoneResponse{
		SitoneID:      sitoneID,
		Name:          sitoneID,
		OwnedQuantity: ownedQuantity,
		Available:     ownedQuantity > 0,
	}
	if h.content == nil {
		return response
	}
	sitone, ok := h.content.GetSitone(sitoneID)
	if !ok {
		return response
	}
	response.Name = sitone.Name
	response.Type = sitone.Type
	response.IconPath = sitone.IconPath
	response.AbilityName = sitone.AbilityName
	response.AbilityValue = sitone.AbilityValue
	response.FrontAffinityCommands = frontAffinityCommands(sitone.Type)
	return response
}

func normalizedSitoneLoadout(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func sitoneCounts(ids []string) map[string]int {
	counts := make(map[string]int)
	for _, id := range normalizedSitoneLoadout(ids) {
		counts[id]++
	}
	return counts
}

func sortedSitoneIDs(counts map[string]int) []string {
	ids := make([]string, 0, len(counts))
	for id, quantity := range counts {
		if id != "" && quantity > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func normalizedSitoneIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func frontTeamIndex(teams []mongomodel.FrontTeam, teamID string) int {
	for i, team := range teams {
		if team.TeamID == teamID {
			return i
		}
	}
	return -1
}

func stableTeamIndex(teamID string, length int) int {
	if length <= 0 {
		return 0
	}
	sum := 0
	for _, value := range teamID {
		sum += int(value)
	}
	if sum < 0 {
		sum = -sum
	}
	return sum % length
}
