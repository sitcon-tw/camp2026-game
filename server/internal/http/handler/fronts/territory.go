package fronts

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	contentFrontMapModeTerritoryGrid = content.FrontMapModeTerritoryGrid
	territoryMaxDefense              = 100
	territoryInitialDefense          = 20
	territoryExpandLimit             = 8
	territoryAttackLimit             = 6
	territoryReinforceLimit          = 8
	territorySitoneMilestoneSize     = 25
	frontTradeHourlyLimit            = 300
	frontGarrisonDefensePerSitone    = 5
	frontGarrisonAttackCostPercent   = 10
	frontTerritoryLevelCostPercent   = 20
	frontCaptureRewardChancePercent  = 30
	frontCaptureRewardMaximum        = 4
	frontTerritoryMaximumLevel       = 4
	frontTerritoryLevelDuration      = 90 * time.Minute
	frontTerritoryRewardInterval     = 30 * time.Minute
)

var territoryTeamColors = []string{
	"#e85d75", "#8f6bd8", "#2da985", "#4d91d1", "#f0b72f",
	"#d56a45", "#5d74c9", "#63a644", "#c45fa0",
}

type territoryCell struct {
	Playable             bool
	Owner                string
	Defense              int
	OccupiedAt           time.Time
	HoldingRewardPeriods int
}

type territoryCandidate struct {
	X        int
	Y        int
	Distance int
}

func frontFromTerritoryTemplate(template content.TerritoryMapTemplate) mongomodel.Front {
	territory := territoryFromTemplate(template)
	teams := territoryTeams()
	syncTerritoryTeamRanks(teams, territory)
	events := territoryEvents(territory.Landmarks)
	front := mongomodel.Front{
		ID:           template.ID,
		MapID:        template.ID,
		MapMode:      content.FrontMapModeTerritoryGrid,
		Name:         template.Name,
		Status:       mongomodel.FrontStatusOpenPlay,
		Current:      true,
		Revision:     1,
		Teams:        teams,
		Territory:    territory,
		ActiveEvents: events,
		CreatedAt:    seedFrontUpdatedAt,
		UpdatedAt:    seedFrontUpdatedAt,
	}
	front.Leaderboard = deriveLeaderboard(front)
	return front
}

func territoryFromTemplate(template content.TerritoryMapTemplate) *mongomodel.FrontTerritory {
	matrix := makeTerritoryMatrix(template.Grid.Width, template.Grid.Height)
	for y, row := range template.Grid.PlayableMask.Rows {
		for x := range row {
			matrix[y][x].Playable = row[x] == '1'
		}
	}

	bases := make([]mongomodel.FrontTerritoryBase, 0, len(template.Bases))
	for _, base := range template.Bases {
		bases = append(bases, mongomodel.FrontTerritoryBase{
			TeamID: base.TeamID, X: base.X, Y: base.Y,
			CoreDefense: base.CoreDefense, InitialRadius: base.InitialRadius,
		})
	}
	sort.Slice(bases, func(i, j int) bool { return bases[i].TeamID < bases[j].TeamID })
	initialDefense := template.BaseRules.InitialTerritory.Defense
	if initialDefense <= 0 {
		initialDefense = territoryInitialDefense
	}
	for _, base := range bases {
		for y := maxFrontInt(0, base.Y-base.InitialRadius); y <= minFrontInt(template.Grid.Height-1, base.Y+base.InitialRadius); y++ {
			for x := maxFrontInt(0, base.X-base.InitialRadius); x <= minFrontInt(template.Grid.Width-1, base.X+base.InitialRadius); x++ {
				if !matrix[y][x].Playable || absFrontInt(x-base.X)+absFrontInt(y-base.Y) > base.InitialRadius {
					continue
				}
				if matrix[y][x].Owner == "" {
					matrix[y][x].Owner = base.TeamID
					matrix[y][x].Defense = initialDefense
				}
			}
		}
		matrix[base.Y][base.X].Owner = base.TeamID
		matrix[base.Y][base.X].Defense = territoryMaxDefense
	}

	landmarks := make([]mongomodel.FrontMapLandmark, 0, len(template.Landmarks))
	for _, landmark := range template.Landmarks {
		landmarks = append(landmarks, mongomodel.FrontMapLandmark{
			ID: landmark.ID, Kind: landmark.Kind, X: landmark.X, Y: landmark.Y, Label: landmark.Label,
			InitialDefense: landmark.InitialDefense, InitialResource: landmark.InitialResource,
		})
		if matrix[landmark.Y][landmark.X].Owner == "" && landmark.InitialDefense > matrix[landmark.Y][landmark.X].Defense {
			matrix[landmark.Y][landmark.X].Defense = landmark.InitialDefense
		}
	}
	return &mongomodel.FrontTerritory{
		Width: template.Grid.Width, Height: template.Grid.Height,
		Connectivity: template.Grid.Connectivity,
		OriginX:      template.Grid.Origin.X, OriginY: template.Grid.Origin.Y,
		BackgroundSrc:   template.Background.Src,
		BackgroundWidth: template.Background.Width, BackgroundHeight: template.Background.Height,
		Rows: encodeTerritoryRows(matrix), Bases: bases, Landmarks: landmarks,
	}
}

