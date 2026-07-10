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

func detailResponse(front mongomodel.Front, currentPlayerID string, currentTeamID string, sitones frontSitoneInventory, playerOpenPower int) DetailResponse {
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
		AvailableCommands:      commandOptionResponses(front, currentPlayerID, currentTeamID, playerOpenPower),
		Garrisons:              garrisonResponses(front.Garrisons, currentPlayerID),
		RailSegments:           railSegmentResponses(front.RailSegments),
		TradeRoutes:            tradeRouteResponses(front.TradeRoutes),
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
			TradeScore:              team.TradeScore,
			TradeHourlyEarned:       team.TradeHourlyEarned,
			TradeHourlyLimit:        team.TradeHourlyLimit,
			TradeWindowEndsAt:       tradeWindowEndsAt(team.TradeWindowStartedAt),
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
			TradeScore:         entry.TradeScore,
			Current:            entry.TeamID != "" && entry.TeamID == currentTeamID,
		})
	}
	return out
}

func commandOptionResponses(front mongomodel.Front, currentPlayerID string, currentTeamID string, playerOpenPower int) []FrontCommandOptionResponse {
	return territoryCommandOptionResponses(front, currentPlayerID, currentTeamID, playerOpenPower)
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
		GarrisonID:           command.GarrisonID,
		CapturedGarrisons:    capturedGarrisonResponses(command.CapturedGarrisons),
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
		GarrisonID:           command.GarrisonID,
		CapturedGarrisons:    capturedGarrisonResponses(command.CapturedGarrisons),
		Payload:              clonePayload(command.Payload),
		Accepted:             command.Accepted,
		Applied:              command.Applied,
		RejectReason:         command.RejectReason,
		CreatedAt:            command.CreatedAt,
	}
}

func garrisonResponses(garrisons []mongomodel.FrontGarrison, currentPlayerID string) []FrontGarrisonResponse {
	out := make([]FrontGarrisonResponse, 0, len(garrisons))
	for _, garrison := range garrisons {
		out = append(out, FrontGarrisonResponse{
			ID: garrison.ID, PlayerID: garrison.PlayerID, TeamID: garrison.TeamID,
			X: garrison.X, Y: garrison.Y, SitoneIDs: append([]string(nil), garrison.SitoneIDs...),
			SitoneCount: len(garrison.SitoneIDs), DefenseBonus: garrison.DefenseBonus,
			TradeBonusPercent: garrison.TradeBonusPercent,
			Mine:              garrison.PlayerID != "" && garrison.PlayerID == currentPlayerID,
			StationedAt:       garrison.StationedAt,
		})
	}
	return out
}

func tradeRouteResponses(routes []mongomodel.FrontTradeRoute) []FrontTradeRouteResponse {
	out := make([]FrontTradeRouteResponse, 0, len(routes))
	for _, route := range routes {
		out = append(out, FrontTradeRouteResponse{
			ID: route.ID, SourceGarrisonID: route.SourceGarrisonID, TargetGarrisonID: route.TargetGarrisonID,
			SourceTeamID: route.SourceTeamID, TargetTeamID: route.TargetTeamID,
			SourceX: route.SourceX, SourceY: route.SourceY, TargetX: route.TargetX, TargetY: route.TargetY,
			Distance: route.Distance, PotentialReward: route.PotentialReward,
			SourceReward: route.SourceReward, TargetReward: route.TargetReward,
			Waypoints: tradeWaypointResponses(route.Waypoints), NextStopIndex: route.NextStopIndex,
			Status: route.Status, StartedAt: route.StartedAt, ArrivesAt: route.ArrivesAt,
			SettledAt: route.SettledAt, CancellationReason: route.CancellationReason,
		})
	}
	return out
}

func railSegmentResponses(segments []mongomodel.FrontRailSegment) []FrontRailSegmentResponse {
	out := make([]FrontRailSegmentResponse, 0, len(segments))
	for _, segment := range segments {
		out = append(out, FrontRailSegmentResponse{
			ID: segment.ID, SourceGarrisonID: segment.SourceGarrisonID, TargetGarrisonID: segment.TargetGarrisonID,
			SourceX: segment.SourceX, SourceY: segment.SourceY, TargetX: segment.TargetX, TargetY: segment.TargetY,
			Distance: segment.Distance,
		})
	}
	return out
}

func tradeWaypointResponses(waypoints []mongomodel.FrontTradeWaypoint) []FrontTradeWaypointResponse {
	out := make([]FrontTradeWaypointResponse, 0, len(waypoints))
	for _, waypoint := range waypoints {
		out = append(out, FrontTradeWaypointResponse{
			GarrisonID: waypoint.GarrisonID, TeamID: waypoint.TeamID, X: waypoint.X, Y: waypoint.Y,
			ArrivesAt: waypoint.ArrivesAt, PotentialReward: waypoint.PotentialReward,
			SourceReward: waypoint.SourceReward, TargetReward: waypoint.TargetReward, SettledAt: waypoint.SettledAt,
		})
	}
	return out
}

func capturedGarrisonResponses(garrisons []mongomodel.FrontCapturedGarrison) []FrontCapturedGarrisonResponse {
	out := make([]FrontCapturedGarrisonResponse, 0, len(garrisons))
	for _, garrison := range garrisons {
		out = append(out, FrontCapturedGarrisonResponse{
			GarrisonID: garrison.GarrisonID, PlayerID: garrison.PlayerID, TeamID: garrison.TeamID,
			X: garrison.X, Y: garrison.Y, SitoneIDs: append([]string(nil), garrison.SitoneIDs...),
		})
	}
	return out
}

func tradeWindowEndsAt(startedAt time.Time) *time.Time {
	if startedAt.IsZero() {
		return nil
	}
	endsAt := startedAt.Add(time.Hour)
	return &endsAt
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
