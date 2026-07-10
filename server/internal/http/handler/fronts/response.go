package fronts

import "time"

type CurrentResponse struct {
	Front *FrontSessionSummaryResponse `json:"front"`
}

type DetailResponse struct {
	Front                     FrontSessionSummaryResponse     `json:"front"`
	MapMode                   string                          `json:"mapMode,omitempty"`
	Grid                      *FrontTerritoryGridResponse     `json:"grid,omitempty"`
	TerritoryRows             []FrontTerritoryRowResponse     `json:"territoryRows,omitempty"`
	Bases                     []FrontTerritoryBaseResponse    `json:"bases,omitempty"`
	Landmarks                 []FrontMapLandmarkResponse      `json:"landmarks,omitempty"`
	CanPlay                   bool                            `json:"canPlay"`
	ServerTime                time.Time                       `json:"serverTime"`
	Cells                     []FrontCellResponse             `json:"cells"`
	Teams                     []FrontTeamResponse             `json:"teams"`
	ActiveEvents              []FrontMapEventResponse         `json:"activeEvents"`
	Leaderboard               []FrontLeaderboardEntryResponse `json:"leaderboard"`
	MyTeamID                  string                          `json:"myTeamId,omitempty"`
	MyTeamRank                int                             `json:"myTeamRank,omitempty"`
	Cooldowns                 map[string]int64                `json:"cooldowns"`
	SelectedSitones           []FrontSitoneResponse           `json:"selectedSitones"`
	AvailableSitones          []FrontSitoneResponse           `json:"availableSitones"`
	CurrentTeamFrontOpenPower int                             `json:"currentTeamFrontOpenPower,omitempty"`
	SupportTokens             int                             `json:"supportTokens,omitempty"`
	AvailableCommands         []FrontCommandOptionResponse    `json:"availableCommands"`
}

type LeaderboardResponse struct {
	FrontID string                          `json:"frontId" example:"camp_day1_training"`
	Entries []FrontLeaderboardEntryResponse `json:"entries"`
}

type CreateCommandRequest struct {
	ClientCommandID  string         `json:"clientCommandId,omitempty" example:"front_cmd_7H9K2Q_001"`
	Kind             string         `json:"kind,omitempty" example:"expand"`
	FromCellID       string         `json:"fromCellId,omitempty" example:"base_team_001"`
	ToCellID         string         `json:"toCellId,omitempty" example:"system_a"`
	TargetX          *int           `json:"targetX,omitempty" example:"24"`
	TargetY          *int           `json:"targetY,omitempty" example:"18"`
	ExpectedRevision *int64         `json:"expectedRevision,omitempty" example:"4"`
	SitoneID         string         `json:"sitoneId,omitempty" example:"stone_engineering_base"`
	Type             string         `json:"type,omitempty" example:"expand"`
	Payload          map[string]any `json:"payload,omitempty"`
}

type CreateCommandResponse struct {
	Accepted bool                 `json:"accepted" example:"true"`
	Command  FrontCommandResponse `json:"command"`
	Front    DetailResponse       `json:"front"`
}

type FrontSessionSummaryResponse struct {
	ID          string     `json:"id" example:"camp_day1_training"`
	MapID       string     `json:"mapId" example:"camp_day1_training"`
	MapMode     string     `json:"mapMode,omitempty" example:"territory_grid"`
	Status      string     `json:"status" example:"open_play"`
	Tick        int64      `json:"tick" example:"0"`
	Revision    int64      `json:"revision,omitempty" example:"1"`
	StartsAt    *time.Time `json:"startsAt,omitempty"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	FrozenAt    *time.Time `json:"frozenAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type FrontTerritoryGridResponse struct {
	Width        int                              `json:"width" example:"64"`
	Height       int                              `json:"height" example:"57"`
	Connectivity int                              `json:"connectivity" example:"4"`
	Origin       FrontTerritoryOriginResponse     `json:"origin"`
	Background   FrontTerritoryBackgroundResponse `json:"background"`
}

