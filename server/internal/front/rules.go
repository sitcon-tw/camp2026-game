package front

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
)

const (
	StatusClosed       = "closed"
	StatusQuiet        = "quiet"
	StatusOpenPlay     = "open_play"
	StatusSurge        = "surge"
	StatusFinaleFreeze = "finale_freeze"
	StatusCompleted    = "completed"

	CommandExpand    = "expand"
	CommandAttack    = "attack"
	CommandReinforce = "reinforce"
	CommandRepair    = "repair"
	CommandScout     = "scout"
	CommandRescue    = "rescue"
	CommandSupport   = "support"
)

const (
	defaultInitialFrontOpenPower = 100
	defaultCommandCooldownTicks  = int64(2)
	defaultCommandResolveTicks   = int64(1)
	defaultResourceDecayPerTick  = 1
	defaultSupportTokens         = 1
	maxDefense                   = 100
)

var commandCosts = map[string]int{
	CommandExpand:    10,
	CommandAttack:    15,
	CommandReinforce: 8,
	CommandRepair:    12,
	CommandScout:     4,
	CommandRescue:    10,
	CommandSupport:   0,
}

type TeamSeed struct {
	ID             string
	Color          string
	FrontOpenPower int
	SupportTokens  int
}

type State struct {
	MapID                     string
	MapName                   string
	Status                    string
	Tick                      int64
	TickSeconds               int
	Cells                     []Cell
	Teams                     []TeamState
	EventSchedule             []Event
	ActiveEvents              []Event
	PlayerLastCommandTick     map[string]int64
	SitoneBusyUntilTick       map[string]int64
	CommandCooldownTicks      int64
	CommandResolveTicks       int64
	ResourceDecayPerTick      int
	TeamOpenPowerDecayPerTick int
	UpdatedAt                 time.Time
}

type Cell struct {
	ID             string
	X              int
	Y              int
	Terrain        string
	Zone           string
	OwnerTeamID    string
	Control        int
	Defense        int
	Resource       int
	PressureByTeam map[string]int
	NeighborIDs    []string
	EventID        string
}

type TeamState struct {
	TeamID             string
	Color              string
	Score              int
	Rank               int
	PreviousRank       int
	FrontOpenPower     int
	SupportTokens      int
	ControlledCells    int
	RescuedSitones     int
	RepairedEvents     int
	CollaborationScore int
	LastCommandTick    int64
}

type Event struct {
	ID            string
	CellID        string
	Kind          string
	Title         string
	StartsAtTick  int64
	ExpiresAtTick int64
	Severity      int
}

type Command struct {
	ID         string
	PlayerID   string
	TeamID     string
	Kind       string
	FromCellID string
	ToCellID   string
	SitoneID   string
}

type CommandResult struct {
	CommandID      string
	PlayerID       string
	TeamID         string
	Kind           string
	CellID         string
	Accepted       bool
	Applied        bool
	RejectReason   string
	ScoreDelta     int
	ResourceDelta  int
	CapturedCell   bool
	ResolvedAtTick int64
	ResolvedAt     time.Time
}

type LeaderboardEntry struct {
	TeamID             string
	Color              string
	Rank               int
	PreviousRank       int
	Score              int
	ControlledCells    int
	FrontOpenPower     int
	RescuedSitones     int
	RepairedEvents     int
	CollaborationScore int
}

