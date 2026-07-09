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
	CreatedAt    time.Time               `bson:"created_at,omitempty"`
	UpdatedAt    time.Time               `bson:"updated_at,omitempty"`
}

type FrontTeam struct {
	TeamID             string    `bson:"team_id"`
	Name               string    `bson:"name"`
	Color              string    `bson:"color,omitempty"`
	Score              int       `bson:"score,omitempty"`
	Rank               int       `bson:"rank,omitempty"`
	PreviousRank       int       `bson:"previous_rank,omitempty"`
	FrontOpenPower     int       `bson:"front_open_power,omitempty"`
	ControlledCells    int       `bson:"controlled_cells,omitempty"`
	RescuedSitones     int       `bson:"rescued_sitones,omitempty"`
	RepairedEvents     int       `bson:"repaired_events,omitempty"`
	CollaborationScore int       `bson:"collaboration_score,omitempty"`
	LastCommandAt      time.Time `bson:"last_command_at,omitempty"`
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
	ID              string         `bson:"_id"`
	ClientCommandID string         `bson:"client_command_id,omitempty"`
	FrontID         string         `bson:"front_id"`
	PlayerID        string         `bson:"player_id"`
	TeamID          string         `bson:"team_id,omitempty"`
	Kind            string         `bson:"kind"`
	Type            string         `bson:"type,omitempty"`
	FromCellID      string         `bson:"from_cell_id,omitempty"`
	ToCellID        string         `bson:"to_cell_id,omitempty"`
	SitoneID        string         `bson:"sitone_id,omitempty"`
	Payload         map[string]any `bson:"payload,omitempty"`
	Accepted        bool           `bson:"accepted,omitempty"`
	Applied         bool           `bson:"applied,omitempty"`
	RejectReason    string         `bson:"reject_reason,omitempty"`
	CreatedAt       time.Time      `bson:"created_at"`
}

type FrontCommandSummary struct {
	ID              string         `bson:"id"`
	ClientCommandID string         `bson:"client_command_id,omitempty"`
	Kind            string         `bson:"kind"`
	Type            string         `bson:"type,omitempty"`
	PlayerID        string         `bson:"player_id"`
	TeamID          string         `bson:"team_id,omitempty"`
	FromCellID      string         `bson:"from_cell_id,omitempty"`
	ToCellID        string         `bson:"to_cell_id,omitempty"`
	SitoneID        string         `bson:"sitone_id,omitempty"`
	Payload         map[string]any `bson:"payload,omitempty"`
	Accepted        bool           `bson:"accepted,omitempty"`
	Applied         bool           `bson:"applied,omitempty"`
	RejectReason    string         `bson:"reject_reason,omitempty"`
	CreatedAt       time.Time      `bson:"created_at"`
}
