package content

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const frontMapsFile = "front_maps.toml"

const (
	FrontTerrainBase          = "base"
	FrontTerrainCourse        = "course"
	FrontTerrainSystem        = "system"
	FrontTerrainNeutral       = "neutral"
	FrontTerrainCenter        = "center"
	FrontTerrainCommunityDay5 = "community_day5"

	FrontZoneBase      = "base"
	FrontZoneCourse    = "course"
	FrontZoneSystem    = "system"
	FrontZoneFrontier  = "frontier"
	FrontZoneCommunity = "community"

	FrontEventRepair    = "repair"
	FrontEventRescue    = "rescue"
	FrontEventResource  = "resource"
	FrontEventChallenge = "challenge"
	FrontEventPressure  = "pressure"
)

var validFrontTerrains = map[string]struct{}{
	FrontTerrainBase:          {},
	FrontTerrainCourse:        {},
	FrontTerrainSystem:        {},
	FrontTerrainNeutral:       {},
	FrontTerrainCenter:        {},
	FrontTerrainCommunityDay5: {},
}

var validFrontZones = map[string]struct{}{
	FrontZoneBase:      {},
	FrontZoneCourse:    {},
	FrontZoneSystem:    {},
	FrontZoneFrontier:  {},
	FrontZoneCommunity: {},
}

var validFrontEventKinds = map[string]struct{}{
	FrontEventRepair:    {},
	FrontEventRescue:    {},
	FrontEventResource:  {},
	FrontEventChallenge: {},
	FrontEventPressure:  {},
}

type FrontMapTemplate struct {
	ID                    string                  `toml:"id"`
	Name                  string                  `toml:"name"`
	Enabled               bool                    `toml:"enabled"`
	TickSeconds           int                     `toml:"tick_seconds"`
	InitialFrontOpenPower int                     `toml:"initial_front_open_power"`
	CommandCooldownTicks  int64                   `toml:"command_cooldown_ticks"`
	CommandResolveTicks   int64                   `toml:"command_resolve_ticks"`
	ResourceDecayPerTick  int                     `toml:"resource_decay_per_tick"`
	Cells                 []FrontMapCell          `toml:"cells"`
	Events                []FrontMapEventTemplate `toml:"events"`
}

type FrontMapCell struct {
	ID              string   `toml:"id"`
	X               int      `toml:"x"`
	Y               int      `toml:"y"`
	Terrain         string   `toml:"terrain"`
	Zone            string   `toml:"zone"`
	Neighbors       []string `toml:"neighbors"`
	InitialControl  int      `toml:"initial_control"`
	InitialDefense  int      `toml:"initial_defense"`
	InitialResource int      `toml:"initial_resource"`
}

type FrontMapEventTemplate struct {
	ID                string `toml:"id"`
	CellID            string `toml:"cell_id"`
	Kind              string `toml:"kind"`
	Title             string `toml:"title"`
	StartsAtTick      int64  `toml:"starts_at_tick"`
	ExpiresAfterTicks int64  `toml:"expires_after_ticks"`
	Severity          int    `toml:"severity"`
}

type frontMapsDocument struct {
	FrontMaps []FrontMapTemplate `toml:"front_maps"`
}

func (s *Store) ListFrontMaps() []FrontMapTemplate {
	if s == nil || len(s.frontMaps) == 0 {
		return nil
	}

	frontMaps := make([]FrontMapTemplate, len(s.frontMaps))
	for i := range s.frontMaps {
		frontMaps[i] = copyFrontMapTemplate(s.frontMaps[i])
	}
	return frontMaps
}

func (s *Store) GetFrontMap(id string) (FrontMapTemplate, bool) {
	if s == nil {
		return FrontMapTemplate{}, false
	}

	template, ok := s.frontMapsByID[id]
	if !ok {
		return FrontMapTemplate{}, false
	}
	return copyFrontMapTemplate(template), true
}

func ValidateFrontMap(template FrontMapTemplate) error {
	_, err := validateFrontMap("front_map", template)
	return err
}

func validateFrontMaps(path string, templates []FrontMapTemplate) ([]FrontMapTemplate, map[string]FrontMapTemplate, error) {
	var errs []error
	seen := make(map[string]struct{}, len(templates))
	normalized := make([]FrontMapTemplate, 0, len(templates))

	if len(templates) == 0 {
		errs = append(errs, fmt.Errorf("%s: front_maps is required", path))
	}

	for i, template := range templates {
		template = normalizeFrontMapTemplate(template)
		location := fmt.Sprintf("%s: front_maps[%d]", path, i)

		if template.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", location))
		} else if _, ok := seen[template.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate front map id %q", path, template.ID))
		} else {
			seen[template.ID] = struct{}{}
		}

		template, err := validateFrontMap(location, template)
		if err != nil {
			errs = append(errs, err)
		}
		normalized = append(normalized, template)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].ID < normalized[j].ID
	})

	byID := make(map[string]FrontMapTemplate, len(normalized))
	for _, template := range normalized {
		byID[template.ID] = copyFrontMapTemplate(template)
	}
	return normalized, byID, nil
}