func territoryEvents(landmarks []mongomodel.FrontMapLandmark) []mongomodel.FrontMapEvent {
	events := make([]mongomodel.FrontMapEvent, 0, len(landmarks))
	for _, landmark := range landmarks {
		kind := territoryEventKind(landmark.Kind)
		if kind == "" {
			continue
		}
		events = append(events, mongomodel.FrontMapEvent{
			ID: "territory_" + landmark.ID, CellID: landmark.ID,
			Kind: kind, Title: landmark.Label, StartsAtTick: 0, Severity: 1,
		})
	}
	return events
}

func territoryEventKind(kind string) string {
	switch kind {
	case "repair", "rescue", "resource", "challenge":
		return kind
	default:
		return ""
	}
}

func territoryTeams() []mongomodel.FrontTeam {
	teams := make([]mongomodel.FrontTeam, 0, 9)
	for i := 1; i <= 9; i++ {
		teamID := fmt.Sprintf("team-%03d", i)
		teams = append(teams, mongomodel.FrontTeam{
			TeamID: teamID, Name: fmt.Sprintf("第 %d 小隊", i),
			Color: territoryTeamColors[i-1], TradeHourlyLimit: frontTradeHourlyLimit,
		})
	}
	return teams
}

func isParticipatingTerritoryTeam(teamID string) bool {
	if len(teamID) != len("team-001") || !strings.HasPrefix(teamID, "team-") {
		return false
	}
	number, err := strconv.Atoi(strings.TrimPrefix(teamID, "team-"))
	return err == nil && number >= 1 && number <= 9
}

func territoryCommandOptionResponses(front mongomodel.Front, playerID string, teamID string, playerOpenPower int) []FrontCommandOptionResponse {
	if front.Territory == nil || !isParticipatingTerritoryTeam(teamID) {
		return []FrontCommandOptionResponse{}
	}
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		return []FrontCommandOptionResponse{}
	}
	hasNeutral := len(territoryBoundaryCandidates(matrix, teamID, "")) > 0
	hasEnemy := len(territoryEnemyBoundaryCandidates(matrix, teamID, territoryBaseSet(front.Territory.Bases))) > 0
	reinforceCost := territoryReinforceCost(matrix, teamID, 0, 0, territoryReinforceLimit)
	hasReinforce := reinforceCost > 0
	options := []FrontCommandOptionResponse{
		territoryCommandOption("expand", "擴張", frontCommandCosts["expand"], hasNeutral, playerOpenPower),
		territoryCommandOption("attack", "攻擊", frontCommandCosts["attack"], hasEnemy, playerOpenPower),
		territoryCommandOption("reinforce", "防守", reinforceCost, hasReinforce, playerOpenPower),
	}
	hasStationTarget := false
	bases := territoryBaseSet(front.Territory.Bases)
	for y := range matrix {
		for x := range matrix[y] {
			if matrix[y][x].Owner != teamID {
				continue
			}
			if _, isBase := bases[[2]int{x, y}]; isBase {
				continue
			}
			if _, occupied := frontGarrisonAt(front.Garrisons, x, y); !occupied {
				hasStationTarget = true
				break
			}
		}
		if hasStationTarget {
			break
		}
	}
	hasWithdrawTarget := false
	for _, garrison := range front.Garrisons {
		if garrison.PlayerID == playerID {
			hasWithdrawTarget = true
			break
		}
	}
	options = append(options,
		territoryCommandOption("station", "駐點", 0, hasStationTarget, playerOpenPower),
		territoryCommandOption("withdraw", "撤回", 0, hasWithdrawTarget, playerOpenPower),
	)
	landmarkByID := make(map[string]mongomodel.FrontMapLandmark, len(front.Territory.Landmarks))
	for _, landmark := range front.Territory.Landmarks {
		landmarkByID[landmark.ID] = landmark
	}
	for _, event := range front.ActiveEvents {
		landmark, ok := landmarkByID[event.CellID]
		if !ok || !territoryInBounds(matrix, landmark.X, landmark.Y) {
			continue
		}
		kind, label := territoryEventCommand(event.Kind)
		if kind == "" {
			continue
		}
		x, y := landmark.X, landmark.Y
		owned := matrix[y][x].Owner == teamID
		option := territoryCommandOption(kind, label, frontCommandCosts[kind], owned, playerOpenPower)
		option.TargetX = &x
		option.TargetY = &y
		if !owned {
			option.Reason = "必須先佔領地標"
		}
		options = append(options, option)
	}
	return options
}

func territoryEventCommand(eventKind string) (string, string) {
	switch eventKind {
	case "repair":
		return "repair", "修復"
	case "rescue":
		return "rescue", "救援"
	case "resource":
		return "support", "採集小石"
	case "challenge":
		return "answer_challenge", "挑戰"
	default:
		return "", ""
	}
}

func territoryCommandOption(kind string, label string, cost int, hasTarget bool, playerOpenPower int) FrontCommandOptionResponse {
	hasPower := playerOpenPower >= cost
	option := FrontCommandOptionResponse{Kind: kind, Label: label, Cost: cost, Enabled: hasTarget && hasPower}
	if !hasTarget {
		if kind == "reinforce" {
			option.Reason = "邊界防禦已滿或沒有可防守領土"
		} else {
			option.Reason = "目前沒有可用目標"
		}
	} else if !hasPower {
		option.Reason = "開源力不足"
	}
	return option
}

