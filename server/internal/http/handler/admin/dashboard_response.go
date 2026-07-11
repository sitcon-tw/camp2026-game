package admin

import "time"

type DashboardResponse struct {
	GeneratedAt time.Time                   `json:"generatedAt"`
	Summary     DashboardSummaryResponse    `json:"summary"`
	TopPlayers  DashboardTopPlayersResponse `json:"topPlayers"`
	Teams       []DashboardTeamResponse     `json:"teams"`
	Players     []DashboardPlayerResponse   `json:"players"`
	Inventory   DashboardInventoryResponse  `json:"inventory"`
	Matches     DashboardMatchesResponse    `json:"matches"`
}

type DashboardHistoryResponse struct {
	GeneratedAt    time.Time                       `json:"generatedAt"`
	Bucket         string                          `json:"bucket" example:"hour"`
	SitoneBaseline int                             `json:"sitoneBaseline" example:"5"`
	OpenPowerStart int                             `json:"openPowerStart" example:"0"`
	Points         []DashboardHistoryPointResponse `json:"points"`
}

type DashboardHistoryPointResponse struct {
	Timestamp      time.Time `json:"timestamp"`
	SitoneCount    int       `json:"sitoneCount" example:"420"`
	OpenPower      int       `json:"openPower" example:"98220"`
	SitoneDelta    int       `json:"sitoneDelta" example:"3"`
	OpenPowerDelta int       `json:"openPowerDelta" example:"120"`
}

type DashboardSummaryResponse struct {
	PlayerCount          int     `json:"playerCount" example:"120"`
	StaffCount           int     `json:"staffCount" example:"18"`
	TeamCount            int     `json:"teamCount" example:"12"`
	UngroupedPlayerCount int     `json:"ungroupedPlayerCount" example:"3"`
	TotalSitones         int     `json:"totalSitones" example:"420"`
	TotalItems           int     `json:"totalItems" example:"188"`
	TotalOpenPower       int     `json:"totalOpenPower" example:"98220"`
	MedianOpenPower      float64 `json:"medianOpenPower" example:"818.5"`
	TotalMatches         int     `json:"totalMatches" example:"64"`
	WaitingMatches       int     `json:"waitingMatches" example:"2"`
	ActiveMatches        int     `json:"activeMatches" example:"4"`
	CompletedMatches     int     `json:"completedMatches" example:"58"`
	AnswerCount          int     `json:"answerCount" example:"240"`
	CorrectAnswerCount   int     `json:"correctAnswerCount" example:"180"`
	AnswerAccuracy       int     `json:"answerAccuracy" example:"75"`
	ShopPurchaseCount    int     `json:"shopPurchaseCount" example:"44"`
	FusionCount          int     `json:"fusionCount" example:"21"`
	StaffRewardCount     int     `json:"staffRewardCount" example:"88"`
	ItemDropCount        int     `json:"itemDropCount" example:"114"`
	DroppedItemCount     int     `json:"droppedItemCount" example:"39"`
}

type DashboardTopPlayersResponse struct {
	BySitones   []DashboardPlayerRankResponse `json:"bySitones"`
	ByOpenPower []DashboardPlayerRankResponse `json:"byOpenPower"`
	ByItems     []DashboardPlayerRankResponse `json:"byItems"`
	ByScore     []DashboardPlayerRankResponse `json:"byScore"`
	ByAccuracy  []DashboardPlayerRankResponse `json:"byAccuracy"`
}

type DashboardTeamSummaryResponse struct {
	TeamID    string `json:"teamId" example:"8M4RXP"`
	Name      string `json:"name" example:"Blue Team"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/blue.png"`
}

