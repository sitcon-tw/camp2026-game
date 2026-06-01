package apimodel

type AuthLoginRequest struct {
	Token string `json:"token" validate:"required,min=16,max=512" example:"eyJjYW1wMjAyNiI6InBsYXllci0wMSJ9"`
}

type AuthLoginResponse struct {
	Player AuthPlayerSummary `json:"player"`
}

type AuthPlayerSummary struct {
	PlayerID  string          `json:"playerId" example:"7H9K2Q"`
	Nickname  string          `json:"nickname" example:"Alice"`
	Team      AuthTeamSummary `json:"team"`
	OpenPower int             `json:"openPower" example:"1280"`
	AvatarURL string          `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
}

type AuthTeamSummary struct {
	TeamID string `json:"teamId" example:"8M4RXP"`
	Name   string `json:"name" example:"Blue Team"`
}
