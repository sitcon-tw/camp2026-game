package communitystands

import mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"

const (
	rewardKindItem      = "item"
	rewardKindSitone    = "sitone"
	rewardKindOpenPower = "open_power"
)

type RewardResponse struct {
	Kind     string `json:"kind" example:"item"`
	RefID    string `json:"refId,omitempty" example:"item_booth_sticker"`
	Name     string `json:"name" example:"攤位貼紙"`
	Quantity int    `json:"quantity,omitempty" example:"1"`
	Amount   int    `json:"amount,omitempty" example:"50"`
	IconPath string `json:"iconPath,omitempty" example:"/game-icons/items/item_booth_sticker.png"`
}

type StandResponse struct {
	StandID     string         `json:"standId" example:"q7m4x2v9"`
	Name        string         `json:"name" example:"SITCON 社群攤位"`
	Description string         `json:"description" example:"介紹學生社群與開源參與方式。"`
	LogoURL     string         `json:"logoUrl,omitempty" example:"/game-icons/features/team.png"`
	WebsiteURL  string         `json:"websiteUrl,omitempty" example:"https://sitcon.org"`
	Reward      RewardResponse `json:"reward"`
}

type DetailResponse struct {
	Stand   StandResponse `json:"stand"`
	Claimed bool          `json:"claimed" example:"false"`
}

type ClaimResponse struct {
	ClaimID string         `json:"claimId" example:"community_claim_507f1f77bcf86cd799439011"`
	Stand   StandResponse  `json:"stand"`
	Reward  RewardResponse `json:"reward"`
	Claimed bool           `json:"claimed" example:"true"`
}

type DisplayResponse struct {
	Stand      StandResponse `json:"stand"`
	VisitCount int64         `json:"visitCount" example:"42"`
	ClaimCount int64         `json:"claimCount" example:"38"`
}

func standResponse(stand mongomodel.CommunityStand, reward RewardResponse) StandResponse {
	return StandResponse{
		StandID:     stand.ID,
		Name:        stand.Name,
		Description: stand.Description,
		LogoURL:     stand.LogoURL,
		WebsiteURL:  stand.WebsiteURL,
		Reward:      reward,
	}
}
