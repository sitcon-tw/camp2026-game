package model

import "time"

const (
	FrontsCollection        = "fronts"
	FrontCommandsCollection = "front_commands"
)

const (
	FrontStatusOpenPlay = "open_play"
	FrontStatusActive   = FrontStatusOpenPlay
)

type Front struct {
	ID           string                  `bson:"_id"`
	MapID        string                  `bson:"map_id,omitempty"`
	MapMode      string                  `bson:"map_mode,omitempty"`
	Name         string                  `bson:"name"`
	Status       string                  `bson:"status"`
	Current      bool                    `bson:"current,omitempty"`
	Tick         int64                   `bson:"tick,omitempty"`
	Revision     int64                   `bson:"revision,omitempty"`
	Cells        []FrontCell             `bson:"cells,omitempty"`
	Teams        []FrontTeam             `bson:"teams,omitempty"`
	Zones        []FrontZone             `bson:"zones,omitempty"`
	ActiveEvents []FrontMapEvent         `bson:"active_events,omitempty"`
	Leaderboard  []FrontLeaderboardEntry `bson:"leaderboard,omitempty"`
	LastCommand  *FrontCommandSummary    `bson:"last_command,omitempty"`
	Territory    *FrontTerritory         `bson:"territory,omitempty"`
	CreatedAt    time.Time               `bson:"created_at,omitempty"`
	UpdatedAt    time.Time               `bson:"updated_at,omitempty"`
}

type FrontTerritory struct {
	Width            int                  `bson:"width"`
	Height           int                  `bson:"height"`
	Connectivity     int                  `bson:"connectivity"`
	OriginX          int                  `bson:"origin_x,omitempty"`
	OriginY          int                  `bson:"origin_y,omitempty"`
	BackgroundSrc    string               `bson:"background_src,omitempty"`
	BackgroundWidth  int                  `bson:"background_width,omitempty"`
	BackgroundHeight int                  `bson:"background_height,omitempty"`
	Rows             []FrontTerritoryRow  `bson:"rows"`
	Bases            []FrontTerritoryBase `bson:"bases"`
	Landmarks        []FrontMapLandmark   `bson:"landmarks,omitempty"`
}

// FrontTerritoryRow stores every playable cell as a compact sequence of runs.
// Missing coordinates are water or outside the campus boundary.
type FrontTerritoryRow struct {
	Y    int                 `bson:"y"`
	Runs []FrontTerritoryRun `bson:"runs"`
}

type FrontTerritoryRun struct {
	X           int    `bson:"x"`
	Length      int    `bson:"length"`
	OwnerTeamID string `bson:"owner_team_id,omitempty"`
	Defense     int    `bson:"defense,omitempty"`
}

type FrontTerritoryBase struct {
	TeamID        string `bson:"team_id"`
	X             int    `bson:"x"`
	Y             int    `bson:"y"`
	CoreDefense   int    `bson:"core_defense"`
	InitialRadius int    `bson:"initial_radius,omitempty"`
}

type FrontMapLandmark struct {
	ID              string `bson:"id"`
	Kind            string `bson:"kind"`
	X               int    `bson:"x"`
	Y               int    `bson:"y"`
	Label           string `bson:"label"`
	InitialDefense  int    `bson:"initial_defense,omitempty"`
	InitialResource int    `bson:"initial_resource,omitempty"`
}

type FrontTeam struct {
	TeamID                  string    `bson:"team_id"`
	Name                    string    `bson:"name"`
	Color                   string    `bson:"color,omitempty"`
	Score                   int       `bson:"score,omitempty"`
	Rank                    int       `bson:"rank,omitempty"`
	PreviousRank            int       `bson:"previous_rank,omitempty"`
	FrontOpenPower          int       `bson:"front_open_power,omitempty"`
	EmergencyResupplies     int       `bson:"emergency_resupplies,omitempty"`
	ControlledCells         int       `bson:"controlled_cells,omitempty"`
	MaxControlledCells      int       `bson:"max_controlled_cells,omitempty"`
	SitoneMilestonesReached int       `bson:"sitone_milestones_reached,omitempty"`
	NextSitoneMilestone     int       `bson:"next_sitone_milestone,omitempty"`
	RescuedSitones          int       `bson:"rescued_sitones,omitempty"`
	RepairedEvents          int       `bson:"repaired_events,omitempty"`
	CollaborationScore      int       `bson:"collaboration_score,omitempty"`
	LastCommandAt           time.Time `bson:"last_command_at,omitempty"`
}

type FrontCell struct {
	ID             string         `bson:"id"`
	Name           string         `bson:"name,omitempty"`
	X              int            `bson:"x"`
	Y              int            `bson:"y"`
	Terrain        string         `bson:"terrain"`
	Zone           string         `bson:"zone"`
	OwnerTeamID    string         `bson:"owner_team_id,omitempty"`
	Control        int            `bson:"control,omitempty"`
	Defense        int            `bson:"defense,omitempty"`
	Resource       int            `bson:"resource,omitempty"`
	PressureByTeam map[string]int `bson:"pressure_by_team,omitempty"`
	NeighborIDs    []string       `bson:"neighbor_ids,omitempty"`
	EventID        string         `bson:"event_id,omitempty"`
	LockedUntil    time.Time      `bson:"locked_until,omitempty"`
}

