package staff

type CreateRewardRequest struct {
	QRCodeToken string `json:"qrcodeToken,omitempty" validate:"omitempty,min=4,max=512" example:"qr_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok"`
	PlayerID    string `json:"playerId,omitempty" validate:"omitempty,min=1,max=128" example:"7H9K2Q"`
	TeamID      string `json:"teamId,omitempty" validate:"omitempty,min=1,max=128" example:"8M4RXP"`
	AllPlayers  bool   `json:"allPlayers,omitempty" example:"false"`
	Kind        string `json:"kind" validate:"required,oneof=item sitone open_power" example:"sitone"`
	RefID       string `json:"refId,omitempty" validate:"omitempty,min=1,max=128" example:"stone_engineering_base"`
	Quantity    int    `json:"quantity,omitempty" validate:"omitempty,min=1,max=99" example:"1"`
	Amount      int    `json:"amount,omitempty" validate:"omitempty,min=1,max=99999" example:"100"`
}

type CreateRewardTokenRequest struct {
	Kind     string `json:"kind" validate:"required,oneof=item sitone open_power" example:"sitone"`
	RefID    string `json:"refId,omitempty" validate:"omitempty,min=1,max=128" example:"stone_engineering_base"`
	Quantity int    `json:"quantity,omitempty" validate:"omitempty,min=1,max=99" example:"1"`
	Amount   int    `json:"amount,omitempty" validate:"omitempty,min=1,max=99999" example:"100"`
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
	AvatarURL   string `json:"avatarUrl,omitempty" example:"https://example.test/avatar/blue.png"`
	MemberCount int    `json:"memberCount" example:"12"`
}

type CreateRewardResponse struct {
	RewardIDs    []string              `json:"rewardIds"`
	GrantedCount int                   `json:"grantedCount" example:"1"`
	AllPlayers   bool                  `json:"allPlayers,omitempty" example:"false"`
	Player       *RewardPlayerResponse `json:"player,omitempty"`
	Team         *RewardTeamResponse   `json:"team,omitempty"`
	Reward       RewardResponse        `json:"reward"`
}

type CreateRewardTokenResponse struct {
	Token     string         `json:"token" example:"srt_6H_x7lM20CK8BBnPfwEG1Ei97-PM9ZGr8Dy9yW-BYok"`
	ExpiresAt string         `json:"expiresAt" example:"2026-07-08T08:10:00Z"`
	Reward    RewardResponse `json:"reward"`
}

type ClaimRewardTokenResponse struct {
	RewardID string              `json:"rewardId" example:"staff_reward_507f1f77bcf86cd799439011"`
	Reward   RewardResponse      `json:"reward"`
	Staff    RewardStaffResponse `json:"staff"`
}

type RewardPlayerResponse struct {
	PlayerID  string              `json:"playerId" example:"7H9K2Q"`
	Nickname  string              `json:"nickname" example:"Alice"`
	AvatarURL string              `json:"avatarUrl,omitempty" example:"https://example.test/avatar/alice.png"`
	Team      *RewardTeamResponse `json:"team,omitempty"`
}

type RewardTeamResponse struct {
	TeamID string `json:"teamId" example:"8M4RXP"`
	Name   string `json:"name" example:"Blue Team"`
}

type RewardStaffResponse struct {
	PlayerID string `json:"playerId" example:"staff-token-1"`
	Nickname string `json:"nickname" example:"Staff"`
}

type RewardResponse struct {
	Kind     string `json:"kind" example:"sitone"`
	ID       string `json:"id,omitempty" example:"stone_engineering_base"`
	Name     string `json:"name" example:"工程型小石"`
	Quantity int    `json:"quantity,omitempty" example:"1"`
	Amount   int    `json:"amount,omitempty" example:"100"`
}