func validateFrontMap(location string, template FrontMapTemplate) (FrontMapTemplate, error) {
	template = normalizeFrontMapTemplate(template)

	var errs []error
	cellIDs := make(map[string]struct{}, len(template.Cells))
	neighborSets := make(map[string]map[string]struct{}, len(template.Cells))
	baseCellCount := 0

	if template.ID == "" {
		errs = append(errs, fmt.Errorf("%s.id is required", location))
	}
	if template.Name == "" {
		errs = append(errs, fmt.Errorf("%s.name is required", location))
	}
	if template.TickSeconds <= 0 {
		errs = append(errs, fmt.Errorf("%s.tick_seconds must be greater than 0", location))
	}
	if template.InitialFrontOpenPower < 0 {
		errs = append(errs, fmt.Errorf("%s.initial_front_open_power must be greater than or equal to 0", location))
	}
	if template.CommandCooldownTicks < 0 {
		errs = append(errs, fmt.Errorf("%s.command_cooldown_ticks must be greater than or equal to 0", location))
	}
	if template.CommandResolveTicks < 0 {
		errs = append(errs, fmt.Errorf("%s.command_resolve_ticks must be greater than or equal to 0", location))
	}
	if template.ResourceDecayPerTick < 0 {
		errs = append(errs, fmt.Errorf("%s.resource_decay_per_tick must be greater than or equal to 0", location))
	}
	if len(template.Cells) == 0 {
		errs = append(errs, fmt.Errorf("%s.cells is required", location))
	}

	for i, cell := range template.Cells {
		cellLocation := fmt.Sprintf("%s.cells[%d]", location, i)
		if cell.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", cellLocation))
		} else if _, ok := cellIDs[cell.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate cell id %q", location, cell.ID))
		} else {
			cellIDs[cell.ID] = struct{}{}
		}
		if cell.Terrain == "" {
			errs = append(errs, fmt.Errorf("%s.terrain is required", cellLocation))
		} else if _, ok := validFrontTerrains[cell.Terrain]; !ok {
			errs = append(errs, fmt.Errorf("%s.terrain must be one of %s", cellLocation, sortedKeys(validFrontTerrains)))
		}
		if cell.Zone == "" {
			errs = append(errs, fmt.Errorf("%s.zone is required", cellLocation))
		} else if _, ok := validFrontZones[cell.Zone]; !ok {
			errs = append(errs, fmt.Errorf("%s.zone must be one of %s", cellLocation, sortedKeys(validFrontZones)))
		}
		if cell.InitialControl < 0 || cell.InitialControl > 100 {
			errs = append(errs, fmt.Errorf("%s.initial_control must be between 0 and 100", cellLocation))
		}
		if cell.InitialDefense < 0 || cell.InitialDefense > 100 {
			errs = append(errs, fmt.Errorf("%s.initial_defense must be between 0 and 100", cellLocation))
		}
		if cell.InitialResource < 0 {
			errs = append(errs, fmt.Errorf("%s.initial_resource must be greater than or equal to 0", cellLocation))
		}
		if cell.Terrain == FrontTerrainBase || cell.Zone == FrontZoneBase {
			baseCellCount++
		}

		neighbors := make(map[string]struct{}, len(cell.Neighbors))
		for j, neighborID := range cell.Neighbors {
			neighborLocation := fmt.Sprintf("%s.neighbors[%d]", cellLocation, j)
			if neighborID == "" {
				errs = append(errs, fmt.Errorf("%s is required", neighborLocation))
				continue
			}
			if neighborID == cell.ID {
				errs = append(errs, fmt.Errorf("%s cannot reference itself", neighborLocation))
			}
			if _, ok := neighbors[neighborID]; ok {
				errs = append(errs, fmt.Errorf("%s contains duplicate neighbor %q", cellLocation, neighborID))
			}
			neighbors[neighborID] = struct{}{}
		}
		if cell.ID != "" {
			neighborSets[cell.ID] = neighbors
		}
	}

	if len(template.Cells) > 0 && baseCellCount == 0 {
		errs = append(errs, fmt.Errorf("%s.cells must include at least one base cell", location))
	}

	for cellID, neighbors := range neighborSets {
		for neighborID := range neighbors {
			reverse, ok := neighborSets[neighborID]
			if _, exists := cellIDs[neighborID]; !exists {
				errs = append(errs, fmt.Errorf("%s.cells[%s].neighbors references unknown cell %q", location, cellID, neighborID))
				continue
			}
			if ok {
				if _, linkedBack := reverse[cellID]; !linkedBack {
					errs = append(errs, fmt.Errorf("%s.cells[%s].neighbors must be bidirectional with %q", location, cellID, neighborID))
				}
			}
		}
	}

	eventIDs := make(map[string]struct{}, len(template.Events))
	for i, event := range template.Events {
		eventLocation := fmt.Sprintf("%s.events[%d]", location, i)
		if event.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", eventLocation))
		} else if _, ok := eventIDs[event.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate event id %q", location, event.ID))
		} else {
			eventIDs[event.ID] = struct{}{}
		}
		if event.CellID == "" {
			errs = append(errs, fmt.Errorf("%s.cell_id is required", eventLocation))
		} else if _, ok := cellIDs[event.CellID]; !ok {
			errs = append(errs, fmt.Errorf("%s.cell_id references unknown cell %q", eventLocation, event.CellID))
		}
		if event.Kind == "" {
			errs = append(errs, fmt.Errorf("%s.kind is required", eventLocation))
		} else if _, ok := validFrontEventKinds[event.Kind]; !ok {
			errs = append(errs, fmt.Errorf("%s.kind must be one of %s", eventLocation, sortedKeys(validFrontEventKinds)))
		}
		if event.Title == "" {
			errs = append(errs, fmt.Errorf("%s.title is required", eventLocation))
		}
		if event.StartsAtTick < 0 {
			errs = append(errs, fmt.Errorf("%s.starts_at_tick must be greater than or equal to 0", eventLocation))
		}
		if event.ExpiresAfterTicks <= 0 {
			errs = append(errs, fmt.Errorf("%s.expires_after_ticks must be greater than 0", eventLocation))
		}
		if event.Severity <= 0 {
			errs = append(errs, fmt.Errorf("%s.severity must be greater than 0", eventLocation))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return FrontMapTemplate{}, err
	}

	sort.Slice(template.Cells, func(i, j int) bool {
		return template.Cells[i].ID < template.Cells[j].ID
	})
	sort.Slice(template.Events, func(i, j int) bool {
		if template.Events[i].StartsAtTick != template.Events[j].StartsAtTick {
			return template.Events[i].StartsAtTick < template.Events[j].StartsAtTick
		}
		return template.Events[i].ID < template.Events[j].ID
	})

	return template, nil
}

