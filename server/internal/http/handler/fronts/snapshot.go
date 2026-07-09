package fronts

import (
	"sort"
	"strings"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const defaultFrontID = "front_main"

var seedFrontUpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

var seedFronts = map[string]mongomodel.Front{
	defaultFrontID: {
		ID:       defaultFrontID,
		MapID:    defaultFrontID,
		Name:     "Camp 2026 Main Front",
		Status:   mongomodel.FrontStatusOpenPlay,
		Current:  true,
		Revision: 1,
		Cells: []mongomodel.FrontCell{
			{ID: "base", Name: "Base", X: 0, Y: 0, Terrain: "base", Zone: "base", OwnerTeamID: "team-blue", Control: 100, Defense: 80, NeighborIDs: []string{"gate"}},
			{ID: "gate", Name: "Gate", X: 1, Y: 0, Terrain: "system", Zone: "system", OwnerTeamID: "team-blue", Control: 70, Defense: 50, NeighborIDs: []string{"base", "commons"}},
			{ID: "commons", Name: "Commons", X: 2, Y: 0, Terrain: "center", Zone: "frontier", Control: 40, Defense: 30, Resource: 30, NeighborIDs: []string{"gate"}},
		},
		Teams: []mongomodel.FrontTeam{
			{TeamID: "team-blue", Name: "Blue", Color: "#2563eb", Score: 120, FrontOpenPower: 100, ControlledCells: 2},
			{TeamID: "team-green", Name: "Green", Color: "#16a34a", Score: 95, FrontOpenPower: 100},
			{TeamID: "team-gold", Name: "Gold", Color: "#ca8a04", Score: 80, FrontOpenPower: 100},
		},
		CreatedAt: seedFrontUpdatedAt,
		UpdatedAt: seedFrontUpdatedAt,
	},
}

var seedTeams = []mongomodel.FrontTeam{
	{TeamID: "team-blue", Name: "Blue", Color: "#2563eb"},
	{TeamID: "team-green", Name: "Green", Color: "#16a34a"},
	{TeamID: "team-gold", Name: "Gold", Color: "#ca8a04"},
}

func (h *Handler) fallbackCurrentFront() mongomodel.Front {
	if h.content != nil {
		for _, template := range h.content.ListFrontMaps() {
			if template.Enabled {
				return frontFromTemplate(template)
			}
		}
	}
	return cloneFront(seedFronts[defaultFrontID])
}

func (h *Handler) fallbackFrontByID(frontID string) (mongomodel.Front, bool) {
	if h.content != nil {
		template, ok := h.content.GetFrontMap(frontID)
		if ok && template.Enabled {
			return frontFromTemplate(template), true
		}
	}
	front, ok := seedFronts[frontID]
	if !ok {
		return mongomodel.Front{}, false
	}
	return cloneFront(front), true
}

func frontFromTemplate(template content.FrontMapTemplate) mongomodel.Front {
	teams := templateTeams(template.InitialFrontOpenPower)
	cells := templateCells(template, teams)
	events := templateEvents(template)
	attachEventIDs(cells, events)
	syncTeamRanks(teams, cells)

	return mongomodel.Front{
		ID:           template.ID,
		MapID:        template.ID,
		Name:         template.Name,
		Status:       mongomodel.FrontStatusOpenPlay,
		Current:      true,
		Revision:     1,
		Cells:        cells,
		Teams:        teams,
		ActiveEvents: events,
		CreatedAt:    seedFrontUpdatedAt,
		UpdatedAt:    seedFrontUpdatedAt,
	}
}

func templateTeams(frontOpenPower int) []mongomodel.FrontTeam {
	teams := append([]mongomodel.FrontTeam(nil), seedTeams...)
	if frontOpenPower <= 0 {
		frontOpenPower = 100
	}
	for i := range teams {
		teams[i].FrontOpenPower = frontOpenPower
	}
	return teams
}

func templateCells(template content.FrontMapTemplate, teams []mongomodel.FrontTeam) []mongomodel.FrontCell {
	cells := make([]mongomodel.FrontCell, 0, len(template.Cells))
	baseIndex := 0
	for _, cell := range template.Cells {
		ownerTeamID := ""
		control := cell.InitialControl
		if cell.Terrain == content.FrontTerrainBase && len(teams) > 0 {
			ownerTeamID = teams[baseIndex%len(teams)].TeamID
			baseIndex++
			if control == 0 {
				control = 100
			}
		}
		cells = append(cells, mongomodel.FrontCell{
			ID:             cell.ID,
			Name:           titleFromID(cell.ID),
			X:              cell.X,
			Y:              cell.Y,
			Terrain:        cell.Terrain,
			Zone:           cell.Zone,
			OwnerTeamID:    ownerTeamID,
			Control:        control,
			Defense:        cell.InitialDefense,
			Resource:       cell.InitialResource,
			PressureByTeam: map[string]int{},
			NeighborIDs:    append([]string(nil), cell.Neighbors...),
		})
	}
	return cells
}

func templateEvents(template content.FrontMapTemplate) []mongomodel.FrontMapEvent {
	events := make([]mongomodel.FrontMapEvent, 0, len(template.Events))
	for _, event := range template.Events {
		events = append(events, mongomodel.FrontMapEvent{
			ID:                event.ID,
			CellID:            event.CellID,
			Kind:              event.Kind,
			Title:             event.Title,
			StartsAtTick:      event.StartsAtTick,
			ExpiresAtTick:     event.StartsAtTick + event.ExpiresAfterTicks,
			ExpiresAfterTicks: event.ExpiresAfterTicks,
			Severity:          event.Severity,
		})
	}
	return events
}

func attachEventIDs(cells []mongomodel.FrontCell, events []mongomodel.FrontMapEvent) {
	eventByCell := make(map[string]string, len(events))
	for _, event := range events {
		if event.CellID != "" {
			eventByCell[event.CellID] = event.ID
		}
	}
	for i := range cells {
		cells[i].EventID = eventByCell[cells[i].ID]
	}
}

func syncTeamRanks(teams []mongomodel.FrontTeam, cells []mongomodel.FrontCell) {
	controlled := make(map[string]int)
	resources := make(map[string]int)
	for _, cell := range cells {
		if cell.OwnerTeamID == "" {
			continue
		}
		controlled[cell.OwnerTeamID]++
		resources[cell.OwnerTeamID] += cell.Resource
	}
	for i := range teams {
		teams[i].ControlledCells = controlled[teams[i].TeamID]
		if teams[i].Score == 0 {
			teams[i].Score = teams[i].FrontOpenPower + controlled[teams[i].TeamID]*10 + resources[teams[i].TeamID]
		}
	}

	sort.SliceStable(teams, func(i, j int) bool {
		if teams[i].Score != teams[j].Score {
			return teams[i].Score > teams[j].Score
		}
		return teams[i].TeamID < teams[j].TeamID
	})
	for i := range teams {
		teams[i].Rank = i + 1
	}
	sort.SliceStable(teams, func(i, j int) bool {
		return teams[i].TeamID < teams[j].TeamID
	})
}

func titleFromID(id string) string {
	parts := strings.Split(id, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func cloneFront(front mongomodel.Front) mongomodel.Front {
	if front.Cells != nil {
		front.Cells = append([]mongomodel.FrontCell(nil), front.Cells...)
		for i := range front.Cells {
			front.Cells[i].NeighborIDs = append([]string(nil), front.Cells[i].NeighborIDs...)
			front.Cells[i].PressureByTeam = cloneIntMap(front.Cells[i].PressureByTeam)
		}
	}
	if front.Teams != nil {
		front.Teams = append([]mongomodel.FrontTeam(nil), front.Teams...)
	}
	if front.Zones != nil {
		front.Zones = append([]mongomodel.FrontZone(nil), front.Zones...)
	}
	if front.ActiveEvents != nil {
		front.ActiveEvents = append([]mongomodel.FrontMapEvent(nil), front.ActiveEvents...)
	}
	if front.Leaderboard != nil {
		front.Leaderboard = append([]mongomodel.FrontLeaderboardEntry(nil), front.Leaderboard...)
	}
	if front.LastCommand != nil {
		command := *front.LastCommand
		command.Payload = clonePayload(command.Payload)
		front.LastCommand = &command
	}
	return front
}

func cloneIntMap(values map[string]int) map[string]int {
	if values == nil {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		out[key] = value
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
	cellCounts := make(map[string]int)
	resources := make(map[string]int)
	for _, cell := range front.Cells {
		if cell.OwnerTeamID == "" {
			continue
		}
		cellCounts[cell.OwnerTeamID]++
		resources[cell.OwnerTeamID] += cell.Resource
	}

	entries := make([]mongomodel.FrontLeaderboardEntry, 0, len(front.Teams))
	seenTeams := make(map[string]struct{}, len(front.Teams))
	for _, team := range front.Teams {
		if team.TeamID == "" {
			continue
		}
		seenTeams[team.TeamID] = struct{}{}
		score := team.Score
		if score == 0 {
			score = team.FrontOpenPower + cellCounts[team.TeamID]*10 + resources[team.TeamID]
		}
		entries = append(entries, mongomodel.FrontLeaderboardEntry{
			TeamID:             team.TeamID,
			TeamName:           team.Name,
			Score:              score,
			ControlledCells:    cellCounts[team.TeamID],
			RescuedSitones:     team.RescuedSitones,
			RepairedEvents:     team.RepairedEvents,
			CollaborationScore: team.CollaborationScore,
			PreviousRank:       team.PreviousRank,
		})
	}

	for teamID, count := range cellCounts {
		if _, ok := seenTeams[teamID]; ok {
			continue
		}
		entries = append(entries, mongomodel.FrontLeaderboardEntry{
			TeamID:          teamID,
			TeamName:        teamID,
			Score:           resources[teamID] + count*10,
			ControlledCells: count,
		})
	}
	return entries
}