type FrontTerritoryOriginResponse struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type FrontTerritoryBackgroundResponse struct {
	Src    string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type FrontTerritoryRowResponse struct {
	Y    int                         `json:"y"`
	Runs []FrontTerritoryRunResponse `json:"runs"`
}

type FrontTerritoryRunResponse struct {
	X           int    `json:"x"`
	Length      int    `json:"length"`
	OwnerTeamID string `json:"ownerTeamId,omitempty"`
	Defense     int    `json:"defense"`
}

type FrontTerritoryBaseResponse struct {
	TeamID        string `json:"teamId"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	CoreDefense   int    `json:"coreDefense"`
	InitialRadius int    `json:"initialRadius"`
}

type FrontMapLandmarkResponse struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Label           string `json:"label"`
	InitialDefense  int    `json:"initialDefense,omitempty"`
	InitialResource int    `json:"initialResource,omitempty"`
}

type FrontCellResponse struct {
	ID             string         `json:"id" example:"base_team_001"`
	Name           string         `json:"name,omitempty" example:"Base Team 001"`
	X              int            `json:"x" example:"0"`
	Y              int            `json:"y" example:"0"`
	Terrain        string         `json:"terrain" example:"base"`
	Zone           string         `json:"zone" example:"base"`
	OwnerTeamID    string         `json:"ownerTeamId,omitempty" example:"team-blue"`
	Control        int            `json:"control" example:"100"`
	Defense        int            `json:"defense" example:"6"`
	Resource       int            `json:"resource" example:"40"`
	PressureByTeam map[string]int `json:"pressureByTeam"`
	NeighborIDs    []string       `json:"neighborIds"`
	EventID        string         `json:"eventId,omitempty" example:"evt_config_bug"`
	LockedUntil    *time.Time     `json:"lockedUntil,omitempty"`
}

type FrontTeamResponse struct {
	TeamID             string     `json:"teamId" example:"team-blue"`
	Name               string     `json:"name,omitempty" example:"Blue"`
	Color              string     `json:"color,omitempty" example:"#2563eb"`
	Score              int        `json:"score" example:"120"`
	Rank               int        `json:"rank" example:"1"`
	PreviousRank       int        `json:"previousRank,omitempty" example:"2"`
	FrontOpenPower     int        `json:"frontOpenPower" example:"100"`
	ControlledCells    int        `json:"controlledCells" example:"3"`
	RescuedSitones     int        `json:"rescuedSitones" example:"0"`
	RepairedEvents     int        `json:"repairedEvents" example:"0"`
	CollaborationScore int        `json:"collaborationScore" example:"0"`
	LastCommandAt      *time.Time `json:"lastCommandAt,omitempty"`
}

type FrontMapEventResponse struct {
	ID                string `json:"id" example:"evt_config_bug"`
	CellID            string `json:"cellId,omitempty" example:"repair_a"`
	Kind              string `json:"kind" example:"repair"`
	Title             string `json:"title" example:"Config repair"`
	StartsAtTick      int64  `json:"startsAtTick,omitempty" example:"0"`
	ExpiresAtTick     int64  `json:"expiresAtTick,omitempty" example:"300"`
	ExpiresAfterTicks int64  `json:"expiresAfterTicks,omitempty" example:"300"`
	Severity          int    `json:"severity,omitempty" example:"1"`
}

type FrontLeaderboardEntryResponse struct {
	TeamID             string `json:"teamId" example:"team-blue"`
	TeamName           string `json:"teamName,omitempty" example:"Blue"`
	Rank               int    `json:"rank" example:"1"`
	PreviousRank       int    `json:"previousRank,omitempty" example:"2"`
	Score              int    `json:"score" example:"120"`
	ControlledCells    int    `json:"controlledCells" example:"3"`
	RescuedSitones     int    `json:"rescuedSitones" example:"0"`
	RepairedEvents     int    `json:"repairedEvents" example:"0"`
	CollaborationScore int    `json:"collaborationScore" example:"0"`
	Current            bool   `json:"current,omitempty" example:"true"`
}

type FrontSitoneResponse struct {
	SitoneID               string `json:"sitoneId" example:"stone_engineering_base"`
	Name                   string `json:"name" example:"Engineering Stone"`
	Type                   string `json:"type,omitempty" example:"engineering"`
	IconPath               string `json:"iconPath,omitempty"`
	Available              bool   `json:"available" example:"true"`
	CooldownUntilTick      int64  `json:"cooldownUntilTick,omitempty"`
	RemainingCooldownTicks int64  `json:"remainingCooldownTicks,omitempty"`
	AssignedCellID         string `json:"assignedCellId,omitempty"`
}

type FrontCommandOptionResponse struct {
	Kind       string `json:"kind" example:"expand"`
	Label      string `json:"label,omitempty" example:"Expand"`
	Enabled    bool   `json:"enabled" example:"true"`
	Cost       int    `json:"cost,omitempty" example:"5"`
	FromCellID string `json:"fromCellId,omitempty"`
	ToCellID   string `json:"toCellId,omitempty"`
	TargetX    *int   `json:"targetX,omitempty"`
	TargetY    *int   `json:"targetY,omitempty"`
	SitoneID   string `json:"sitoneId,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type FrontCommandResponse struct {
	CommandID        string                    `json:"commandId" example:"front_command_507f1f77bcf86cd799439011"`
	ClientCommandID  string                    `json:"clientCommandId,omitempty" example:"front_cmd_7H9K2Q_001"`
	Kind             string                    `json:"kind" example:"expand"`
	Type             string                    `json:"type,omitempty" example:"expand"`
	PlayerID         string                    `json:"playerId" example:"7H9K2Q"`
	TeamID           string                    `json:"teamId,omitempty" example:"team-blue"`
	FromCellID       string                    `json:"fromCellId,omitempty" example:"base_team_001"`
	ToCellID         string                    `json:"toCellId,omitempty" example:"system_a"`
	TargetX          *int                      `json:"targetX,omitempty"`
	TargetY          *int                      `json:"targetY,omitempty"`
	ExpectedRevision *int64                    `json:"expectedRevision,omitempty"`
	AffectedCells    []FrontCoordinateResponse `json:"affectedCells,omitempty"`
	SitoneID         string                    `json:"sitoneId,omitempty" example:"stone_engineering_base"`
	Payload          map[string]any            `json:"payload,omitempty"`
	Accepted         bool                      `json:"accepted" example:"true"`
	Applied          bool                      `json:"applied" example:"true"`
	RejectReason     string                    `json:"rejectReason,omitempty"`
	CreatedAt        time.Time                 `json:"createdAt"`
}

type FrontCoordinateResponse struct {
	X int `json:"x"`
	Y int `json:"y"`
}
