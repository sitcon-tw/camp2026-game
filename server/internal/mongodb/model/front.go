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
	Teams        []FrontTeam             `bson:"teams,omitempty"`
	ActiveEvents []FrontMapEvent         `bson:"active_events,omitempty"`
	Leaderboard  []FrontLeaderboardEntry `bson:"leaderboard,omitempty"`
	LastCommand  *FrontCommandSummary    `bson:"last_command,omitempty"`
	Territory    *FrontTerritory         `bson:"territory,omitempty"`
	Garrisons    []FrontGarrison         `bson:"garrisons,omitempty"`
	RailSegments []FrontRailSegment      `bson:"rail_segments,omitempty"`
	TradeRoutes  []FrontTradeRoute       `bson:"trade_routes,omitempty"`
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
	X                    int       `bson:"x"`
	Length               int       `bson:"length"`
	OwnerTeamID          string    `bson:"owner_team_id,omitempty"`
	Defense              int       `bson:"defense,omitempty"`
	OccupiedAt           time.Time `bson:"occupied_at,omitempty"`
	HoldingRewardPeriods int       `bson:"holding_reward_periods,omitempty"`
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
	ControlledCells         int       `bson:"controlled_cells,omitempty"`
	MaxControlledCells      int       `bson:"max_controlled_cells,omitempty"`
	SitoneMilestonesReached int       `bson:"sitone_milestones_reached,omitempty"`
	NextSitoneMilestone     int       `bson:"next_sitone_milestone,omitempty"`
	RescuedSitones          int       `bson:"rescued_sitones,omitempty"`
	RepairedEvents          int       `bson:"repaired_events,omitempty"`
	CollaborationScore      int       `bson:"collaboration_score,omitempty"`
	TradeScore              int       `bson:"trade_score,omitempty"`
	TradeHourlyEarned       int       `bson:"trade_hourly_earned,omitempty"`
	TradeHourlyLimit        int       `bson:"trade_hourly_limit,omitempty"`
	TradeWindowStartedAt    time.Time `bson:"trade_window_started_at,omitempty"`
	LastCommandAt           time.Time `bson:"last_command_at,omitempty"`
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
	TradeScore         int    `bson:"trade_score,omitempty"`
}

type FrontGarrison struct {
	ID                string    `bson:"id"`
	PlayerID          string    `bson:"player_id"`
	TeamID            string    `bson:"team_id"`
	X                 int       `bson:"x"`
	Y                 int       `bson:"y"`
	SitoneIDs         []string  `bson:"sitone_ids"`
	DefenseBonus      int       `bson:"defense_bonus,omitempty"`
	TradeBonusPercent int       `bson:"trade_bonus_percent,omitempty"`
	StationedAt       time.Time `bson:"stationed_at"`
	LastTrainAt       time.Time `bson:"last_train_at,omitempty"`
}

type FrontRailSegment struct {
	ID               string `bson:"id"`
	SourceGarrisonID string `bson:"source_garrison_id"`
	TargetGarrisonID string `bson:"target_garrison_id"`
	SourceX          int    `bson:"source_x"`
	SourceY          int    `bson:"source_y"`
	TargetX          int    `bson:"target_x"`
	TargetY          int    `bson:"target_y"`
	Distance         int    `bson:"distance"`
}

type FrontTradeWaypoint struct {
	GarrisonID      string     `bson:"garrison_id"`
	TeamID          string     `bson:"team_id"`
	X               int        `bson:"x"`
	Y               int        `bson:"y"`
	ArrivesAt       time.Time  `bson:"arrives_at"`
	PotentialReward int        `bson:"potential_reward,omitempty"`
	SourceReward    int        `bson:"source_reward,omitempty"`
	TargetReward    int        `bson:"target_reward,omitempty"`
	SettledAt       *time.Time `bson:"settled_at,omitempty"`
}

type FrontTradeRoute struct {
	ID                 string               `bson:"id"`
	SourceGarrisonID   string               `bson:"source_garrison_id"`
	TargetGarrisonID   string               `bson:"target_garrison_id"`
	SourcePlayerID     string               `bson:"source_player_id"`
	TargetPlayerID     string               `bson:"target_player_id"`
	SourceTeamID       string               `bson:"source_team_id"`
	TargetTeamID       string               `bson:"target_team_id"`
	SourceX            int                  `bson:"source_x"`
	SourceY            int                  `bson:"source_y"`
	TargetX            int                  `bson:"target_x"`
	TargetY            int                  `bson:"target_y"`
	Distance           int                  `bson:"distance"`
	PotentialReward    int                  `bson:"potential_reward"`
	SourceReward       int                  `bson:"source_reward,omitempty"`
	TargetReward       int                  `bson:"target_reward,omitempty"`
	Waypoints          []FrontTradeWaypoint `bson:"waypoints,omitempty"`
	NextStopIndex      int                  `bson:"next_stop_index,omitempty"`
	Status             string               `bson:"status"`
	StartedAt          time.Time            `bson:"started_at"`
	ArrivesAt          time.Time            `bson:"arrives_at"`
	SettledAt          *time.Time           `bson:"settled_at,omitempty"`
	CancellationReason string               `bson:"cancellation_reason,omitempty"`
}

