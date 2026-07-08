package leaderboards

type ListResponse struct {
	Scope         string              `json:"scope" example:"teams"`
	Entries       []RankEntryResponse `json:"entries"`
	CurrentEntry  *RankEntryResponse  `json:"currentEntry,omitempty"`
	GapToPrevious float64             `json:"gapToPrevious" example:"3"`
}

type RankEntryResponse struct {
	Rank             int     `json:"rank" example:"2"`
	ID               string  `json:"id" example:"8M4RXP"`
	Name             string  `json:"name" example:"Blue Team"`
	TeamID           string  `json:"teamId,omitempty" example:"8M4RXP"`
	TeamName         string  `json:"teamName,omitempty" example:"Blue Team"`
	AvatarURL        string  `json:"avatarUrl,omitempty" example:"/game-icons/stones/basic_blue.png"`
	PlayerCount      int     `json:"playerCount,omitempty" example:"10"`
	SitoneCount      int     `json:"sitoneCount" example:"18"`
	AverageSitones   float64 `json:"averageSitones,omitempty" example:"1.8"`
	OpenPower        int     `json:"openPower" example:"1188"`
	AverageOpenPower float64 `json:"averageOpenPower,omitempty" example:"118.8"`
	Current          bool    `json:"current" example:"true"`
}

type TeamSummaryResponse struct {
	TeamID    string `json:"teamId" example:"8M4RXP"`
	Name      string `json:"name" example:"Blue Team"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"/game-icons/stones/basic_blue.png"`
}

type TeamPlayersResponse struct {
	Team    TeamSummaryResponse         `json:"team"`
	Players []TeamPlayerSummaryResponse `json:"players"`
}

type TeamPlayerSummaryResponse struct {
	PlayerID    string `json:"playerId" example:"7H9K2Q"`
	Nickname    string `json:"nickname" example:"Alice"`
	AvatarURL   string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	SitoneCount int    `json:"sitoneCount" example:"18"`
	ItemCount   int    `json:"itemCount" example:"6"`
	OpenPower   int    `json:"openPower" example:"1188"`
	Current     bool   `json:"current" example:"true"`
}

type PlayerInventoryResponse struct {
	Player  InventoryPlayerResponse   `json:"player"`
	Team    TeamSummaryResponse       `json:"team"`
	Items   []InventoryItemResponse   `json:"items"`
	Sitones []InventorySitoneResponse `json:"sitones"`
}

type InventoryPlayerResponse struct {
	PlayerID    string `json:"playerId" example:"7H9K2Q"`
	Nickname    string `json:"nickname" example:"Alice"`
	AvatarURL   string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	SitoneCount int    `json:"sitoneCount" example:"18"`
	ItemCount   int    `json:"itemCount" example:"6"`
	OpenPower   int    `json:"openPower" example:"1188"`
	Current     bool   `json:"current" example:"true"`
}

type InventoryItemResponse struct {
	ID       string              `json:"id" example:"owned-item-001"`
	ItemID   string              `json:"itemId" example:"item_adventure_backpack"`
	Quantity int                 `json:"quantity" example:"3"`
	Item     InventoryItemDetail `json:"item"`
}

type InventoryItemDetail struct {
	ID          string `json:"id" example:"item_adventure_backpack"`
	Name        string `json:"name" example:"冒險背包"`
	Type        string `json:"type" example:"material"`
	Rarity      string `json:"rarity" example:"common"`
	Description string `json:"description" example:"冒險背包，可用於小石合成。"`
	IconPath    string `json:"iconPath,omitempty" example:"/game-icons/items/item_adventure_backpack.png"`
	Source      string `json:"source,omitempty" example:"shop"`
}

type InventorySitoneResponse struct {
	ID       string                `json:"id" example:"owned-sitone-001"`
	SitoneID string                `json:"sitoneId" example:"stone_engineering_base"`
	Quantity int                   `json:"quantity" example:"1"`
	Sitone   InventorySitoneDetail `json:"sitone"`
}

type InventorySitoneDetail struct {
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