func normalizeFrontMapTemplate(template FrontMapTemplate) FrontMapTemplate {
	template.ID = strings.TrimSpace(template.ID)
	template.Name = strings.TrimSpace(template.Name)
	for i := range template.Cells {
		template.Cells[i] = normalizeFrontMapCell(template.Cells[i])
	}
	for i := range template.Events {
		template.Events[i] = normalizeFrontMapEvent(template.Events[i])
	}
	return template
}

func normalizeFrontMapCell(cell FrontMapCell) FrontMapCell {
	cell.ID = strings.TrimSpace(cell.ID)
	cell.Terrain = strings.TrimSpace(cell.Terrain)
	cell.Zone = strings.TrimSpace(cell.Zone)
	for i := range cell.Neighbors {
		cell.Neighbors[i] = strings.TrimSpace(cell.Neighbors[i])
	}
	return cell
}

func normalizeFrontMapEvent(event FrontMapEventTemplate) FrontMapEventTemplate {
	event.ID = strings.TrimSpace(event.ID)
	event.CellID = strings.TrimSpace(event.CellID)
	event.Kind = strings.TrimSpace(event.Kind)
	event.Title = strings.TrimSpace(event.Title)
	return event
}

func copyFrontMapTemplate(template FrontMapTemplate) FrontMapTemplate {
	template.Cells = copyFrontMapCells(template.Cells)
	template.Events = copyFrontMapEvents(template.Events)
	return template
}

func copyFrontMapCells(cells []FrontMapCell) []FrontMapCell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]FrontMapCell, len(cells))
	copy(out, cells)
	for i := range out {
		out[i].Neighbors = copyStrings(out[i].Neighbors)
	}
	return out
}

func copyFrontMapEvents(events []FrontMapEventTemplate) []FrontMapEventTemplate {
	if len(events) == 0 {
		return nil
	}
	out := make([]FrontMapEventTemplate, len(events))
	copy(out, events)
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
