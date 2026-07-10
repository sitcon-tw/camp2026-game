package fronts

import (
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func frontSummaryResponse(front mongomodel.Front) *FrontSessionSummaryResponse {
	return &FrontSessionSummaryResponse{
		ID:       front.ID,
		MapID:    front.MapID,
		MapMode:  front.MapMode,
		Status:   front.Status,
		Tick:     front.Tick,
		Revision: front.Revision,
	}
}

func detailResponse(front mongomodel.Front, currentTeamID string, sitones []FrontSitoneResponse) DetailResponse {
	leaderboard := rankedLeaderboard(front)
	myTeamRank := 0
	for _, entry := range leaderboard {
		if entry.TeamID == currentTeamID {
			myTeamRank = entry.Rank
			break
		}
	}

	response := DetailResponse{
		Front:                     *frontSummaryResponse(front),
		MapMode:                   front.MapMode,
		CanPlay:                   front.MapMode != contentFrontMapModeTerritoryGrid || isParticipatingTerritoryTeam(currentTeamID),
		ServerTime:                front.UpdatedAt,
		Cells:                     cellResponses(front.Cells),
		Teams:                     teamResponses(front.Teams),
		ActiveEvents:              eventResponses(front.ActiveEvents),
		Leaderboard:               leaderboardResponse(leaderboard, currentTeamID),
		MyTeamID:                  currentTeamID,
		MyTeamRank:                myTeamRank,
		Cooldowns:                 map[string]int64{},
		SelectedSitones:           sitones,
		AvailableSitones:          sitones,
		CurrentTeamFrontOpenPower: teamFrontOpenPower(front.Teams, currentTeamID),
		SupportTokens:             1,
		AvailableCommands:         commandOptionResponses(front, currentTeamID),
	}
	if front.Territory != nil {
		response.Grid = territoryGridResponse(front.Territory)
		response.TerritoryRows = territoryRowResponses(front.Territory.Rows)
		response.Bases = territoryBaseResponses(front.Territory.Bases)
		response.Landmarks = territoryLandmarkResponses(front.Territory.Landmarks)
	}
	return response
}

func cellResponses(cells []mongomodel.FrontCell) []FrontCellResponse {
	out := make([]FrontCellResponse, 0, len(cells))
	for _, cell := range cells {
		out = append(out, FrontCellResponse{
			ID:             cell.ID,
			Name:           cell.Name,
			X:              cell.X,
			Y:              cell.Y,
			Terrain:        cell.Terrain,
			Zone:           cell.Zone,
			OwnerTeamID:    cell.OwnerTeamID,
			Control:        cell.Control,
			Defense:        cell.Defense,
			Resource:       cell.Resource,
			PressureByTeam: cloneIntMapOrEmpty(cell.PressureByTeam),
			NeighborIDs:    append([]string(nil), cell.NeighborIDs...),
			EventID:        cell.EventID,
			LockedUntil:    timePtrIfSet(cell.LockedUntil),
		})
	}
	return out
}

func teamResponses(teams []mongomodel.FrontTeam) []FrontTeamResponse {
	out := make([]FrontTeamResponse, 0, len(teams))
	for _, team := range teams {
		out = append(out, FrontTeamResponse{
			TeamID:             team.TeamID,
			Name:               team.Name,
			Color:              team.Color,
			Score:              team.Score,
			Rank:               team.Rank,
			PreviousRank:       team.PreviousRank,
			FrontOpenPower:     team.FrontOpenPower,
			ControlledCells:    team.ControlledCells,
			RescuedSitones:     team.RescuedSitones,
			RepairedEvents:     team.RepairedEvents,
			CollaborationScore: team.CollaborationScore,
			LastCommandAt:      timePtrIfSet(team.LastCommandAt),
		})
	}
	return out
}

func eventResponses(events []mongomodel.FrontMapEvent) []FrontMapEventResponse {
	out := make([]FrontMapEventResponse, 0, len(events))
	for _, event := range events {
		out = append(out, FrontMapEventResponse{
			ID:                event.ID,
			CellID:            event.CellID,
			Kind:              event.Kind,
			Title:             event.Title,
			StartsAtTick:      event.StartsAtTick,
			ExpiresAtTick:     event.ExpiresAtTick,
			ExpiresAfterTicks: event.ExpiresAfterTicks,
			Severity:          event.Severity,
		})
	}
	return out
}

func leaderboardResponse(entries []mongomodel.FrontLeaderboardEntry, currentTeamID string) []FrontLeaderboardEntryResponse {
	out := make([]FrontLeaderboardEntryResponse, 0, len(entries))
	for _, entry := range entries {
		out = append(out, FrontLeaderboardEntryResponse{
			TeamID:             entry.TeamID,
			TeamName:           entry.TeamName,
			Rank:               entry.Rank,
			PreviousRank:       entry.PreviousRank,
			Score:              entry.Score,
			ControlledCells:    entry.ControlledCells,
			RescuedSitones:     entry.RescuedSitones,
			RepairedEvents:     entry.RepairedEvents,
			CollaborationScore: entry.CollaborationScore,
			Current:            entry.TeamID != "" && entry.TeamID == currentTeamID,
		})
	}
	return out
}

func commandOptionResponses(front mongomodel.Front, currentTeamID string) []FrontCommandOptionResponse {
	if front.MapMode == contentFrontMapModeTerritoryGrid {
		return territoryCommandOptionResponses(front, currentTeamID)
	}
	out := make([]FrontCommandOptionResponse, 0, len(front.Cells)*4)
	frontOpenPower := teamFrontOpenPower(front.Teams, currentTeamID)
	eventByCell := make(map[string]mongomodel.FrontMapEvent, len(front.ActiveEvents))
	for _, event := range front.ActiveEvents {
		if event.CellID != "" {
			eventByCell[event.CellID] = event
		}
	}

	for _, cell := range front.Cells {
		sourceID := firstOwnedNeighborCellID(front.Cells, cell.ID, currentTeamID)
		if sourceID == "" {
			out = append(out,
				FrontCommandOptionResponse{Kind: "expand", Label: "擴張", Enabled: false, Cost: 10, ToCellID: cell.ID, Reason: "需要相鄰的己方節點"},
				FrontCommandOptionResponse{Kind: "attack", Label: "攻擊", Enabled: false, Cost: 15, ToCellID: cell.ID, Reason: "需要相鄰的己方節點"},
				FrontCommandOptionResponse{Kind: "reinforce", Label: "防守", Enabled: false, Cost: 8, ToCellID: cell.ID, Reason: "需要相鄰的己方節點"},
				FrontCommandOptionResponse{Kind: "scout", Label: "偵查", Enabled: false, Cost: 4, ToCellID: cell.ID, Reason: "需要相鄰的己方節點"},
			)
			continue
		}

		out = append(out,
			commandOptionForCell("expand", "擴張", 10, sourceID, cell, cell.OwnerTeamID == "", frontOpenPower),
			commandOptionForCell("attack", "攻擊", 15, sourceID, cell, cell.OwnerTeamID != "" && cell.OwnerTeamID != currentTeamID, frontOpenPower),
			commandOptionForCell("reinforce", "防守", 8, sourceID, cell, cell.OwnerTeamID == currentTeamID && (cell.Control < 100 || cell.Defense < territoryMaxDefense), frontOpenPower),
			commandOptionForCell("scout", "偵查", 4, sourceID, cell, true, frontOpenPower),
		)
		if event, ok := eventByCell[cell.ID]; ok {
			switch event.Kind {
			case "repair":
				out = append(out, commandOptionForCell("repair", "修復", 12, sourceID, cell, true, frontOpenPower))
			case "rescue":
				out = append(out, commandOptionForCell("rescue", "救援", 10, sourceID, cell, true, frontOpenPower))
			case "challenge":
				out = append(out, FrontCommandOptionResponse{Kind: "answer_challenge", Label: "挑戰", Enabled: true, Cost: 0, FromCellID: sourceID, ToCellID: cell.ID})
			case "resource":
				out = append(out, FrontCommandOptionResponse{Kind: "support", Label: "支援", Enabled: true, Cost: 0, FromCellID: sourceID, ToCellID: cell.ID})
			}
		}
	}
	return out
}

func commandOptionForCell(kind string, label string, cost int, sourceID string, cell mongomodel.FrontCell, targetEnabled bool, frontOpenPower int) FrontCommandOptionResponse {
	enabled := targetEnabled && frontOpenPower >= cost
	option := FrontCommandOptionResponse{
		Kind:       kind,
		Label:      label,
		Enabled:    enabled,
		Cost:       cost,
		FromCellID: sourceID,
		ToCellID:   cell.ID,
	}
	if !targetEnabled {
		option.Reason = "目標類型不符合命令"
	} else if !enabled {
		option.Reason = "前線開源力不足"
	}
	return option
}

func firstOwnedNeighborCellID(cells []mongomodel.FrontCell, cellID string, teamID string) string {
	if teamID == "" {
		return ""
	}
	cellByID := make(map[string]mongomodel.FrontCell, len(cells))
	for _, cell := range cells {
		cellByID[cell.ID] = cell
	}
	cell, ok := cellByID[cellID]
	if !ok {
		return ""
	}
	for _, neighborID := range cell.NeighborIDs {
		neighbor, ok := cellByID[neighborID]
		if ok && neighbor.OwnerTeamID == teamID {
			return neighbor.ID
		}
	}
	return ""
}

func teamFrontOpenPower(teams []mongomodel.FrontTeam, teamID string) int {
	for _, team := range teams {
		if team.TeamID == teamID {
			return team.FrontOpenPower
		}
	}
	return 0
}

func commandResponse(command mongomodel.FrontCommand) FrontCommandResponse {
	return FrontCommandResponse{
		CommandID:        command.ID,
		ClientCommandID:  command.ClientCommandID,
		Kind:             command.Kind,
		Type:             command.Type,
		PlayerID:         command.PlayerID,
		TeamID:           command.TeamID,
		FromCellID:       command.FromCellID,
		ToCellID:         command.ToCellID,
		TargetX:          command.TargetX,
		TargetY:          command.TargetY,
		ExpectedRevision: command.ExpectedRevision,
		AffectedCells:    coordinateResponses(command.AffectedCells),
		SitoneID:         command.SitoneID,
		Payload:          clonePayload(command.Payload),
		Accepted:         command.Accepted,
		Applied:          command.Applied,
		RejectReason:     command.RejectReason,
		CreatedAt:        command.CreatedAt,
	}
}

func commandSummaryResponse(command *mongomodel.FrontCommandSummary) *FrontCommandResponse {
	if command == nil {
		return nil
	}
	return &FrontCommandResponse{
		CommandID:        command.ID,
		ClientCommandID:  command.ClientCommandID,
		Kind:             command.Kind,
		Type:             command.Type,
		PlayerID:         command.PlayerID,
		TeamID:           command.TeamID,
		FromCellID:       command.FromCellID,
		ToCellID:         command.ToCellID,
		TargetX:          command.TargetX,
		TargetY:          command.TargetY,
		ExpectedRevision: command.ExpectedRevision,
		AffectedCells:    coordinateResponses(command.AffectedCells),
		SitoneID:         command.SitoneID,
		Payload:          clonePayload(command.Payload),
		Accepted:         command.Accepted,
		Applied:          command.Applied,
		RejectReason:     command.RejectReason,
		CreatedAt:        command.CreatedAt,
	}
}

func territoryGridResponse(territory *mongomodel.FrontTerritory) *FrontTerritoryGridResponse {
	if territory == nil {
		return nil
	}
	return &FrontTerritoryGridResponse{
		Width:        territory.Width,
		Height:       territory.Height,
		Connectivity: territory.Connectivity,
		Origin:       FrontTerritoryOriginResponse{X: territory.OriginX, Y: territory.OriginY},
		Background: FrontTerritoryBackgroundResponse{
			Src:    territory.BackgroundSrc,
			Width:  territory.BackgroundWidth,
			Height: territory.BackgroundHeight,
		},
	}
}

func territoryRowResponses(rows []mongomodel.FrontTerritoryRow) []FrontTerritoryRowResponse {
	out := make([]FrontTerritoryRowResponse, 0, len(rows))
	for _, row := range rows {
		runs := make([]FrontTerritoryRunResponse, 0, len(row.Runs))
		for _, run := range row.Runs {
			runs = append(runs, FrontTerritoryRunResponse{X: run.X, Length: run.Length, OwnerTeamID: run.OwnerTeamID, Defense: run.Defense})
		}
		out = append(out, FrontTerritoryRowResponse{Y: row.Y, Runs: runs})
	}
	return out
}

func territoryBaseResponses(bases []mongomodel.FrontTerritoryBase) []FrontTerritoryBaseResponse {
	out := make([]FrontTerritoryBaseResponse, 0, len(bases))
	for _, base := range bases {
		out = append(out, FrontTerritoryBaseResponse{TeamID: base.TeamID, X: base.X, Y: base.Y, CoreDefense: base.CoreDefense, InitialRadius: base.InitialRadius})
	}
	return out
}

func territoryLandmarkResponses(landmarks []mongomodel.FrontMapLandmark) []FrontMapLandmarkResponse {
	out := make([]FrontMapLandmarkResponse, 0, len(landmarks))
	for _, landmark := range landmarks {
		out = append(out, FrontMapLandmarkResponse{ID: landmark.ID, Kind: landmark.Kind, X: landmark.X, Y: landmark.Y, Label: landmark.Label, InitialDefense: landmark.InitialDefense, InitialResource: landmark.InitialResource})
	}
	return out
}

func coordinateResponses(coordinates []mongomodel.FrontCoordinate) []FrontCoordinateResponse {
	out := make([]FrontCoordinateResponse, 0, len(coordinates))
	for _, coordinate := range coordinates {
		out = append(out, FrontCoordinateResponse{X: coordinate.X, Y: coordinate.Y})
	}
	return out
}

func cloneIntMapOrEmpty(values map[string]int) map[string]int {
	if len(values) == 0 {
		return map[string]int{}
	}
	return cloneIntMap(values)
}

func timePtrIfSet(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