func applyTerritoryCommand(front mongomodel.Front, command mongomodel.FrontCommand, playerOpenPower int) (mongomodel.Front, mongomodel.FrontCommand, error) {
	if !isPlayableFrontStatus(front.Status) {
		return front, command, errors.New("front is not accepting commands")
	}
	if !isParticipatingTerritoryTeam(command.TeamID) {
		return front, command, errors.New("team is not allowed to play this front")
	}
	teamIndex := frontTeamIndex(front.Teams, command.TeamID)
	if teamIndex < 0 {
		return front, command, errors.New("team is not part of this front")
	}
	if command.TargetX == nil || command.TargetY == nil {
		return front, command, errors.New("target coordinates are required")
	}
	if front.Territory == nil {
		return front, command, errors.New("territory map is unavailable")
	}
	front = cloneFront(front)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		return front, command, err
	}
	x, y := *command.TargetX, *command.TargetY
	if !territoryInBounds(matrix, x, y) || !matrix[y][x].Playable {
		return front, command, errors.New("target must be a playable map cell")
	}
	target := matrix[y][x]
	bases := territoryBaseSet(front.Territory.Bases)
	if command.Kind == "station" || command.Kind == "withdraw" {
		return applyGarrisonCommand(front, matrix, teamIndex, command, bases)
	}
	switch command.Kind {
	case "expand":
		if target.Owner != "" {
			return front, command, errors.New("expand target must be neutral")
		}
	case "attack":
		if target.Owner == "" || target.Owner == command.TeamID {
			return front, command, errors.New("attack target must be controlled by another team")
		}
		if _, immutable := bases[[2]int{x, y}]; immutable {
			return front, command, errors.New("base cells cannot be attacked")
		}
	case "reinforce":
		if target.Owner != command.TeamID {
			return front, command, errors.New("reinforce target must be controlled by your team")
		}
	}

	cost, supported := frontCommandCosts[command.Kind]
	if !supported || !territoryCommandIsSupported(command.Kind) {
		return front, command, errors.New("command is unsupported on a territory map")
	}
	actionLimit := territoryCommandLimit(command.Kind, command.SitoneEffect.TotalBonusPercent)
	if command.Kind == "reinforce" {
		cost = territoryReinforceCost(matrix, command.TeamID, x, y, actionLimit)
	} else if command.Kind == "attack" {
		cost = territoryAttackOpenPowerCost(front, target, x, y, command.CreatedAt)
	}
	controlledBefore := territoryControlledCellCount(matrix, command.TeamID)
	if playerOpenPower < cost {
		return front, command, fmt.Errorf("%w: 無法執行前線命令", errFrontInsufficientOpenPower)
	}

	var affected []mongomodel.FrontCoordinate
	var captured []mongomodel.FrontCoordinate
	scoreDelta := 0
	landmarkSitoneReward := 0
	switch command.Kind {
	case "expand":
		affected = expandTerritory(matrix, command.TeamID, x, y, actionLimit)
		captured = append(captured, affected...)
	case "attack":
		affected, captured = attackTerritory(matrix, command.TeamID, target.Owner, x, y, actionLimit, bases)
	case "reinforce":
		affected = reinforceTerritory(matrix, command.TeamID, x, y, actionLimit)
	case "repair", "rescue", "support", "answer_challenge":
		var landmarkErr error
		affected, scoreDelta, landmarkSitoneReward, landmarkErr = applyTerritoryLandmarkCommand(&front, matrix, teamIndex, command, &command.SitoneEffect)
		if landmarkErr != nil {
			return front, command, landmarkErr
		}
	}
	if command.Kind == "expand" || command.Kind == "attack" || command.Kind == "reinforce" {
		command.SitoneEffect.AffectedCellBonus = maxFrontInt(0, len(affected)-territoryBaseCommandLimit(command.Kind))
	}
	enclosed := []mongomodel.FrontCoordinate(nil)
	if (command.Kind == "expand" || command.Kind == "attack") && len(captured) > 0 {
		enclosed = captureEnclosedTerritory(matrix, command.TeamID, bases)
		captured = appendUniqueTerritoryCoordinates(captured, enclosed...)
		affected = appendUniqueTerritoryCoordinates(affected, enclosed...)
		for _, coordinate := range captured {
			matrix[coordinate.Y][coordinate.X].OccupiedAt = command.CreatedAt
			matrix[coordinate.Y][coordinate.X].HoldingRewardPeriods = 0
		}
		displaceFrontGarrisons(&front, &command, captured)
	}
	if len(affected) == 0 {
		switch command.Kind {
		case "reinforce":
			return front, command, errors.New("all nearby boundary territory is already at maximum defense")
		case "expand":
			return front, command, errors.New("there is no connected neutral territory to expand into")
		default:
			return front, command, errors.New("there is no attackable enemy boundary near that direction")
		}
	}
	capturedScoreDelta := len(captured) * 10
	controlledAfter := territoryControlledCellCount(matrix, command.TeamID)
	milestonesEarned := applyTerritorySitoneMilestones(&front.Teams[teamIndex], controlledBefore, controlledAfter)

	front.Teams[teamIndex].LastCommandAt = command.CreatedAt
	front.Teams[teamIndex].Score += scoreDelta
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Revision++
	front.Tick++
	front.UpdatedAt = command.CreatedAt
	command.Accepted = true
	command.Applied = true
	command.AffectedCells = affected
	command.CapturedCellCount = len(captured)
	command.EnclosedCellCount = len(enclosed)
	command.ScoreDelta = scoreDelta + capturedScoreDelta
	command.FrontOpenPowerReward = territoryCaptureOpenPowerReward(front.ID, command.ID, captured)
	command.FrontOpenPowerDelta = command.FrontOpenPowerReward - cost
	command.FrontOpenPowerCost = cost
	command.RewardSitoneQuantity += milestonesEarned + landmarkSitoneReward
	summary := commandSummary(command)
	front.LastCommand = &summary
	syncTerritoryTeamRanks(front.Teams, front.Territory)
	front.Leaderboard = deriveLeaderboard(front)
	return front, command, nil
}

