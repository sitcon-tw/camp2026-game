package me

import "time"

type TeamResponse struct {
	TeamID string `json:"teamId" example:"8M4RXP"`
	Name   string `json:"name" example:"Blue Team"`
}

type TeamMemberResponse struct {
	PlayerID  string `json:"playerId" example:"7H9K2Q"`
	Nickname  string `json:"nickname" example:"Alice"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	Role      string `json:"role,omitempty" example:"staff"`
}

type StatusResponse struct {
	PlayerID    string               `json:"playerId" example:"7H9K2Q"`
	Nickname    string               `json:"nickname" example:"Alice"`
	Team        *TeamResponse        `json:"team,omitempty"`
	TeamMembers []TeamMemberResponse `json:"teamMembers"`
	OpenPower   int                  `json:"openPower" example:"1280"`
	AvatarURL   string               `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	Role        string               `json:"role,omitempty" example:"staff"`
}

type HomeResponse struct {
	Player   StatusResponse       `json:"player"`
	Summary  HomeSummaryResponse  `json:"summary"`
	TeamRank *TeamRankResponse    `json:"teamRank,omitempty"`
	Actions  []HomeActionResponse `json:"actions"`
}

type HomeSummaryResponse struct {
	OpenPower   int `json:"openPower" example:"1280"`
	SitoneCount int `json:"sitoneCount" example:"8"`
	ItemCount   int `json:"itemCount" example:"29"`
}

type TeamRankResponse struct {
	Rank          int    `json:"rank" example:"2"`
	TeamID        string `json:"teamId" example:"8M4RXP"`
	Name          string `json:"name" example:"Blue Team"`
	SitoneCount   int    `json:"sitoneCount" example:"18"`
	OpenPower     int    `json:"openPower" example:"1188"`
	GapToPrevious int    `json:"gapToPrevious" example:"3"`
}

type HomeActionResponse struct {
	ID      string `json:"id" example:"battle"`
	Label   string `json:"label" example:"知識王戰"`
	Enabled bool   `json:"enabled" example:"true"`
}

type QRCodeResponse struct {
	QRCodeToken string `json:"qrcodeToken" example:"qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok"`
}

type SitoneLoadoutRequest struct {
	SitoneIDs []string `json:"sitoneIds" validate:"required,min=1,max=5" example:"stone_engineering_base,stone_explorer_base"`
}

type SitoneLoadoutResponse struct {
	SitoneIDs []string `json:"sitoneIds" example:"stone_engineering_base,stone_explorer_base"`
}

type SitoneListResponse struct {
	Sitones []PlayerSitoneResponse `json:"sitones"`
}

type PlayerSitoneResponse struct {
	ID       string         `json:"id" example:"owned-sitone-001"`
	SitoneID string         `json:"sitoneId" example:"stone_engineering_base"`
	Quantity int            `json:"quantity" example:"1"`
	Sitone   SitoneResponse `json:"sitone"`
}

type SitoneResponse struct {
	ID                 string `json:"id" example:"stone_engineering_base"`
	Name               string `json:"name" example:"工程型小石"`
	Type               string `json:"type" example:"engineering"`
	Rarity             string `json:"rarity" example:"base"`
	Style              string `json:"style" example:"default"`
	Description        string `json:"description" example:"修 bug、分享解法、完成技術任務。"`
	IconPath           string `json:"iconPath,omitempty" example:"/game-icons/stones/basic_blue.png"`
	AbilityName        string `json:"abilityName" example:"穩定輸出"`
	AbilityKind        string `json:"abilityKind" example:"answer_score_bonus"`
	AbilityValue       int    `json:"abilityValue" example:"5"`
	AbilityCount       int    `json:"abilityCount" example:"0"`
	AbilityDescription string `json:"abilityDescription" example:"答對時分數提高 5%。"`
}

type ItemListResponse struct {
	Items []PlayerItemResponse `json:"items"`
}

type PlayerItemResponse struct {
	ID       string       `json:"id" example:"owned-item-001"`
	ItemID   string       `json:"itemId" example:"item_adventure_backpack"`
	Quantity int          `json:"quantity" example:"3"`
	Item     ItemResponse `json:"item"`
}

type ItemResponse struct {
	ID          string `json:"id" example:"item_adventure_backpack"`
	Name        string `json:"name" example:"冒險背包"`
	Type        string `json:"type" example:"material"`
	Rarity      string `json:"rarity" example:"common"`
	Description string `json:"description" example:"冒險背包，可用於小石合成。"`
	IconPath    string `json:"iconPath,omitempty" example:"/game-icons/items/item_adventure_backpack.png"`
	Source      string `json:"source,omitempty" example:"shop"`
}

type CompletedMatchListResponse struct {
	Matches    []CompletedMatchResponse `json:"matches"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
	Total      int                      `json:"total"`
	TotalPages int                      `json:"totalPages"`
}

type CompletedMatchResponse struct {
	MatchID       string                         `json:"matchId" example:"match_7H9K2Q"`
	Status        string                         `json:"status" example:"completed"`
	HostPlayerID  string                         `json:"hostPlayerId" example:"7H9K2Q"`
	Players       []CompletedMatchPlayerResponse `json:"players"`
	QuestionCount int                            `json:"questionCount" example:"10"`
	CreatedAt     time.Time                      `json:"createdAt"`
	StartedAt     *time.Time                     `json:"startedAt,omitempty"`
	CompletedAt   *time.Time                     `json:"completedAt,omitempty"`
}

type CompletedMatchPlayerResponse struct {
	PlayerID  string   `json:"playerId" example:"7H9K2Q"`
	Nickname  string   `json:"nickname" example:"Alice"`
	SitoneIDs []string `json:"sitoneIds,omitempty" example:"stone_engineering_base,stone_explorer_base"`
	Score     int      `json:"score" example:"850"`
}