func NewState(template content.FrontMapTemplate, teams []TeamSeed) (State, error) {
	if err := content.ValidateFrontMap(template); err != nil {
		return State{}, err
	}

	teams = normalizeTeamSeeds(teams)
	if len(teams) == 0 {
		return State{}, errors.New("teams are required")
	}
	if err := validateTeamSeeds(teams); err != nil {
		return State{}, err
	}

	baseCells := baseCellIDs(template.Cells)
	if len(baseCells) < len(teams) {
		return State{}, fmt.Errorf("front map %q has %d base cells for %d teams", template.ID, len(baseCells), len(teams))
	}

	initialFrontOpenPower := template.InitialFrontOpenPower
	if initialFrontOpenPower == 0 {
		initialFrontOpenPower = defaultInitialFrontOpenPower
	}

	commandCooldownTicks := template.CommandCooldownTicks
	if commandCooldownTicks == 0 {
		commandCooldownTicks = defaultCommandCooldownTicks
	}

	commandResolveTicks := template.CommandResolveTicks
	if commandResolveTicks == 0 {
		commandResolveTicks = defaultCommandResolveTicks
	}

	resourceDecayPerTick := template.ResourceDecayPerTick
	if resourceDecayPerTick == 0 {
		resourceDecayPerTick = defaultResourceDecayPerTick
	}

	state := State{
		MapID:                 strings.TrimSpace(template.ID),
		MapName:               strings.TrimSpace(template.Name),
		Status:                StatusOpenPlay,
		TickSeconds:           template.TickSeconds,
		Cells:                 make([]Cell, 0, len(template.Cells)),
		Teams:                 make([]TeamState, 0, len(teams)),
		EventSchedule:         make([]Event, 0, len(template.Events)),
		PlayerLastCommandTick: make(map[string]int64),
		SitoneBusyUntilTick:   make(map[string]int64),
		CommandCooldownTicks:  commandCooldownTicks,
		CommandResolveTicks:   commandResolveTicks,
		ResourceDecayPerTick:  resourceDecayPerTick,
	}

	baseOwnerByCell := make(map[string]string, len(teams))
	for i, team := range teams {
		baseOwnerByCell[baseCells[i]] = team.ID
	}

	for _, cell := range template.Cells {
		stateCell := Cell{
			ID:             strings.TrimSpace(cell.ID),
			X:              cell.X,
			Y:              cell.Y,
			Terrain:        strings.TrimSpace(cell.Terrain),
			Zone:           strings.TrimSpace(cell.Zone),
			Control:        cell.InitialControl,
			Defense:        cell.InitialDefense,
			Resource:       cell.InitialResource,
			PressureByTeam: make(map[string]int),
			NeighborIDs:    copyStrings(cell.Neighbors),
		}
		if ownerTeamID := baseOwnerByCell[stateCell.ID]; ownerTeamID != "" {
			stateCell.OwnerTeamID = ownerTeamID
			if stateCell.Control == 0 {
				stateCell.Control = 100
			}
			if stateCell.Defense == 0 {
				stateCell.Defense = 20
			}
		}
		state.Cells = append(state.Cells, stateCell)
	}

	for _, team := range teams {
		frontOpenPower := team.FrontOpenPower
		if frontOpenPower == 0 {
			frontOpenPower = initialFrontOpenPower
		}
		supportTokens := team.SupportTokens
		if supportTokens == 0 {
			supportTokens = defaultSupportTokens
		}
		state.Teams = append(state.Teams, TeamState{
			TeamID:         team.ID,
			Color:          team.Color,
			FrontOpenPower: frontOpenPower,
			SupportTokens:  supportTokens,
		})
	}

	for _, eventTemplate := range template.Events {
		event := Event{
			ID:            strings.TrimSpace(eventTemplate.ID),
			CellID:        strings.TrimSpace(eventTemplate.CellID),
			Kind:          strings.TrimSpace(eventTemplate.Kind),
			Title:         strings.TrimSpace(eventTemplate.Title),
			StartsAtTick:  eventTemplate.StartsAtTick,
			ExpiresAtTick: eventTemplate.StartsAtTick + eventTemplate.ExpiresAfterTicks,
			Severity:      eventTemplate.Severity,
		}
		state.EventSchedule = append(state.EventSchedule, event)
		if event.StartsAtTick <= state.Tick {
			activateEvent(&state, event)
		}
	}

	refreshTeamDerived(&state)
	updateTeamRanks(&state)
	return state, nil
}