func territoryAttackOpenPowerCost(front mongomodel.Front, target territoryCell, x int, y int, now time.Time) int {
	sitoneCount := 0
	for _, garrison := range front.Garrisons {
		if garrison.X == x && garrison.Y == y {
			sitoneCount = len(garrison.SitoneIDs)
			break
		}
	}
	return frontAttackOpenPowerCost(territoryHoldingLevel(target.OccupiedAt, now), sitoneCount)
}

func frontAttackOpenPowerCost(level int, sitoneCount int) int {
	base := frontCommandCosts["attack"]
	level = clampFrontInt(level, 1, frontTerritoryMaximumLevel)
	sitoneCount = maxFrontInt(0, sitoneCount)
	percent := 100 + (level-1)*frontTerritoryLevelCostPercent + sitoneCount*frontGarrisonAttackCostPercent
	return (base*percent + 99) / 100
}

func territoryHoldingLevel(occupiedAt time.Time, now time.Time) int {
	if occupiedAt.IsZero() || now.Before(occupiedAt) {
		return 1
	}
	return clampFrontInt(int(now.Sub(occupiedAt)/frontTerritoryLevelDuration)+1, 1, frontTerritoryMaximumLevel)
}

func territoryCaptureOpenPowerReward(frontID string, commandID string, captured []mongomodel.FrontCoordinate) int {
	if len(captured) == 0 {
		return 0
	}
	reward := 0
	for _, coordinate := range captured {
		hash := fnv.New32a()
		_, _ = fmt.Fprintf(hash, "%s:%s:%d:%d", frontID, commandID, coordinate.X, coordinate.Y)
		if int(hash.Sum32()%100) < frontCaptureRewardChancePercent {
			reward++
		}
	}
	return clampFrontInt(reward, 1, frontCaptureRewardMaximum)
}

func territoryBaseCommandLimit(kind string) int {
	switch kind {
	case "expand":
		return territoryExpandLimit
	case "attack":
		return territoryAttackLimit
	case "reinforce":
		return territoryReinforceLimit
	default:
		return 0
	}
}

func territoryCommandLimit(kind string, bonusPercent int) int {
	base := territoryBaseCommandLimit(kind)
	return base + scaledFrontSitoneBonus(base, bonusPercent)
}

func territoryControlledCellCount(matrix [][]territoryCell, teamID string) int {
	count := 0
	for y := range matrix {
		for x := range matrix[y] {
			if matrix[y][x].Playable && matrix[y][x].Owner == teamID {
				count++
			}
		}
	}
	return count
}

func applyTerritorySitoneMilestones(team *mongomodel.FrontTeam, controlledBefore int, controlledAfter int) int {
	if team == nil {
		return 0
	}
	previousMax := maxFrontInt(team.MaxControlledCells, controlledBefore)
	previousReached := maxFrontInt(team.SitoneMilestonesReached, previousMax/territorySitoneMilestoneSize)
	previousMax = maxFrontInt(previousMax, previousReached*territorySitoneMilestoneSize)
	newMax := maxFrontInt(previousMax, controlledAfter)
	newReached := maxFrontInt(previousReached, newMax/territorySitoneMilestoneSize)
	team.MaxControlledCells = newMax
	team.SitoneMilestonesReached = newReached
	team.NextSitoneMilestone = (newReached + 1) * territorySitoneMilestoneSize
	return newReached - previousReached
}

func territoryCommandIsSupported(kind string) bool {
	switch kind {
	case "expand", "attack", "reinforce", "repair", "rescue", "support", "answer_challenge", "station", "withdraw":
		return true
	default:
		return false
	}
}

