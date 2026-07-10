package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const territoryMapFile = "campus_territory_v1.json"

const FrontMapModeTerritoryGrid = "territory_grid"

type TerritoryMapTemplate struct {
	SchemaVersion int                    `json:"schemaVersion"`
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	MapMode       string                 `json:"mapMode"`
	Background    TerritoryMapBackground `json:"background"`
	Grid          TerritoryMapGrid       `json:"grid"`
	BaseRules     TerritoryMapBaseRules  `json:"baseRules"`
	Bases         []TerritoryMapBase     `json:"bases"`
	Landmarks     []TerritoryMapLandmark `json:"landmarks"`
}

type TerritoryMapBaseRules struct {
	CoreDefense        int                          `json:"coreDefense"`
	CoreIndestructible bool                         `json:"coreIndestructible"`
	InitialTerritory   TerritoryMapInitialTerritory `json:"initialTerritory"`
}

type TerritoryMapInitialTerritory struct {
	Shape   string `json:"shape"`
	Radius  int    `json:"radius"`
	Defense int    `json:"defense"`
}

type TerritoryMapBackground struct {
	Src    string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type TerritoryMapGrid struct {
	Width        int                      `json:"width"`
	Height       int                      `json:"height"`
	Connectivity int                      `json:"connectivity"`
	Origin       TerritoryMapOrigin       `json:"origin"`
	PlayableMask TerritoryMapPlayableMask `json:"playableMask"`
}

type TerritoryMapOrigin struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type TerritoryMapPlayableMask struct {
	Encoding string   `json:"encoding"`
	Rows     []string `json:"rows"`
}

type TerritoryMapBase struct {
	TeamID        string `json:"teamId"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	CoreDefense   int    `json:"coreDefense"`
	InitialRadius int    `json:"initialRadius"`
}

type TerritoryMapLandmark struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Label           string `json:"label"`
	InitialDefense  int    `json:"initialDefense"`
	InitialResource int    `json:"initialResource"`
}

func loadTerritoryMap(dir string) (*TerritoryMapTemplate, error) {
	path := filepath.Join(dir, territoryMapFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: read: %w", path, err)
	}

	var template TerritoryMapTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("%s: decode json: %w", path, err)
	}
	if err := ValidateTerritoryMap(template); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	copy := copyTerritoryMapTemplate(template)
	return &copy, nil
}

func (s *Store) GetTerritoryMap(id string) (TerritoryMapTemplate, bool) {
	if s == nil || s.territoryMap == nil || s.territoryMap.ID != strings.TrimSpace(id) {
		return TerritoryMapTemplate{}, false
	}
	return copyTerritoryMapTemplate(*s.territoryMap), true
}

func (s *Store) TerritoryMap() (TerritoryMapTemplate, bool) {
	if s == nil || s.territoryMap == nil {
		return TerritoryMapTemplate{}, false
	}
	return copyTerritoryMapTemplate(*s.territoryMap), true
}

func ValidateTerritoryMap(template TerritoryMapTemplate) error {
	var errs []error
	if template.SchemaVersion != 1 {
		errs = append(errs, errors.New("schemaVersion must be 1"))
	}
	if strings.TrimSpace(template.ID) == "" {
		errs = append(errs, errors.New("id is required"))
	}
	if strings.TrimSpace(template.Name) == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if template.MapMode != FrontMapModeTerritoryGrid {
		errs = append(errs, fmt.Errorf("mapMode must be %q", FrontMapModeTerritoryGrid))
	}
	if strings.TrimSpace(template.Background.Src) == "" || template.Background.Width <= 0 || template.Background.Height <= 0 {
		errs = append(errs, errors.New("background src, width, and height are required"))
	}
	if template.Grid.Width != 64 || template.Grid.Height != 57 {
		errs = append(errs, errors.New("grid must be 64x57"))
	}
	if template.Grid.Connectivity != 4 {
		errs = append(errs, errors.New("grid connectivity must be 4"))
	}
	if template.BaseRules.CoreDefense != 100 {
		errs = append(errs, errors.New("baseRules.coreDefense must be 100"))
	}
	if !template.BaseRules.CoreIndestructible {
		errs = append(errs, errors.New("baseRules.coreIndestructible must be true"))
	}
	if template.BaseRules.InitialTerritory.Shape != "manhattan" {
		errs = append(errs, errors.New("baseRules.initialTerritory.shape must be manhattan"))
	}
	if template.BaseRules.InitialTerritory.Radius < 0 {
		errs = append(errs, errors.New("baseRules.initialTerritory.radius must be non-negative"))
	}
	if template.BaseRules.InitialTerritory.Defense < 0 || template.BaseRules.InitialTerritory.Defense > 100 {
		errs = append(errs, errors.New("baseRules.initialTerritory.defense must be between 0 and 100"))
	}
	if template.Grid.PlayableMask.Encoding != "bitmap-rows" {
		errs = append(errs, errors.New("playableMask.encoding must be bitmap-rows"))
	}
	if len(template.Grid.PlayableMask.Rows) != template.Grid.Height {
		errs = append(errs, fmt.Errorf("playableMask.rows must contain %d rows", template.Grid.Height))
	}
	for y, row := range template.Grid.PlayableMask.Rows {
		if len(row) != template.Grid.Width {
			errs = append(errs, fmt.Errorf("playableMask.rows[%d] must contain %d cells", y, template.Grid.Width))
			continue
		}
		for x := range row {
			if row[x] != '0' && row[x] != '1' {
				errs = append(errs, fmt.Errorf("playableMask.rows[%d][%d] must be 0 or 1", y, x))
			}
		}
	}

	baseTeams := make(map[string]struct{}, len(template.Bases))
	baseCells := make(map[[2]int]string, len(template.Bases))
	haloCells := make(map[[2]int]string, len(template.Bases)*5)
	for i, base := range template.Bases {
		if !isParticipatingTerritoryTeam(base.TeamID) {
			errs = append(errs, fmt.Errorf("bases[%d].teamId must be team-001 through team-009", i))
		} else if _, exists := baseTeams[base.TeamID]; exists {
			errs = append(errs, fmt.Errorf("bases contains duplicate teamId %q", base.TeamID))
		}
		baseTeams[base.TeamID] = struct{}{}
		if !territoryCoordinateIsPlayable(template.Grid, base.X, base.Y) {
			errs = append(errs, fmt.Errorf("bases[%d] must be on a playable cell", i))
		}
		if base.CoreDefense != 100 {
			errs = append(errs, fmt.Errorf("bases[%d].coreDefense must be 100", i))
		}
		if base.CoreDefense != template.BaseRules.CoreDefense {
			errs = append(errs, fmt.Errorf("bases[%d].coreDefense must match baseRules.coreDefense", i))
		}
		if base.InitialRadius != template.BaseRules.InitialTerritory.Radius {
			errs = append(errs, fmt.Errorf("bases[%d].initialRadius must match baseRules.initialTerritory.radius", i))
		}
		coordinate := [2]int{base.X, base.Y}
		if previous, exists := baseCells[coordinate]; exists {
			errs = append(errs, fmt.Errorf("bases[%d] overlaps %s", i, previous))
		}
		baseCells[coordinate] = base.TeamID
		for y := base.Y - base.InitialRadius; y <= base.Y+base.InitialRadius; y++ {
			for x := base.X - base.InitialRadius; x <= base.X+base.InitialRadius; x++ {
				if absTerritoryInt(x-base.X)+absTerritoryInt(y-base.Y) > base.InitialRadius {
					continue
				}
				if !territoryCoordinateIsPlayable(template.Grid, x, y) {
					errs = append(errs, fmt.Errorf("bases[%d] initial territory includes blocked cell (%d,%d)", i, x, y))
					continue
				}
				haloCoordinate := [2]int{x, y}
				if previous, exists := haloCells[haloCoordinate]; exists && previous != base.TeamID {
					errs = append(errs, fmt.Errorf("bases[%d] initial territory overlaps %s at (%d,%d)", i, previous, x, y))
				}
				haloCells[haloCoordinate] = base.TeamID
			}
		}
	}
	if len(baseTeams) != 9 {
		errs = append(errs, errors.New("bases must contain exactly one base for each of team-001 through team-009"))
	}

	landmarkIDs := make(map[string]struct{}, len(template.Landmarks))
	for i, landmark := range template.Landmarks {
		if strings.TrimSpace(landmark.ID) == "" || strings.TrimSpace(landmark.Kind) == "" || strings.TrimSpace(landmark.Label) == "" {
			errs = append(errs, fmt.Errorf("landmarks[%d] id, kind, and label are required", i))
		}
		if _, exists := landmarkIDs[landmark.ID]; exists {
			errs = append(errs, fmt.Errorf("landmarks contains duplicate id %q", landmark.ID))
		}
		landmarkIDs[landmark.ID] = struct{}{}
		if !territoryCoordinateIsPlayable(template.Grid, landmark.X, landmark.Y) {
			errs = append(errs, fmt.Errorf("landmarks[%d] must be on a playable cell", i))
		}
		if teamID, isBase := baseCells[[2]int{landmark.X, landmark.Y}]; isBase {
			errs = append(errs, fmt.Errorf("landmarks[%d] overlaps base for %s", i, teamID))
		}
		if landmark.InitialDefense < 0 || landmark.InitialDefense > 100 || landmark.InitialResource < 0 {
			errs = append(errs, fmt.Errorf("landmarks[%d] initialDefense/resource are invalid", i))
		}
	}
	return errors.Join(errs...)
}

func absTerritoryInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func territoryCoordinateIsPlayable(grid TerritoryMapGrid, x int, y int) bool {
	if x < 0 || y < 0 || x >= grid.Width || y >= grid.Height || y >= len(grid.PlayableMask.Rows) {
		return false
	}
	row := grid.PlayableMask.Rows[y]
	return x < len(row) && row[x] == '1'
}

func isParticipatingTerritoryTeam(teamID string) bool {
	if len(teamID) != len("team-001") || !strings.HasPrefix(teamID, "team-") {
		return false
	}
	number, err := strconv.Atoi(strings.TrimPrefix(teamID, "team-"))
	return err == nil && number >= 1 && number <= 9
}

func copyTerritoryMapTemplate(template TerritoryMapTemplate) TerritoryMapTemplate {
	template.Grid.PlayableMask.Rows = append([]string(nil), template.Grid.PlayableMask.Rows...)
	template.Bases = append([]TerritoryMapBase(nil), template.Bases...)
	template.Landmarks = append([]TerritoryMapLandmark(nil), template.Landmarks...)
	return template
}
