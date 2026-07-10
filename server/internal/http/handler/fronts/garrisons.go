package fronts

import (
	"errors"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	frontTradeRouteActive    = "active"
	frontTradeRouteCompleted = "completed"
	frontTradeRouteCancelled = "cancelled"
)

func applyGarrisonCommand(
	front mongomodel.Front,
	matrix [][]territoryCell,
	teamIndex int,
	command mongomodel.FrontCommand,
	bases map[[2]int]string,
) (mongomodel.Front, mongomodel.FrontCommand, error) {
	x, y := *command.TargetX, *command.TargetY
	target := matrix[y][x]
	if target.Owner != command.TeamID {
		return front, command, errors.New("garrisons must be placed on territory controlled by your team")
	}
	if _, immutable := bases[[2]int{x, y}]; immutable {
		return front, command, errors.New("garrisons cannot be placed on permanent bases")
	}

	switch command.Kind {
	case "station":
		if len(command.SitoneIDs) < 1 || len(command.SitoneIDs) > maxFrontSitoneLoadout {
			return front, command, errors.New("station requires between 1 and 5 sitones")
		}
		if _, found := frontGarrisonAt(front.Garrisons, x, y); found {
			return front, command, errors.New("this territory already has a garrison")
		}
		baseDefense := frontGarrisonDefensePerSitone * len(command.SitoneIDs)
		requestedDefense := baseDefense + scaledFrontSitoneBonus(baseDefense, command.SitoneEffect.TotalBonusPercent)
		previousDefense := matrix[y][x].Defense
		matrix[y][x].Defense = clampFrontInt(previousDefense+requestedDefense, 0, territoryMaxDefense)
		defenseBonus := matrix[y][x].Defense - previousDefense
		command.SitoneEffect.DefenseBonus = defenseBonus
		command.GarrisonID = "front_garrison_" + command.ID
		front.Garrisons = append(front.Garrisons, mongomodel.FrontGarrison{
			ID:                command.GarrisonID,
			PlayerID:          command.PlayerID,
			TeamID:            command.TeamID,
			X:                 x,
			Y:                 y,
			SitoneIDs:         append([]string(nil), command.SitoneIDs...),
			DefenseBonus:      defenseBonus,
			TradeBonusPercent: command.SitoneEffect.TotalBonusPercent,
			StationedAt:       command.CreatedAt,
		})
	case "withdraw":
		garrison, found := frontGarrisonAt(front.Garrisons, x, y)
		if !found {
			return front, command, errors.New("this territory has no garrison")
		}
		if garrison.PlayerID != command.PlayerID {
			return front, command, errors.New("only the player who stationed this garrison can withdraw it")
		}
		command.GarrisonID = garrison.ID
		command.SitoneIDs = append([]string(nil), garrison.SitoneIDs...)
		matrix[y][x].Defense = maxFrontInt(0, matrix[y][x].Defense-garrison.DefenseBonus)
		front.Garrisons = removeFrontGarrison(front.Garrisons, garrison.ID)
	default:
		return front, command, errors.New("unsupported garrison command")
	}
	reconcileFrontRailNetwork(&front, command.CreatedAt, "garrison network changed")

	front.Teams[teamIndex].LastCommandAt = command.CreatedAt
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Revision++
	front.Tick++
	front.UpdatedAt = command.CreatedAt
	command.Accepted = true
	command.Applied = true
	command.AffectedCells = []mongomodel.FrontCoordinate{{X: x, Y: y}}
	front.LastCommand = ptrFrontCommandSummary(commandSummary(command))
	syncTerritoryTeamRanks(front.Teams, front.Territory)
	front.Leaderboard = deriveLeaderboard(front)
	return front, command, nil
}

func frontGarrisonAt(garrisons []mongomodel.FrontGarrison, x int, y int) (mongomodel.FrontGarrison, bool) {
	for _, garrison := range garrisons {
		if garrison.X == x && garrison.Y == y {
			return garrison, true
		}
	}
	return mongomodel.FrontGarrison{}, false
}

func removeFrontGarrison(garrisons []mongomodel.FrontGarrison, garrisonID string) []mongomodel.FrontGarrison {
	out := make([]mongomodel.FrontGarrison, 0, len(garrisons))
	for _, garrison := range garrisons {
		if garrison.ID != garrisonID {
			out = append(out, garrison)
		}
	}
	return out
}

func captureFrontGarrisons(front *mongomodel.Front, command *mongomodel.FrontCommand, captured []mongomodel.FrontCoordinate) {
	if front == nil || command == nil || len(front.Garrisons) == 0 || len(captured) == 0 {
		return
	}
	capturedCells := make(map[[2]int]struct{}, len(captured))
	for _, coordinate := range captured {
		capturedCells[[2]int{coordinate.X, coordinate.Y}] = struct{}{}
	}
	removedIDs := make(map[string]struct{})
	remaining := make([]mongomodel.FrontGarrison, 0, len(front.Garrisons))
	for _, garrison := range front.Garrisons {
		if _, ok := capturedCells[[2]int{garrison.X, garrison.Y}]; !ok || garrison.TeamID == command.TeamID {
			remaining = append(remaining, garrison)
			continue
		}
		removedIDs[garrison.ID] = struct{}{}
		command.CapturedGarrisons = append(command.CapturedGarrisons, mongomodel.FrontCapturedGarrison{
			GarrisonID: garrison.ID,
			PlayerID:   garrison.PlayerID,
			TeamID:     garrison.TeamID,
			X:          garrison.X,
			Y:          garrison.Y,
			SitoneIDs:  append([]string(nil), garrison.SitoneIDs...),
		})
	}
	front.Garrisons = remaining
	if len(removedIDs) > 0 {
		reconcileFrontRailNetwork(front, command.CreatedAt, "garrison captured")
	}
}

func ptrFrontCommandSummary(summary mongomodel.FrontCommandSummary) *mongomodel.FrontCommandSummary {
	return &summary
}

func timePtr(value time.Time) *time.Time {
	return &value
}