func applyTerritoryLandmarkCommand(front *mongomodel.Front, matrix [][]territoryCell, teamIndex int, command mongomodel.FrontCommand, effect *mongomodel.FrontSitoneEffect) ([]mongomodel.FrontCoordinate, int, int, error) {
	x, y := *command.TargetX, *command.TargetY
	landmark, ok := territoryLandmarkAt(front.Territory.Landmarks, x, y)
	if !ok {
		return nil, 0, 0, errors.New("target has no landmark for this command")
	}
	if matrix[y][x].Owner != command.TeamID {
		return nil, 0, 0, errors.New("your team must control the landmark first")
	}
	expectedKind := map[string]string{
		"repair": "repair", "rescue": "rescue", "support": "resource", "answer_challenge": "challenge",
	}[command.Kind]
	if activeEventKindAtFrontCell(*front, landmark.ID) != expectedKind {
		return nil, 0, 0, errors.New("landmark event is no longer active")
	}

	scoreDelta, sitoneReward := 0, 0
	switch command.Kind {
	case "repair":
		baseApplied := minFrontInt(25, territoryMaxDefense-matrix[y][x].Defense)
		defenseGain := 25 + scaledFrontSitoneBonus(25, effect.TotalBonusPercent)
		actualApplied := minFrontInt(defenseGain, territoryMaxDefense-matrix[y][x].Defense)
		matrix[y][x].Defense += actualApplied
		effect.DefenseBonus = maxFrontInt(0, actualApplied-baseApplied)
		front.Teams[teamIndex].RepairedEvents++
		scoreDelta = 30
	case "rescue":
		front.Teams[teamIndex].RescuedSitones++
		scoreDelta = 30
	case "support":
		sitoneReward = 1
		scoreDelta = 10
	case "answer_challenge":
		front.Teams[teamIndex].CollaborationScore += 3
		scoreDelta = 20
	}
	effect.ScoreBonus = scaledFrontSitoneBonus(scoreDelta, effect.TotalBonusPercent)
	scoreDelta += effect.ScoreBonus
	removeActiveFrontEventAtCell(front, landmark.ID)
	return []mongomodel.FrontCoordinate{{X: x, Y: y}}, scoreDelta, sitoneReward, nil
}

func territoryLandmarkAt(landmarks []mongomodel.FrontMapLandmark, x int, y int) (mongomodel.FrontMapLandmark, bool) {
	for _, landmark := range landmarks {
		if landmark.X == x && landmark.Y == y {
			return landmark, true
		}
	}
	return mongomodel.FrontMapLandmark{}, false
}

