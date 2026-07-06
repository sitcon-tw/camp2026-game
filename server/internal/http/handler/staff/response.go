package staff

type CreateRewardRequest struct {
	QRCodeToken string `json:"qrcodeToken,omitempty" validate:"omitempty,min=4,max=512" example:"qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok"`
	PlayerID    string `json:"playerId,omitempty" validate:"omitempty,min=1,max=128" example:"7H9K2Q"`
	TeamID      string `json:"teamId,omitempty" validate:"omitempty,min=1,max=128" example:"8M4RXP"`
	Kind        string `json:"kind" validate:"required,oneof=item sitone" example:"sitone"`
	RefID       string `json:"refId" validate:"required,min=1,max=128" example:"stone_engineering_base"`
	Quantity    int    `json:"quantity" validate:"required,min=1,max=99" example:"1"`
}

type ListPlayersResponse struct {
	Players []StaffPlayerResponse `json:"players"`
}

type ListTeamsResponse struct {
	Teams []StaffTeamResponse `json:"teams"`
}

type StaffPlayerResponse struct {
	PlayerID  string              `json:"playerId" example:"7H9K2Q"`
	Nickname  string              `json:"nickname" example:"Alice"`
	Team      *RewardTeamResponse `json:"team,omitempty"`
	AvatarURL string              `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
}

type StaffTeamResponse struct {
	TeamID      string `json:"teamId" example:"8M4RXP"`
	Name        string `json:"name" example:"Blue Team"`
	MemberCount int    `json:"memberCount" example:"12"`
}

type CreateRewardResponse struct {
	RewardIDs    []string              `json:"rewardIds"`
	GrantedCount int                   `json:"grantedCount" example:"1"`
	Player       *RewardPlayerResponse `json:"player,omitempty"`
	Team         *RewardTeamResponse   `json:"team,omitempty"`
	Reward       RewardResponse        `json:"reward"`
}

type RewardPlayerResponse struct {
	PlayerID string              `json:"playerId" example:"7H9K2Q"`
	Nickname string              `json:"nickname" example:"Alice"`
	Team     *RewardTeamResponse `json:"team,omitempty"`
}

type RewardTeamResponse struct {
	TeamID string `json:"teamId" example:"8M4RXP"`
	Name   string `json:"name" example:"Blue Team"`
}

type RewardResponse struct {
	Kind     string `json:"kind" example:"sitone"`
	ID       string `json:"id" example:"stone_engineering_base"`
	Name     string `json:"name" example:"工程型小石"`
	Quantity int    `json:"quantity" example:"1"`
}