func ValidateCommand(state State, command Command) error {
	command = normalizeCommand(command)

	var errs []error
	if !isPlayableStatus(state.Status) {
		errs = append(errs, fmt.Errorf("front status %q does not accept commands", state.Status))
	}
	if command.PlayerID == "" {
		errs = append(errs, errors.New("player_id is required"))
	}
	if command.TeamID == "" {
		errs = append(errs, errors.New("team_id is required"))
	}
	if command.Kind == "" {
		errs = append(errs, errors.New("kind is required"))
	} else if _, ok := commandCosts[command.Kind]; !ok {
		errs = append(errs, fmt.Errorf("kind %q is not supported", command.Kind))
	}
	if command.FromCellID == "" {
		errs = append(errs, errors.New("from_cell_id is required"))
	}
	if command.ToCellID == "" {
		errs = append(errs, errors.New("to_cell_id is required"))
	}
	if command.SitoneID == "" {
		errs = append(errs, errors.New("sitone_id is required"))
	}
	if command.FromCellID != "" && command.FromCellID == command.ToCellID {
		errs = append(errs, errors.New("from_cell_id and to_cell_id must be different"))
	}

	teamIndex := teamIndexByID(state, command.TeamID)
	if command.TeamID != "" && teamIndex < 0 {
		errs = append(errs, fmt.Errorf("team %q is not in this front", command.TeamID))
	}

	fromIndex := cellIndexByID(state, command.FromCellID)
	if command.FromCellID != "" && fromIndex < 0 {
		errs = append(errs, fmt.Errorf("from cell %q does not exist", command.FromCellID))
	}

	toIndex := cellIndexByID(state, command.ToCellID)
	if command.ToCellID != "" && toIndex < 0 {
		errs = append(errs, fmt.Errorf("to cell %q does not exist", command.ToCellID))
	}

	if fromIndex >= 0 && toIndex >= 0 {
		fromCell := state.Cells[fromIndex]
		toCell := state.Cells[toIndex]
		if !hasNeighbor(fromCell, command.ToCellID) {
			errs = append(errs, fmt.Errorf("cell %q is not adjacent to %q", command.ToCellID, command.FromCellID))
		}
		if fromCell.OwnerTeamID != command.TeamID {
			errs = append(errs, fmt.Errorf("from cell %q is not controlled by team %q", command.FromCellID, command.TeamID))
		}
		errs = append(errs, validateCommandTarget(state, command, toCell)...)
	}

	if teamIndex >= 0 {
		team := state.Teams[teamIndex]
		if command.Kind == CommandSupport {
			if team.SupportTokens <= 0 {
				errs = append(errs, fmt.Errorf("team %q has no support tokens", command.TeamID))
			}
		} else if cost := commandCostForTarget(command.Kind, state, toIndex); cost > 0 && team.FrontOpenPower < cost {
			errs = append(errs, fmt.Errorf("team %q needs %d front open power", command.TeamID, cost))
		}
	}

	if command.PlayerID != "" && state.CommandCooldownTicks > 0 {
		if lastTick, ok := state.PlayerLastCommandTick[command.PlayerID]; ok && state.Tick-lastTick < state.CommandCooldownTicks {
			errs = append(errs, fmt.Errorf("player %q is in command cooldown", command.PlayerID))
		}
	}
	if command.SitoneID != "" {
		if busyUntilTick := state.SitoneBusyUntilTick[command.SitoneID]; busyUntilTick > state.Tick {
			errs = append(errs, fmt.Errorf("sitone %q is busy until tick %d", command.SitoneID, busyUntilTick))
		}
	}

	return errors.Join(errs...)
}

func ApplyCommand(state State, command Command) (State, CommandResult, error) {
	next := cloneState(state)
	result, err := applyCommandInPlace(&next, command, time.Time{})
	if err != nil {
		return state, result, err
	}
	updateTeamRanks(&next)
	return next, result, nil
}

func AdvanceTick(state State, commands []Command, now time.Time) (State, []CommandResult) {
	next := cloneState(state)
	next.Tick++
	next.UpdatedAt = now

	releaseBusySitones(&next)
	activateDueEvents(&next)
	expireActiveEvents(&next)
	decayResources(&next)
	decayTeamOpenPower(&next)

	results := make([]CommandResult, 0, len(commands))
	for _, command := range commands {
		result, err := applyCommandInPlace(&next, command, now)
		if err != nil {
			result.Accepted = false
			result.Applied = false
			result.RejectReason = err.Error()
		}
		results = append(results, result)
	}

	refreshTeamDerived(&next)
	updateTeamRanks(&next)
	return next, results
}