type DashboardPlayerResponse struct {
	Rank                int                           `json:"rank" example:"1"`
	PlayerID            string                        `json:"playerId" example:"7H9K2Q"`
	Nickname            string                        `json:"nickname" example:"Alice"`
	Team                *DashboardTeamSummaryResponse `json:"team,omitempty"`
	AvatarURL           string                        `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	Role                string                        `json:"role,omitempty" example:"player"`
	SitoneCount         int                           `json:"sitoneCount" example:"18"`
	ItemCount           int                           `json:"itemCount" example:"6"`
	CodexOwnedCount     int                           `json:"codexOwnedCount" example:"12"`
	CodexTotalCount     int                           `json:"codexTotalCount" example:"48"`
	CodexCompletion     int                           `json:"codexCompletion" example:"25"`
	ItemQuantities      map[string]int                `json:"itemQuantities"`
	OpenPower           int                           `json:"openPower" example:"1188"`
	MatchCount          int                           `json:"matchCount" example:"12"`
	CompletedMatchCount int                           `json:"completedMatchCount" example:"10"`
	AnswerCount         int                           `json:"answerCount" example:"40"`
	CorrectAnswerCount  int                           `json:"correctAnswerCount" example:"31"`
	AnswerAccuracy      int                           `json:"answerAccuracy" example:"78"`
	Score               int                           `json:"score" example:"6200"`
	LastActivityAt      *time.Time                    `json:"lastActivityAt,omitempty"`
}

type DashboardPlayerRankResponse struct {
	Rank                int                           `json:"rank" example:"1"`
	PlayerID            string                        `json:"playerId" example:"7H9K2Q"`
	Nickname            string                        `json:"nickname" example:"Alice"`
	Team                *DashboardTeamSummaryResponse `json:"team,omitempty"`
	AvatarURL           string                        `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	SitoneCount         int                           `json:"sitoneCount" example:"18"`
	ItemCount           int                           `json:"itemCount" example:"6"`
	OpenPower           int                           `json:"openPower" example:"1188"`
	MatchCount          int                           `json:"matchCount" example:"12"`
	CompletedMatchCount int                           `json:"completedMatchCount" example:"10"`
	AnswerCount         int                           `json:"answerCount" example:"40"`
	CorrectAnswerCount  int                           `json:"correctAnswerCount" example:"31"`
	AnswerAccuracy      int                           `json:"answerAccuracy" example:"78"`
	Score               int                           `json:"score" example:"6200"`
	LastActivityAt      *time.Time                    `json:"lastActivityAt,omitempty"`
}

type DashboardTeamResponse struct {
	Rank             int                          `json:"rank" example:"1"`
	TeamID           string                       `json:"teamId" example:"8M4RXP"`
	Name             string                       `json:"name" example:"Blue Team"`
	AvatarURL        string                       `json:"avatarUrl,omitempty" example:"https://example.test/avatar/blue.png"`
	PlayerCount      int                          `json:"playerCount" example:"10"`
	SitoneCount      int                          `json:"sitoneCount" example:"120"`
	ItemCount        int                          `json:"itemCount" example:"44"`
	OpenPower        int                          `json:"openPower" example:"24000"`
	AverageSitones   float64                      `json:"averageSitones" example:"12"`
	AverageItems     float64                      `json:"averageItems" example:"4.4"`
	AverageOpenPower float64                      `json:"averageOpenPower" example:"2400"`
	TopPlayer        *DashboardPlayerRankResponse `json:"topPlayer,omitempty"`
}

type DashboardInventoryResponse struct {
	Sitones []DashboardInventoryEntryResponse `json:"sitones"`
	Items   []DashboardInventoryEntryResponse `json:"items"`
}

type DashboardInventoryEntryResponse struct {
	ID             string `json:"id" example:"stone_engineering_base"`
	Name           string `json:"name" example:"工程型小石"`
	Type           string `json:"type,omitempty" example:"engineering"`
	Rarity         string `json:"rarity,omitempty" example:"base"`
	IconPath       string `json:"iconPath,omitempty" example:"/game-icons/stones/basic_blue.png"`
	Source         string `json:"source,omitempty" example:"drop"`
	Quantity       int    `json:"quantity" example:"30"`
	OwnerCount     int    `json:"ownerCount" example:"18"`
	OwnerPercent   int    `json:"ownerPercent" example:"15"`
	CatalogMissing bool   `json:"catalogMissing" example:"false"`
}

type DashboardMatchesResponse struct {
	Total                int                            `json:"total" example:"64"`
	Waiting              int                            `json:"waiting" example:"2"`
	Active               int                            `json:"active" example:"4"`
	Completed            int                            `json:"completed" example:"58"`
	PVP                  int                            `json:"pvp" example:"41"`
	Computer             int                            `json:"computer" example:"23"`
	AnswerCount          int                            `json:"answerCount" example:"240"`
	CorrectAnswerCount   int                            `json:"correctAnswerCount" example:"180"`
	AnswerAccuracy       int                            `json:"answerAccuracy" example:"75"`
	AverageScore         float64                        `json:"averageScore" example:"88.2"`
	AverageElapsedMillis float64                        `json:"averageElapsedMillis" example:"4200.5"`
	DropAttempts         int                            `json:"dropAttempts" example:"114"`
	DropSuccesses        int                            `json:"dropSuccesses" example:"39"`
	DropRate             int                            `json:"dropRate" example:"34"`
	Recent               []DashboardRecentMatchResponse `json:"recent"`
}

type DashboardRecentMatchResponse struct {
	MatchID        string     `json:"matchId" example:"M8RXP2"`
	Code           string     `json:"code,omitempty" example:"842913"`
	Mode           string     `json:"mode" example:"pvp"`
	Status         string     `json:"status" example:"completed"`
	PlayerCount    int        `json:"playerCount" example:"2"`
	WinnerPlayerID string     `json:"winnerPlayerId,omitempty" example:"7H9K2Q"`
	WinnerNickname string     `json:"winnerNickname,omitempty" example:"Alice"`
	TopScore       int        `json:"topScore" example:"600"`
	CreatedAt      *time.Time `json:"createdAt,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
}