type FrontDisplacedGarrison struct {
	GarrisonID string   `bson:"garrison_id"`
	PlayerID   string   `bson:"player_id"`
	TeamID     string   `bson:"team_id"`
	X          int      `bson:"x"`
	Y          int      `bson:"y"`
	SitoneIDs  []string `bson:"sitone_ids"`
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
	ID                   string                   `bson:"_id"`
	ClientCommandID      string                   `bson:"client_command_id,omitempty"`
	FrontID              string                   `bson:"front_id"`
	PlayerID             string                   `bson:"player_id"`
	TeamID               string                   `bson:"team_id,omitempty"`
	Kind                 string                   `bson:"kind"`
	Type                 string                   `bson:"type,omitempty"`
	TargetX              *int                     `bson:"target_x,omitempty"`
	TargetY              *int                     `bson:"target_y,omitempty"`
	ExpectedRevision     *int64                   `bson:"expected_revision,omitempty"`
	AffectedCells        []FrontCoordinate        `bson:"affected_cells,omitempty"`
	CapturedCellCount    int                      `bson:"captured_cell_count,omitempty"`
	EnclosedCellCount    int                      `bson:"enclosed_cell_count,omitempty"`
	ScoreDelta           int                      `bson:"score_delta,omitempty"`
	FrontOpenPowerDelta  int                      `bson:"front_open_power_delta,omitempty"`
	FrontOpenPowerCost   int                      `bson:"front_open_power_cost,omitempty"`
	FrontOpenPowerReward int                      `bson:"front_open_power_reward,omitempty"`
	RewardSitoneID       string                   `bson:"reward_sitone_id,omitempty"`
	RewardSitoneQuantity int                      `bson:"reward_sitone_quantity,omitempty"`
	SitoneIDs            []string                 `bson:"sitone_ids,omitempty"`
	SitoneEffect         FrontSitoneEffect        `bson:"sitone_effect,omitempty"`
	GarrisonID           string                   `bson:"garrison_id,omitempty"`
	DisplacedGarrisons   []FrontDisplacedGarrison `bson:"displaced_garrisons,omitempty"`
	Payload              map[string]any           `bson:"payload,omitempty"`
	Accepted             bool                     `bson:"accepted,omitempty"`
	Applied              bool                     `bson:"applied,omitempty"`
	RejectReason         string                   `bson:"reject_reason,omitempty"`
	CreatedAt            time.Time                `bson:"created_at"`
}

type FrontSitoneEffect struct {
	SelectedCount        int `bson:"selected_count,omitempty"`
	SquadBonusPercent    int `bson:"squad_bonus_percent,omitempty"`
	AffinityBonusPercent int `bson:"affinity_bonus_percent,omitempty"`
	TotalBonusPercent    int `bson:"total_bonus_percent,omitempty"`
	AffectedCellBonus    int `bson:"affected_cell_bonus,omitempty"`
	DefenseBonus         int `bson:"defense_bonus,omitempty"`
	ScoreBonus           int `bson:"score_bonus,omitempty"`
}

type FrontCoordinate struct {
	X int `bson:"x"`
	Y int `bson:"y"`
}

type FrontCommandSummary struct {
	ID                   string                   `bson:"id"`
	ClientCommandID      string                   `bson:"client_command_id,omitempty"`
	Kind                 string                   `bson:"kind"`
	Type                 string                   `bson:"type,omitempty"`
	PlayerID             string                   `bson:"player_id"`
	TeamID               string                   `bson:"team_id,omitempty"`
	TargetX              *int                     `bson:"target_x,omitempty"`
	TargetY              *int                     `bson:"target_y,omitempty"`
	ExpectedRevision     *int64                   `bson:"expected_revision,omitempty"`
	AffectedCells        []FrontCoordinate        `bson:"affected_cells,omitempty"`
	CapturedCellCount    int                      `bson:"captured_cell_count,omitempty"`
	EnclosedCellCount    int                      `bson:"enclosed_cell_count,omitempty"`
	ScoreDelta           int                      `bson:"score_delta,omitempty"`
	FrontOpenPowerDelta  int                      `bson:"front_open_power_delta,omitempty"`
	FrontOpenPowerCost   int                      `bson:"front_open_power_cost,omitempty"`
	FrontOpenPowerReward int                      `bson:"front_open_power_reward,omitempty"`
	RewardSitoneID       string                   `bson:"reward_sitone_id,omitempty"`
	RewardSitoneQuantity int                      `bson:"reward_sitone_quantity,omitempty"`
	SitoneIDs            []string                 `bson:"sitone_ids,omitempty"`
	SitoneEffect         FrontSitoneEffect        `bson:"sitone_effect,omitempty"`
	GarrisonID           string                   `bson:"garrison_id,omitempty"`
	DisplacedGarrisons   []FrontDisplacedGarrison `bson:"displaced_garrisons,omitempty"`
	Payload              map[string]any           `bson:"payload,omitempty"`
	Accepted             bool                     `bson:"accepted,omitempty"`
	Applied              bool                     `bson:"applied,omitempty"`
	RejectReason         string                   `bson:"reject_reason,omitempty"`
	CreatedAt            time.Time                `bson:"created_at"`
}
