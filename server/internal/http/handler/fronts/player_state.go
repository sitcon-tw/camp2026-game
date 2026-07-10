package fronts

import (
	"context"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const fallbackFrontOpenPower = 100

const maxFrontSitoneInventory = 200

type frontSitoneInventory struct {
	Selected  []FrontSitoneResponse
	Available []FrontSitoneResponse
}

func withCurrentPlayerTeam(front mongomodel.Front, player mongomodel.Player) mongomodel.Front {
	teamID := strings.TrimSpace(player.TeamID)
	if teamID == "" {
		return front
	}
	if front.MapMode == content.FrontMapModeTerritoryGrid {
		// Territory sessions have a fixed roster and immutable base assignment.
		// In particular, team-010 remains a read-only observer.
		return front
	}

	teamIndex := frontTeamIndex(front.Teams, teamID)
	newTeam := false
	if teamIndex < 0 {
		front.Teams = append(front.Teams, mongomodel.FrontTeam{
			TeamID:         teamID,
			Name:           teamID,
			FrontOpenPower: fallbackFrontOpenPower,
		})
		teamIndex = len(front.Teams) - 1
		newTeam = true
	}
	if front.Teams[teamIndex].Name == "" {
		front.Teams[teamIndex].Name = teamID
	}
	if newTeam && front.Teams[teamIndex].FrontOpenPower == 0 {
		front.Teams[teamIndex].FrontOpenPower = fallbackFrontOpenPower
	}

	if !teamControlsAnyCell(front.Cells, teamID) {
		assignBaseCellToTeam(front.Cells, teamID)
	}
	syncTeamRanks(front.Teams, front.Cells)
	front.Leaderboard = deriveLeaderboard(front)
	return front
}

func (h *Handler) playerFrontSitones(ctx context.Context, player mongomodel.Player) (frontSitoneInventory, error) {
	defaults := normalizedSitoneIDs(player.DefaultSitoneIDs)
	owned := append([]string(nil), defaults...)
	if h.db != nil {
		var err error
		owned, err = h.playerOwnedSitoneIDs(ctx, player.ID)
		if err != nil {
			return frontSitoneInventory{}, err
		}
	}
	if h.db == nil && len(owned) == 0 && h.content != nil {
		for _, sitone := range h.content.ListSitones() {
			owned = append(owned, sitone.ID)
			if len(owned) >= 3 {
				break
			}
		}
	}
	ordered := prioritizedOwnedSitoneIDs(defaults, owned)
	selectedIDs := selectedOwnedSitoneIDs(defaults, owned)
	return frontSitoneInventory{
		Selected:  h.frontSitoneResponses(selectedIDs),
		Available: h.frontSitoneResponses(ordered),
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

func selectedOwnedSitoneIDs(defaults []string, owned []string) []string {
	ownedSet := make(map[string]struct{}, len(owned))
	for _, id := range normalizedSitoneIDs(owned) {
		ownedSet[id] = struct{}{}
	}
	out := make([]string, 0, len(defaults))
	for _, id := range normalizedSitoneIDs(defaults) {
		if _, ok := ownedSet[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func (h *Handler) playerOwnedSitoneIDs(ctx context.Context, playerID string) ([]string, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "sitone_id", Value: 1}}).SetLimit(maxFrontSitoneInventory),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var record mongomodel.PlayerSitone
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		if record.SitoneID != "" {
			ids = append(ids, record.SitoneID)
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return normalizedSitoneIDs(ids), nil
}

func (h *Handler) frontSitoneResponses(ids []string) []FrontSitoneResponse {
	out := make([]FrontSitoneResponse, 0, len(ids))
	for _, sitoneID := range ids {
		if h.content != nil {
			if _, ok := h.content.GetSitone(sitoneID); !ok {
				continue
			}
		}
		out = append(out, h.frontSitoneResponse(sitoneID))
	}
	return out
}

func (h *Handler) frontSitoneResponse(sitoneID string) FrontSitoneResponse {
	response := FrontSitoneResponse{
		SitoneID:  sitoneID,
		Name:      sitoneID,
		Available: true,
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
	return response
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

func teamControlsAnyCell(cells []mongomodel.FrontCell, teamID string) bool {
	for _, cell := range cells {
		if cell.OwnerTeamID == teamID {
			return true
		}
	}
	return false
}

func assignBaseCellToTeam(cells []mongomodel.FrontCell, teamID string) {
	baseIndexes := make([]int, 0, len(cells))
	for i, cell := range cells {
		if cell.Terrain == content.FrontTerrainBase || cell.Zone == content.FrontZoneBase {
			baseIndexes = append(baseIndexes, i)
		}
	}
	if len(baseIndexes) == 0 {
		return
	}
	sort.Ints(baseIndexes)
	cell := &cells[baseIndexes[stableTeamIndex(teamID, len(baseIndexes))]]
	cell.OwnerTeamID = teamID
	cell.Control = 100
	if cell.Defense < 20 {
		cell.Defense = 20
	}
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
