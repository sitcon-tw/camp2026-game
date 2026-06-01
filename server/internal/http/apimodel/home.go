package apimodel

type UserStateResponse struct {
	Player           UserStatePlayer `json:"player"`
	OpenPower        int             `json:"openPower" example:"1280"`
	SitoneCount      int             `json:"sitoneCount" example:"5"`
	ItemCount        int             `json:"itemCount" example:"3"`
	ActiveMatchCount int             `json:"activeMatchCount" example:"0"`
}

type UserStatePlayer struct {
	PlayerID  string          `json:"playerId" example:"7H9K2Q"`
	Nickname  string          `json:"nickname" example:"Alice"`
	Team      AuthTeamSummary `json:"team"`
	AvatarURL string          `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
}

type MeResponse struct {
	Player AuthPlayerSummary `json:"player"`
}

type OpenPowerBalanceResponse struct {
	Balance int `json:"balance" example:"1280"`
}

type OpenPowerRecordListResponse struct {
	Records []OpenPowerRecordSummary `json:"records"`
}

type OpenPowerRecordSummary struct {
	RecordID  string `json:"recordId" example:"N6T3ZA9K"`
	Amount    int    `json:"amount" example:"120"`
	Reason    string `json:"reason" example:"match_win"`
	Source    string `json:"source" example:"match"`
	CreatedAt string `json:"createdAt" example:"2026-07-24T10:35:00+08:00"`
}
