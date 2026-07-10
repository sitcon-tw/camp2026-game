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

func detailResponse(front mongomodel.Front, currentTeamID string, sitones frontSitoneInventory, playerOpenPower int) DetailResponse {
	leaderboard := rankedLeaderboard(front)
	myTeamRank := 0
	for _, entry := range leaderboard {
		if entry.TeamID == currentTeamID {
			myTeamRank = entry.Rank
			break
		}
	}

	response := DetailResponse{
		Front:                  *frontSummaryResponse(front),
		MapMode:                front.MapMode,
		CanPlay:                isParticipatingTerritoryTeam(currentTeamID),
		ServerTime:             front.UpdatedAt,
		Teams:                  teamResponses(front.Teams),
		ActiveEvents:           eventResponses(front.ActiveEvents),
		Leaderboard:            leaderboardResponse(leaderboard, currentTeamID),
		MyTeamID:               currentTeamID,
		MyTeamRank:             myTeamRank,
		SelectedSitones:        sitones.Selected,
		AvailableSitones:       sitones.Available,
		CurrentPlayerOpenPower: playerOpenPower,
		FrontPowerSource:       "player_ledger",
		AvailableCommands:      commandOptionResponses(front, currentTeamID, playerOpenPower),
	}
	if front.Territory != nil {
		response.Grid = territoryGridResponse(front.Territory)
		response.TerritoryRows = territoryRowResponses(front.Territory.Rows)
		response.Bases = territoryBaseResponses(front.Territory.Bases)
		response.Landmarks = territoryLandmarkResponses(front.Territory.Landmarks)
	}
	return response
}

func teamResponses(teams []mongomodel.FrontTeam) []FrontTeamResponse {
	out := make([]FrontTeamResponse, 0, len(teams))
	for _, team := range teams {
		out = append(out, FrontTeamResponse{
			TeamID:                  team.TeamID,
			Name:                    team.Name,
			Color:                   team.Color,
			Score:                   team.Score,
			Rank:                    team.Rank,
			PreviousRank:            team.PreviousRank,
			ControlledCells:         team.ControlledCells,
			MaxControlledCells:      team.MaxControlledCells,
			SitoneMilestonesReached: team.SitoneMilestonesReached,
			NextSitoneMilestone:     team.NextSitoneMilestone,
			RescuedSitones:          team.RescuedSitones,
			RepairedEvents:          team.RepairedEvents,
			CollaborationScore:      team.CollaborationScore,
			LastCommandAt:           timePtrIfSet(team.LastCommandAt),
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

func commandOptionResponses(front mongomodel.Front, currentTeamID string, playerOpenPower int) []FrontCommandOptionResponse {
	return territoryCommandOptionResponses(front, currentTeamID, playerOpenPower)
}

func commandResponse(command mongomodel.FrontCommand) FrontCommandResponse {
	return FrontCommandResponse{
		CommandID:            command.ID,
		ClientCommandID:      command.ClientCommandID,
		Kind:                 command.Kind,
		Type:                 command.Type,
		PlayerID:             command.PlayerID,
		TeamID:               command.TeamID,
		TargetX:              command.TargetX,
		TargetY:              command.TargetY,
		ExpectedRevision:     command.ExpectedRevision,
		AffectedCells:        coordinateResponses(command.AffectedCells),
		CapturedCellCount:    command.CapturedCellCount,
		EnclosedCellCount:    command.EnclosedCellCount,
		ScoreDelta:           command.ScoreDelta,
		RewardSitoneID:       command.RewardSitoneID,
		RewardSitoneQuantity: command.RewardSitoneQuantity,
		SitoneIDs:            append([]string(nil), command.SitoneIDs...),
		SitoneEffect:         frontSitoneEffectResponse(command.SitoneEffect),
		Payload:              clonePayload(command.Payload),
		Accepted:             command.Accepted,
		Applied:              command.Applied,
		RejectReason:         command.RejectReason,
		CreatedAt:            command.CreatedAt,
	}
}

func commandSummaryResponse(command *mongomodel.FrontCommandSummary) *FrontCommandResponse {
	if command == nil {
		return nil
	}
	return &FrontCommandResponse{
		CommandID:            command.ID,
		ClientCommandID:      command.ClientCommandID,
		Kind:                 command.Kind,
		Type:                 command.Type,
		PlayerID:             command.PlayerID,
		TeamID:               command.TeamID,
		TargetX:              command.TargetX,
		TargetY:              command.TargetY,
		ExpectedRevision:     command.ExpectedRevision,
		AffectedCells:        coordinateResponses(command.AffectedCells),
		CapturedCellCount:    command.CapturedCellCount,
		EnclosedCellCount:    command.EnclosedCellCount,
		ScoreDelta:           command.ScoreDelta,
		RewardSitoneID:       command.RewardSitoneID,
		RewardSitoneQuantity: command.RewardSitoneQuantity,
		SitoneIDs:            append([]string(nil), command.SitoneIDs...),
		SitoneEffect:         frontSitoneEffectResponse(command.SitoneEffect),
		Payload:              clonePayload(command.Payload),
		Accepted:             command.Accepted,
		Applied:              command.Applied,
		RejectReason:         command.RejectReason,
		CreatedAt:            command.CreatedAt,
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

func timePtrIfSet(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