func BuildLeaderboard(state State) []LeaderboardEntry {
	controlledCounts := controlledCellsByTeam(state.Cells)

	entries := make([]LeaderboardEntry, 0, len(state.Teams))
	for _, team := range state.Teams {
		entries = append(entries, LeaderboardEntry{
			TeamID:             team.TeamID,
			Color:              team.Color,
			PreviousRank:       team.Rank,
			Score:              team.Score,
			ControlledCells:    controlledCounts[team.TeamID],
			FrontOpenPower:     team.FrontOpenPower,
			RescuedSitones:     team.RescuedSitones,
			RepairedEvents:     team.RepairedEvents,
			CollaborationScore: team.CollaborationScore,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if entries[i].ControlledCells != entries[j].ControlledCells {
			return entries[i].ControlledCells > entries[j].ControlledCells
		}
		if entries[i].RepairedEvents != entries[j].RepairedEvents {
			return entries[i].RepairedEvents > entries[j].RepairedEvents
		}
		if entries[i].RescuedSitones != entries[j].RescuedSitones {
			return entries[i].RescuedSitones > entries[j].RescuedSitones
		}
		if entries[i].CollaborationScore != entries[j].CollaborationScore {
			return entries[i].CollaborationScore > entries[j].CollaborationScore
		}
		return entries[i].TeamID < entries[j].TeamID
	})

	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}

func CommandCost(kind string) int {
	return commandCosts[strings.TrimSpace(kind)]
}

func validateCommandTarget(state State, command Command, toCell Cell) []error {
	switch command.Kind {
	case CommandExpand:
		if toCell.OwnerTeamID != "" {
			return []error{fmt.Errorf("expand target %q must be neutral", toCell.ID)}
		}
	case CommandAttack:
		if toCell.OwnerTeamID == "" || toCell.OwnerTeamID == command.TeamID {
			return []error{fmt.Errorf("attack target %q must be controlled by another team", toCell.ID)}
		}
	case CommandReinforce:
		if toCell.OwnerTeamID != command.TeamID {
			return []error{fmt.Errorf("reinforce target %q must be controlled by team %q", toCell.ID, command.TeamID)}
		}
		if toCell.Control >= 100 && toCell.Defense >= maxDefense {
			return []error{fmt.Errorf("reinforce target %q is already at maximum defense", toCell.ID)}
		}
	case CommandRepair:
		if activeEventKindAtCell(state, toCell.ID) != content.FrontEventRepair {
			return []error{fmt.Errorf("repair target %q has no active repair event", toCell.ID)}
		}
	case CommandRescue:
		if activeEventKindAtCell(state, toCell.ID) != content.FrontEventRescue {
			return []error{fmt.Errorf("rescue target %q has no active rescue event", toCell.ID)}
		}
	case CommandScout, CommandSupport:
		return nil
	}
	return nil
}

func applyCommandInPlace(state *State, command Command, now time.Time) (CommandResult, error) {
	command = normalizeCommand(command)
	result := CommandResult{
		CommandID:      command.ID,
		PlayerID:       command.PlayerID,
		TeamID:         command.TeamID,
		Kind:           command.Kind,
		CellID:         command.ToCellID,
		ResolvedAtTick: state.Tick,
		ResolvedAt:     now,
	}

	if err := ValidateCommand(*state, command); err != nil {
		result.RejectReason = err.Error()
		return result, err
	}

	teamIndex := teamIndexByID(*state, command.TeamID)
	toIndex := cellIndexByID(*state, command.ToCellID)
	if teamIndex < 0 || toIndex < 0 {
		err := errors.New("validated command references missing state")
		result.RejectReason = err.Error()
		return result, err
	}

	team := &state.Teams[teamIndex]
	cell := &state.Cells[toIndex]

	if command.Kind == CommandSupport {
		team.SupportTokens--
	} else {
		team.FrontOpenPower -= commandCostForTarget(command.Kind, *state, toIndex)
	}
	team.LastCommandTick = state.Tick
	state.PlayerLastCommandTick[command.PlayerID] = state.Tick
	state.SitoneBusyUntilTick[command.SitoneID] = state.Tick + state.CommandResolveTicks

	result.Accepted = true
	result.Applied = true

	switch command.Kind {
	case CommandExpand:
		addPressure(cell, command.TeamID, 55)
		if pressureFor(cell, command.TeamID) >= 50 {
			result.CapturedCell = captureCell(cell, command.TeamID, 55, 8)
			result.ScoreDelta = 20 + collectCellResource(cell, team, &result)/2
		} else {
			result.ScoreDelta = 4
		}
	case CommandAttack:
		addPressure(cell, command.TeamID, 45)
		cell.Control = clampInt(cell.Control-35, 0, 100)
		cell.Defense = maxInt(0, cell.Defense-10)
		if cell.Control == 0 || pressureFor(cell, command.TeamID) >= 60 {
			result.CapturedCell = captureCell(cell, command.TeamID, 45, 6)
			result.ScoreDelta = 25 + collectCellResource(cell, team, &result)/2
		} else {
			result.ScoreDelta = 8
		}
	case CommandReinforce:
		cell.Control = clampInt(cell.Control+25, 0, 100)
		cell.Defense = clampInt(cell.Defense+12, 0, maxDefense)
		result.ScoreDelta = 5
	case CommandRepair:
		removeActiveEventAtCell(state, cell.ID)
		cell.EventID = ""
		cell.Control = clampInt(cell.Control+20, 0, 100)
		cell.Defense = clampInt(cell.Defense+10, 0, maxDefense)
		team.RepairedEvents++
		result.ScoreDelta = 30
	case CommandRescue:
		removeActiveEventAtCell(state, cell.ID)
		cell.EventID = ""
		team.RescuedSitones++
		result.ScoreDelta = 30
	case CommandScout:
		addPressure(cell, command.TeamID, 8)
		team.CollaborationScore++
		result.ScoreDelta = 3
	case CommandSupport:
		addPressure(cell, command.TeamID, 15)
		team.CollaborationScore += 3
		result.ScoreDelta = 3
	}

	team.Score += result.ScoreDelta
	refreshTeamDerived(state)
	return result, nil
}

func commandCostForTarget(kind string, state State, targetIndex int) int {
	cost := CommandCost(kind)
	if kind != CommandReinforce || targetIndex < 0 || targetIndex >= len(state.Cells) {
		return cost
	}
	target := state.Cells[targetIndex]
	controlApplied := minInt(25, maxInt(0, 100-target.Control))
	defenseApplied := minInt(12, maxInt(0, maxDefense-target.Defense))
	applied := controlApplied + defenseApplied
	if applied <= 0 {
		return 0
	}
	return maxInt(1, (cost*applied+37-1)/37)
}

func activateDueEvents(state *State) {
	activeByID := make(map[string]struct{}, len(state.ActiveEvents))
	for _, event := range state.ActiveEvents {
		activeByID[event.ID] = struct{}{}
	}
	for _, event := range state.EventSchedule {
		if event.StartsAtTick > state.Tick || event.ExpiresAtTick <= state.Tick {
			continue
		}
		if _, ok := activeByID[event.ID]; ok {
			continue
		}
		activateEvent(state, event)
		activeByID[event.ID] = struct{}{}
	}
}

func activateEvent(state *State, event Event) {
	cellIndex := cellIndexByID(*state, event.CellID)
	if cellIndex < 0 || state.Cells[cellIndex].EventID != "" {
		return
	}
	state.Cells[cellIndex].EventID = event.ID
	state.ActiveEvents = append(state.ActiveEvents, event)
}

func expireActiveEvents(state *State) {
	active := state.ActiveEvents[:0]
	for _, event := range state.ActiveEvents {
		if event.ExpiresAtTick > state.Tick {
			active = append(active, event)
			continue
		}

		cellIndex := cellIndexByID(*state, event.CellID)
		if cellIndex < 0 {
			continue
		}
		cell := &state.Cells[cellIndex]
		if cell.EventID == event.ID {
			cell.EventID = ""
		}
		applyExpiredEventPenalty(cell, event)
	}
	state.ActiveEvents = active
}

func applyExpiredEventPenalty(cell *Cell, event Event) {
	switch event.Kind {
	case content.FrontEventRepair, content.FrontEventPressure:
		cell.Control = clampInt(cell.Control-event.Severity*10, 0, 100)
		cell.Defense = maxInt(0, cell.Defense-event.Severity*5)
		if cell.Control == 0 {
			cell.OwnerTeamID = ""
		}
	case content.FrontEventResource:
		cell.Resource = maxInt(0, cell.Resource-event.Severity*10)
	case content.FrontEventRescue:
		cell.Defense = maxInt(0, cell.Defense-event.Severity*3)
	}
}

func decayResources(state *State) {
	if state.ResourceDecayPerTick <= 0 {
		return
	}
	for i := range state.Cells {
		state.Cells[i].Resource = maxInt(0, state.Cells[i].Resource-state.ResourceDecayPerTick)
	}
}

func decayTeamOpenPower(state *State) {
	if state.TeamOpenPowerDecayPerTick <= 0 {
		return
	}
	for i := range state.Teams {
		state.Teams[i].FrontOpenPower = maxInt(0, state.Teams[i].FrontOpenPower-state.TeamOpenPowerDecayPerTick)
	}
}

func releaseBusySitones(state *State) {
	for sitoneID, busyUntilTick := range state.SitoneBusyUntilTick {
		if busyUntilTick <= state.Tick {
			delete(state.SitoneBusyUntilTick, sitoneID)
		}
	}
}

func removeActiveEventAtCell(state *State, cellID string) {
	active := state.ActiveEvents[:0]
	for _, event := range state.ActiveEvents {
		if event.CellID == cellID {
			continue
		}
		active = append(active, event)
	}
	state.ActiveEvents = active
}

func collectCellResource(cell *Cell, team *TeamState, result *CommandResult) int {
	resource := cell.Resource
	if resource <= 0 {
		return 0
	}
	team.FrontOpenPower += resource
	result.ResourceDelta += resource
	cell.Resource = 0
	return resource
}

func captureCell(cell *Cell, teamID string, control int, defense int) bool {
	cell.OwnerTeamID = teamID
	cell.Control = clampInt(control, 1, 100)
	cell.Defense = defense
	cell.PressureByTeam = map[string]int{}
	return true
}

func addPressure(cell *Cell, teamID string, amount int) {
	if cell.PressureByTeam == nil {
		cell.PressureByTeam = make(map[string]int)
	}
	cell.PressureByTeam[teamID] += amount
}

func pressureFor(cell *Cell, teamID string) int {
	if cell.PressureByTeam == nil {
		return 0
	}
	return cell.PressureByTeam[teamID]
}

func refreshTeamDerived(state *State) {
	controlledCounts := controlledCellsByTeam(state.Cells)
	for i := range state.Teams {
		state.Teams[i].ControlledCells = controlledCounts[state.Teams[i].TeamID]
	}
}

func updateTeamRanks(state *State) {
	entries := BuildLeaderboard(*state)
	rankByTeam := make(map[string]int, len(entries))
	for _, entry := range entries {
		rankByTeam[entry.TeamID] = entry.Rank
	}
	for i := range state.Teams {
		state.Teams[i].PreviousRank = state.Teams[i].Rank
		state.Teams[i].Rank = rankByTeam[state.Teams[i].TeamID]
	}
}

func controlledCellsByTeam(cells []Cell) map[string]int {
	counts := make(map[string]int)
	for _, cell := range cells {
		if cell.OwnerTeamID == "" {
			continue
		}
		counts[cell.OwnerTeamID]++
	}
	return counts
}

func isPlayableStatus(status string) bool {
	switch status {
	case StatusOpenPlay, StatusSurge:
		return true
	default:
		return false
	}
}

func hasNeighbor(cell Cell, neighborID string) bool {
	for _, candidate := range cell.NeighborIDs {
		if candidate == neighborID {
			return true
		}
	}
	return false
}

func activeEventKindAtCell(state State, cellID string) string {
	for _, event := range state.ActiveEvents {
		if event.CellID == cellID {
			return event.Kind
		}
	}
	return ""
}

func normalizeCommand(command Command) Command {
	command.ID = strings.TrimSpace(command.ID)
	command.PlayerID = strings.TrimSpace(command.PlayerID)
	command.TeamID = strings.TrimSpace(command.TeamID)
	command.Kind = strings.TrimSpace(command.Kind)
	command.FromCellID = strings.TrimSpace(command.FromCellID)
	command.ToCellID = strings.TrimSpace(command.ToCellID)
	command.SitoneID = strings.TrimSpace(command.SitoneID)
	return command
}

func normalizeTeamSeeds(teams []TeamSeed) []TeamSeed {
	out := make([]TeamSeed, 0, len(teams))
	for _, team := range teams {
		team.ID = strings.TrimSpace(team.ID)
		team.Color = strings.TrimSpace(team.Color)
		out = append(out, team)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func validateTeamSeeds(teams []TeamSeed) error {
	var errs []error
	seen := make(map[string]struct{}, len(teams))
	for i, team := range teams {
		location := fmt.Sprintf("teams[%d]", i)
		if team.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", location))
			continue
		}
		if _, ok := seen[team.ID]; ok {
			errs = append(errs, fmt.Errorf("duplicate team id %q", team.ID))
			continue
		}
		seen[team.ID] = struct{}{}
		if team.FrontOpenPower < 0 {
			errs = append(errs, fmt.Errorf("%s.front_open_power must be greater than or equal to 0", location))
		}
		if team.SupportTokens < 0 {
			errs = append(errs, fmt.Errorf("%s.support_tokens must be greater than or equal to 0", location))
		}
	}
	return errors.Join(errs...)
}

func baseCellIDs(cells []content.FrontMapCell) []string {
	ids := make([]string, 0, len(cells))
	for _, cell := range cells {
		if strings.TrimSpace(cell.Terrain) == content.FrontTerrainBase || strings.TrimSpace(cell.Zone) == content.FrontZoneBase {
			ids = append(ids, strings.TrimSpace(cell.ID))
		}
	}
	sort.Strings(ids)
	return ids
}

func cellIndexByID(state State, id string) int {
	for i, cell := range state.Cells {
		if cell.ID == id {
			return i
		}
	}
	return -1
}

func teamIndexByID(state State, id string) int {
	for i, team := range state.Teams {
		if team.TeamID == id {
			return i
		}
	}
	return -1
}

func cloneState(state State) State {
	out := state
	out.Cells = make([]Cell, len(state.Cells))
	for i, cell := range state.Cells {
		out.Cells[i] = cell
		out.Cells[i].NeighborIDs = copyStrings(cell.NeighborIDs)
		out.Cells[i].PressureByTeam = copyIntMap(cell.PressureByTeam)
	}
	out.Teams = make([]TeamState, len(state.Teams))
	copy(out.Teams, state.Teams)
	out.EventSchedule = make([]Event, len(state.EventSchedule))
	copy(out.EventSchedule, state.EventSchedule)
	out.ActiveEvents = make([]Event, len(state.ActiveEvents))
	copy(out.ActiveEvents, state.ActiveEvents)
	out.PlayerLastCommandTick = copyInt64Map(state.PlayerLastCommandTick)
	out.SitoneBusyUntilTick = copyInt64Map(state.SitoneBusyUntilTick)
	return out
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func copyIntMap(values map[string]int) map[string]int {
	if len(values) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func copyInt64Map(values map[string]int64) map[string]int64 {
	if len(values) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