func expandTerritory(matrix [][]territoryCell, teamID string, targetX int, targetY int, limit int) []mongomodel.FrontCoordinate {
	candidates := territoryBoundaryCandidates(matrix, teamID, "")
	sortTerritoryCandidates(candidates, targetX, targetY)
	if len(candidates) == 0 {
		return nil
	}
	start := candidates[0]
	queue := []mongomodel.FrontCoordinate{{X: start.X, Y: start.Y}}
	seen := map[[2]int]bool{{start.X, start.Y}: true}
	affected := make([]mongomodel.FrontCoordinate, 0, limit)
	for len(queue) > 0 && len(affected) < limit {
		coordinate := queue[0]
		queue = queue[1:]
		cell := &matrix[coordinate.Y][coordinate.X]
		if !cell.Playable || cell.Owner != "" || !territoryCellTouchesOwner(matrix, coordinate.X, coordinate.Y, teamID) {
			continue
		}
		cell.Owner = teamID
		cell.Defense = territoryInitialDefense
		affected = append(affected, coordinate)
		neighbors := territoryNeighbors(matrix, coordinate.X, coordinate.Y)
		sort.Slice(neighbors, func(i, j int) bool {
			di := squaredDistance(neighbors[i].X, neighbors[i].Y, targetX, targetY)
			dj := squaredDistance(neighbors[j].X, neighbors[j].Y, targetX, targetY)
			if di != dj {
				return di < dj
			}
			if neighbors[i].Y != neighbors[j].Y {
				return neighbors[i].Y < neighbors[j].Y
			}
			return neighbors[i].X < neighbors[j].X
		})
		for _, neighbor := range neighbors {
			key := [2]int{neighbor.X, neighbor.Y}
			if !seen[key] && matrix[neighbor.Y][neighbor.X].Owner == "" {
				seen[key] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return affected
}

func attackTerritory(matrix [][]territoryCell, teamID string, targetOwner string, targetX int, targetY int, limit int, bases map[[2]int]string) ([]mongomodel.FrontCoordinate, []mongomodel.FrontCoordinate) {
	candidates := territoryEnemyBoundaryCandidates(matrix, teamID, bases)
	filtered := candidates[:0]
	for _, candidate := range candidates {
		if matrix[candidate.Y][candidate.X].Owner == targetOwner {
			filtered = append(filtered, candidate)
		}
	}
	candidates = filtered
	sortTerritoryCandidates(candidates, targetX, targetY)
	if len(candidates) == 0 {
		return nil, nil
	}
	start := candidates[0]
	candidateSet := make(map[[2]int]bool, len(candidates))
	for _, candidate := range candidates {
		if matrix[candidate.Y][candidate.X].Owner == targetOwner {
			candidateSet[[2]int{candidate.X, candidate.Y}] = true
		}
	}
	queue := []mongomodel.FrontCoordinate{{X: start.X, Y: start.Y}}
	seen := map[[2]int]bool{{start.X, start.Y}: true}
	affected := make([]mongomodel.FrontCoordinate, 0, limit)
	captured := make([]mongomodel.FrontCoordinate, 0, limit)
	for len(queue) > 0 && len(affected) < limit {
		coordinate := queue[0]
		queue = queue[1:]
		cell := &matrix[coordinate.Y][coordinate.X]
		if cell.Owner != targetOwner {
			continue
		}
		cell.Defense = maxFrontInt(0, cell.Defense-35)
		if cell.Defense == 0 {
			cell.Owner = teamID
			cell.Defense = territoryInitialDefense
			captured = append(captured, coordinate)
		}
		affected = append(affected, coordinate)
		neighbors := territoryNeighbors(matrix, coordinate.X, coordinate.Y)
		sort.Slice(neighbors, func(i, j int) bool {
			di := squaredDistance(neighbors[i].X, neighbors[i].Y, targetX, targetY)
			dj := squaredDistance(neighbors[j].X, neighbors[j].Y, targetX, targetY)
			if di != dj {
				return di < dj
			}
			if neighbors[i].Y != neighbors[j].Y {
				return neighbors[i].Y < neighbors[j].Y
			}
			return neighbors[i].X < neighbors[j].X
		})
		for _, neighbor := range neighbors {
			key := [2]int{neighbor.X, neighbor.Y}
			if candidateSet[key] && !seen[key] {
				seen[key] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return affected, captured
}

func reinforceTerritory(matrix [][]territoryCell, teamID string, targetX int, targetY int, limit int) []mongomodel.FrontCoordinate {
	candidates := territoryReinforceCandidates(matrix, teamID)
	sortTerritoryCandidates(candidates, targetX, targetY)
	remainingBudget := limit * 25
	affected := make([]mongomodel.FrontCoordinate, 0, minFrontInt(len(candidates), limit))
	for _, candidate := range candidates {
		if remainingBudget <= 0 {
			break
		}
		cell := &matrix[candidate.Y][candidate.X]
		applied := minFrontInt(25, minFrontInt(territoryMaxDefense-cell.Defense, remainingBudget))
		if applied <= 0 {
			continue
		}
		cell.Defense += applied
		remainingBudget -= applied
		affected = append(affected, mongomodel.FrontCoordinate{X: candidate.X, Y: candidate.Y})
	}
	return affected
}

func territoryReinforceCost(matrix [][]territoryCell, teamID string, targetX int, targetY int, limit int) int {
	candidates := territoryReinforceCandidates(matrix, teamID)
	sortTerritoryCandidates(candidates, targetX, targetY)
	budget := limit * 25
	applied := 0
	for _, candidate := range candidates {
		if applied >= budget {
			break
		}
		missing := territoryMaxDefense - matrix[candidate.Y][candidate.X].Defense
		applied += minFrontInt(25, minFrontInt(missing, budget-applied))
	}
	if applied <= 0 {
		return 0
	}
	return maxFrontInt(1, (frontCommandCosts["reinforce"]*applied+budget-1)/budget)
}

func territoryBoundaryCandidates(matrix [][]territoryCell, teamID string, targetOwner string) []territoryCandidate {
	var candidates []territoryCandidate
	for y := range matrix {
		for x := range matrix[y] {
			cell := matrix[y][x]
			if !cell.Playable || cell.Owner != targetOwner || !territoryCellTouchesOwner(matrix, x, y, teamID) {
				continue
			}
			candidates = append(candidates, territoryCandidate{X: x, Y: y})
		}
	}
	return candidates
}

func territoryEnemyBoundaryCandidates(matrix [][]territoryCell, teamID string, bases map[[2]int]string) []territoryCandidate {
	var candidates []territoryCandidate
	for y := range matrix {
		for x := range matrix[y] {
			cell := matrix[y][x]
			if !cell.Playable || cell.Owner == "" || cell.Owner == teamID || !territoryCellTouchesOwner(matrix, x, y, teamID) {
				continue
			}
			if _, immutable := bases[[2]int{x, y}]; immutable {
				continue
			}
			candidates = append(candidates, territoryCandidate{X: x, Y: y})
		}
	}
	return candidates
}

func territoryReinforceCandidates(matrix [][]territoryCell, teamID string) []territoryCandidate {
	var candidates []territoryCandidate
	for y := range matrix {
		for x := range matrix[y] {
			cell := matrix[y][x]
			if cell.Playable && cell.Owner == teamID && cell.Defense < territoryMaxDefense && territoryCellTouchesDifferentOwner(matrix, x, y, teamID) {
				candidates = append(candidates, territoryCandidate{X: x, Y: y})
			}
		}
	}
	return candidates
}

func territoryCellTouchesOwner(matrix [][]territoryCell, x int, y int, owner string) bool {
	for _, coordinate := range territoryNeighbors(matrix, x, y) {
		if matrix[coordinate.Y][coordinate.X].Owner == owner {
			return true
		}
	}
	return false
}

func territoryCellTouchesDifferentOwner(matrix [][]territoryCell, x int, y int, owner string) bool {
	for _, coordinate := range territoryNeighbors(matrix, x, y) {
		if matrix[coordinate.Y][coordinate.X].Playable && matrix[coordinate.Y][coordinate.X].Owner != owner {
			return true
		}
	}
	return false
}

func territoryNeighbors(matrix [][]territoryCell, x int, y int) []mongomodel.FrontCoordinate {
	directions := [][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}}
	out := make([]mongomodel.FrontCoordinate, 0, 4)
	for _, direction := range directions {
		nx, ny := x+direction[0], y+direction[1]
		if territoryInBounds(matrix, nx, ny) && matrix[ny][nx].Playable {
			out = append(out, mongomodel.FrontCoordinate{X: nx, Y: ny})
		}
	}
	return out
}

func territoryBaseSet(bases []mongomodel.FrontTerritoryBase) map[[2]int]string {
	out := make(map[[2]int]string, len(bases))
	for _, base := range bases {
		out[[2]int{base.X, base.Y}] = base.TeamID
	}
	return out
}

func captureEnclosedTerritory(matrix [][]territoryCell, teamID string, bases map[[2]int]string) []mongomodel.FrontCoordinate {
	if len(matrix) == 0 || len(matrix[0]) == 0 {
		return nil
	}
	height, width := len(matrix), len(matrix[0])
	directions := [][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}}

	outsideBlocked := make([][]bool, height)
	for y := range outsideBlocked {
		outsideBlocked[y] = make([]bool, width)
	}
	blockedQueue := make([]mongomodel.FrontCoordinate, 0)
	enqueueBlocked := func(x int, y int) {
		if x < 0 || y < 0 || x >= width || y >= height || matrix[y][x].Playable || outsideBlocked[y][x] {
			return
		}
		outsideBlocked[y][x] = true
		blockedQueue = append(blockedQueue, mongomodel.FrontCoordinate{X: x, Y: y})
	}
	for x := 0; x < width; x++ {
		enqueueBlocked(x, 0)
		enqueueBlocked(x, height-1)
	}
	for y := 0; y < height; y++ {
		enqueueBlocked(0, y)
		enqueueBlocked(width-1, y)
	}
	for len(blockedQueue) > 0 {
		coordinate := blockedQueue[0]
		blockedQueue = blockedQueue[1:]
		for _, direction := range directions {
			enqueueBlocked(coordinate.X+direction[0], coordinate.Y+direction[1])
		}
	}

	exteriorReachable := make([][]bool, height)
	for y := range exteriorReachable {
		exteriorReachable[y] = make([]bool, width)
	}
	exteriorQueue := make([]mongomodel.FrontCoordinate, 0)
	enqueueExterior := func(x int, y int) {
		if x < 0 || y < 0 || x >= width || y >= height || !matrix[y][x].Playable || matrix[y][x].Owner == teamID || exteriorReachable[y][x] {
			return
		}
		exteriorReachable[y][x] = true
		exteriorQueue = append(exteriorQueue, mongomodel.FrontCoordinate{X: x, Y: y})
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !matrix[y][x].Playable || matrix[y][x].Owner == teamID {
				continue
			}
			isExterior := x == 0 || y == 0 || x == width-1 || y == height-1
			for _, direction := range directions {
				nx, ny := x+direction[0], y+direction[1]
				if nx >= 0 && ny >= 0 && nx < width && ny < height && outsideBlocked[ny][nx] {
					isExterior = true
				}
			}
			if isExterior {
				enqueueExterior(x, y)
			}
		}
	}
	for len(exteriorQueue) > 0 {
		coordinate := exteriorQueue[0]
		exteriorQueue = exteriorQueue[1:]
		for _, direction := range directions {
			enqueueExterior(coordinate.X+direction[0], coordinate.Y+direction[1])
		}
	}

	processed := make([][]bool, height)
	for y := range processed {
		processed[y] = make([]bool, width)
	}
	captured := make([]mongomodel.FrontCoordinate, 0)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !matrix[y][x].Playable || matrix[y][x].Owner == teamID || exteriorReachable[y][x] || processed[y][x] {
				continue
			}
			component := []mongomodel.FrontCoordinate{{X: x, Y: y}}
			processed[y][x] = true
			protectedByBase := false
			for index := 0; index < len(component); index++ {
				coordinate := component[index]
				if baseTeam, isBase := bases[[2]int{coordinate.X, coordinate.Y}]; isBase && baseTeam != teamID {
					protectedByBase = true
				}
				for _, direction := range directions {
					nx, ny := coordinate.X+direction[0], coordinate.Y+direction[1]
					if nx < 0 || ny < 0 || nx >= width || ny >= height || processed[ny][nx] || exteriorReachable[ny][nx] {
						continue
					}
					if matrix[ny][nx].Playable && matrix[ny][nx].Owner != teamID {
						processed[ny][nx] = true
						component = append(component, mongomodel.FrontCoordinate{X: nx, Y: ny})
					}
				}
			}
			if protectedByBase {
				continue
			}
			for _, coordinate := range component {
				matrix[coordinate.Y][coordinate.X].Owner = teamID
				matrix[coordinate.Y][coordinate.X].Defense = territoryInitialDefense
				captured = append(captured, coordinate)
			}
		}
	}
	sort.Slice(captured, func(i, j int) bool {
		if captured[i].Y != captured[j].Y {
			return captured[i].Y < captured[j].Y
		}
		return captured[i].X < captured[j].X
	})
	return captured
}

func appendUniqueTerritoryCoordinates(existing []mongomodel.FrontCoordinate, additions ...mongomodel.FrontCoordinate) []mongomodel.FrontCoordinate {
	seen := make(map[[2]int]struct{}, len(existing)+len(additions))
	for _, coordinate := range existing {
		seen[[2]int{coordinate.X, coordinate.Y}] = struct{}{}
	}
	for _, coordinate := range additions {
		key := [2]int{coordinate.X, coordinate.Y}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, coordinate)
	}
	return existing
}

