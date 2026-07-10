package fronts

import (
	"sort"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var seedFrontUpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func cloneFront(front mongomodel.Front) mongomodel.Front {
	front.Teams = append([]mongomodel.FrontTeam(nil), front.Teams...)
	front.ActiveEvents = append([]mongomodel.FrontMapEvent(nil), front.ActiveEvents...)
	front.Leaderboard = append([]mongomodel.FrontLeaderboardEntry(nil), front.Leaderboard...)
	front.Garrisons = append([]mongomodel.FrontGarrison(nil), front.Garrisons...)
	for i := range front.Garrisons {
		front.Garrisons[i].SitoneIDs = append([]string(nil), front.Garrisons[i].SitoneIDs...)
	}
	front.TradeRoutes = append([]mongomodel.FrontTradeRoute(nil), front.TradeRoutes...)
	if front.LastCommand != nil {
		command := *front.LastCommand
		command.AffectedCells = append([]mongomodel.FrontCoordinate(nil), command.AffectedCells...)
		command.SitoneIDs = append([]string(nil), command.SitoneIDs...)
		command.CapturedGarrisons = cloneCapturedGarrisons(command.CapturedGarrisons)
		command.Payload = clonePayload(command.Payload)
		front.LastCommand = &command
	}
	if front.Territory != nil {
		territory := *front.Territory
		territory.Rows = append([]mongomodel.FrontTerritoryRow(nil), territory.Rows...)
		for i := range territory.Rows {
			territory.Rows[i].Runs = append([]mongomodel.FrontTerritoryRun(nil), territory.Rows[i].Runs...)
		}
		territory.Bases = append([]mongomodel.FrontTerritoryBase(nil), territory.Bases...)
		territory.Landmarks = append([]mongomodel.FrontMapLandmark(nil), territory.Landmarks...)
		front.Territory = &territory
	}
	return front
}

func cloneCapturedGarrisons(garrisons []mongomodel.FrontCapturedGarrison) []mongomodel.FrontCapturedGarrison {
	out := append([]mongomodel.FrontCapturedGarrison(nil), garrisons...)
	for i := range out {
		out[i].SitoneIDs = append([]string(nil), out[i].SitoneIDs...)
	}
	return out
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	out := make(map[string]any, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func rankedLeaderboard(front mongomodel.Front) []mongomodel.FrontLeaderboardEntry {
	entries := append([]mongomodel.FrontLeaderboardEntry(nil), front.Leaderboard...)
	if len(entries) == 0 {
		entries = deriveLeaderboard(front)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if entries[i].ControlledCells != entries[j].ControlledCells {
			return entries[i].ControlledCells > entries[j].ControlledCells
		}
		if entries[i].TeamName != entries[j].TeamName {
			return entries[i].TeamName < entries[j].TeamName
		}
		return entries[i].TeamID < entries[j].TeamID
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}

func deriveLeaderboard(front mongomodel.Front) []mongomodel.FrontLeaderboardEntry {
	controlled := make(map[string]int)
	if matrix, err := decodeTerritoryRows(front.Territory); err == nil {
		for y := range matrix {
			for x := range matrix[y] {
				if matrix[y][x].Owner != "" {
					controlled[matrix[y][x].Owner]++
				}
			}
		}
	}
	entries := make([]mongomodel.FrontLeaderboardEntry, 0, len(front.Teams))
	for _, team := range front.Teams {
		entries = append(entries, mongomodel.FrontLeaderboardEntry{
			TeamID:             team.TeamID,
			TeamName:           team.Name,
			Score:              team.Score + controlled[team.TeamID]*10,
			ControlledCells:    controlled[team.TeamID],
			RescuedSitones:     team.RescuedSitones,
			RepairedEvents:     team.RepairedEvents,
			CollaborationScore: team.CollaborationScore,
			TradeScore:         team.TradeScore,
			PreviousRank:       team.PreviousRank,
		})
	}
	return entries
}