type FrontZone struct {
	ID          string `bson:"id"`
	Name        string `bson:"name"`
	OwnerTeamID string `bson:"owner_team_id,omitempty"`
	Points      int    `bson:"points,omitempty"`
}

type FrontLeaderboardEntry struct {
	Rank               int    `bson:"rank,omitempty"`
	PreviousRank       int    `bson:"previous_rank,omitempty"`
	TeamID             string `bson:"team_id"`
	TeamName           string `bson:"team_name"`
	Score              int    `bson:"score"`
	ControlledCells    int    `bson:"controlled_cells,omitempty"`
	RescuedSitones     int    `bson:"rescued_sitones,omitempty"`
	RepairedEvents     int    `bson:"repaired_events,omitempty"`
	CollaborationScore int    `bson:"collaboration_score,omitempty"`
}

type FrontMapEvent struct {
	ID                string `bson:"id"`
	CellID            string `bson:"cell_id,omitempty"`
	Kind              string `bson:"kind"`
	Title             string `bson:"title"`
	StartsAtTick      int64  `bson:"starts_at_tick,omitempty"`
	ExpiresAtTick     int64  `bson:"expires_at_tick,omitempty"`
	ExpiresAfterTicks int64  `bson:"expires_after_ticks,omitempty"`
	Severity          int    `bson:"severity,omitempty"`
}

type FrontCommand struct {
	ID                      string            `bson:"_id"`
	ClientCommandID         string            `bson:"client_command_id,omitempty"`
	FrontID                 string            `bson:"front_id"`
	PlayerID                string            `bson:"player_id"`
	TeamID                  string            `bson:"team_id,omitempty"`
	Kind                    string            `bson:"kind"`
	Type                    string            `bson:"type,omitempty"`
	FromCellID              string            `bson:"from_cell_id,omitempty"`
	ToCellID                string            `bson:"to_cell_id,omitempty"`
	TargetX                 *int              `bson:"target_x,omitempty"`
	TargetY                 *int              `bson:"target_y,omitempty"`
	ExpectedRevision        *int64            `bson:"expected_revision,omitempty"`
	AffectedCells           []FrontCoordinate `bson:"affected_cells,omitempty"`
	CapturedCellCount       int               `bson:"captured_cell_count,omitempty"`
	EnclosedCellCount       int               `bson:"enclosed_cell_count,omitempty"`
	ScoreDelta              int               `bson:"score_delta,omitempty"`
	FrontOpenPowerDelta     int               `bson:"front_open_power_delta,omitempty"`
	FrontOpenPowerCost      int               `bson:"front_open_power_cost,omitempty"`
	EmergencyResupplyAmount int               `bson:"emergency_resupply_amount,omitempty"`
	RewardSitoneID          string            `bson:"reward_sitone_id,omitempty"`
	RewardSitoneQuantity    int               `bson:"reward_sitone_quantity,omitempty"`
	SitoneID                string            `bson:"sitone_id,omitempty"`
	Payload                 map[string]any    `bson:"payload,omitempty"`
	Accepted                bool              `bson:"accepted,omitempty"`
	Applied                 bool              `bson:"applied,omitempty"`
	RejectReason            string            `bson:"reject_reason,omitempty"`
	CreatedAt               time.Time         `bson:"created_at"`
}

type FrontCoordinate struct {
	X int `bson:"x"`
	Y int `bson:"y"`
}

type FrontCommandSummary struct {
	ID                      string            `bson:"id"`
	ClientCommandID         string            `bson:"client_command_id,omitempty"`
	Kind                    string            `bson:"kind"`
	Type                    string            `bson:"type,omitempty"`
	PlayerID                string            `bson:"player_id"`
	TeamID                  string            `bson:"team_id,omitempty"`
	FromCellID              string            `bson:"from_cell_id,omitempty"`
	ToCellID                string            `bson:"to_cell_id,omitempty"`
	TargetX                 *int              `bson:"target_x,omitempty"`
	TargetY                 *int              `bson:"target_y,omitempty"`
	ExpectedRevision        *int64            `bson:"expected_revision,omitempty"`
	AffectedCells           []FrontCoordinate `bson:"affected_cells,omitempty"`
	CapturedCellCount       int               `bson:"captured_cell_count,omitempty"`
	EnclosedCellCount       int               `bson:"enclosed_cell_count,omitempty"`
	ScoreDelta              int               `bson:"score_delta,omitempty"`
	FrontOpenPowerDelta     int               `bson:"front_open_power_delta,omitempty"`
	FrontOpenPowerCost      int               `bson:"front_open_power_cost,omitempty"`
	EmergencyResupplyAmount int               `bson:"emergency_resupply_amount,omitempty"`
	RewardSitoneID          string            `bson:"reward_sitone_id,omitempty"`
	RewardSitoneQuantity    int               `bson:"reward_sitone_quantity,omitempty"`
	SitoneID                string            `bson:"sitone_id,omitempty"`
	Payload                 map[string]any    `bson:"payload,omitempty"`
	Accepted                bool              `bson:"accepted,omitempty"`
	Applied                 bool              `bson:"applied,omitempty"`
	RejectReason            string            `bson:"reject_reason,omitempty"`
	CreatedAt               time.Time         `bson:"created_at"`
}