func decodeTerritoryRows(territory *mongomodel.FrontTerritory) ([][]territoryCell, error) {
	if territory == nil || territory.Width <= 0 || territory.Height <= 0 {
		return nil, errors.New("territory dimensions are invalid")
	}
	matrix := makeTerritoryMatrix(territory.Width, territory.Height)
	for _, row := range territory.Rows {
		if row.Y < 0 || row.Y >= territory.Height {
			return nil, errors.New("territory row is out of bounds")
		}
		for _, run := range row.Runs {
			if run.X < 0 || run.Length <= 0 || run.X+run.Length > territory.Width || run.Defense < 0 || run.Defense > territoryMaxDefense {
				return nil, errors.New("territory run is invalid")
			}
			for x := run.X; x < run.X+run.Length; x++ {
				if matrix[row.Y][x].Playable {
					return nil, errors.New("territory runs overlap")
				}
				matrix[row.Y][x] = territoryCell{
					Playable: true, Owner: run.OwnerTeamID, Defense: run.Defense,
					OccupiedAt: run.OccupiedAt, HoldingRewardPeriods: run.HoldingRewardPeriods,
				}
			}
		}
	}
	return matrix, nil
}

func encodeTerritoryRows(matrix [][]territoryCell) []mongomodel.FrontTerritoryRow {
	rows := make([]mongomodel.FrontTerritoryRow, 0, len(matrix))
	for y, cells := range matrix {
		row := mongomodel.FrontTerritoryRow{Y: y}
		for x := 0; x < len(cells); {
			if !cells[x].Playable {
				x++
				continue
			}
			start := x
			owner, defense := cells[x].Owner, cells[x].Defense
			occupiedAt, holdingPeriods := cells[x].OccupiedAt, cells[x].HoldingRewardPeriods
			x++
			for x < len(cells) && cells[x].Playable && cells[x].Owner == owner && cells[x].Defense == defense && cells[x].OccupiedAt.Equal(occupiedAt) && cells[x].HoldingRewardPeriods == holdingPeriods {
				x++
			}
			row.Runs = append(row.Runs, mongomodel.FrontTerritoryRun{
				X: start, Length: x - start, OwnerTeamID: owner, Defense: defense,
				OccupiedAt: occupiedAt, HoldingRewardPeriods: holdingPeriods,
			})
		}
		if len(row.Runs) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func syncTerritoryTeamRanks(teams []mongomodel.FrontTeam, territory *mongomodel.FrontTerritory) {
	controlled := make(map[string]int)
	if matrix, err := decodeTerritoryRows(territory); err == nil {
		for y := range matrix {
			for x := range matrix[y] {
				if matrix[y][x].Owner != "" {
					controlled[matrix[y][x].Owner]++
				}
			}
		}
	}
	for i := range teams {
		teams[i].ControlledCells = controlled[teams[i].TeamID]
		teams[i].MaxControlledCells = maxFrontInt(teams[i].MaxControlledCells, teams[i].ControlledCells)
		teams[i].SitoneMilestonesReached = maxFrontInt(teams[i].SitoneMilestonesReached, teams[i].MaxControlledCells/territorySitoneMilestoneSize)
		teams[i].NextSitoneMilestone = (teams[i].SitoneMilestonesReached + 1) * territorySitoneMilestoneSize
	}
	sort.SliceStable(teams, func(i, j int) bool {
		iTotal := teams[i].Score + teams[i].ControlledCells*10
		jTotal := teams[j].Score + teams[j].ControlledCells*10
		if iTotal != jTotal {
			return iTotal > jTotal
		}
		if teams[i].ControlledCells != teams[j].ControlledCells {
			return teams[i].ControlledCells > teams[j].ControlledCells
		}
		return teams[i].TeamID < teams[j].TeamID
	})
	for i := range teams {
		teams[i].Rank = i + 1
	}
	sort.SliceStable(teams, func(i, j int) bool { return teams[i].TeamID < teams[j].TeamID })
}

func sortTerritoryCandidates(candidates []territoryCandidate, targetX int, targetY int) {
	for i := range candidates {
		candidates[i].Distance = squaredDistance(candidates[i].X, candidates[i].Y, targetX, targetY)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Distance != candidates[j].Distance {
			return candidates[i].Distance < candidates[j].Distance
		}
		if candidates[i].Y != candidates[j].Y {
			return candidates[i].Y < candidates[j].Y
		}
		return candidates[i].X < candidates[j].X
	})
}

func makeTerritoryMatrix(width int, height int) [][]territoryCell {
	matrix := make([][]territoryCell, height)
	for y := range matrix {
		matrix[y] = make([]territoryCell, width)
	}
	return matrix
}

func territoryInBounds(matrix [][]territoryCell, x int, y int) bool {
	return y >= 0 && y < len(matrix) && x >= 0 && x < len(matrix[y])
}

func squaredDistance(x1 int, y1 int, x2 int, y2 int) int {
	dx, dy := x1-x2, y1-y2
	return dx*dx + dy*dy
}

func absFrontInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
func minFrontInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
