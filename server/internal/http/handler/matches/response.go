package matches

import "time"

type CreateMatchResponse = MatchStateResponse
type JoinMatchResponse = MatchStateResponse
type OpenMatchResponse = MatchStateResponse
type ReadyMatchResponse = MatchStateResponse
type UpdateLoadoutResponse = MatchStateResponse

type JoinMatchRequest struct {
	Code string `json:"code" validate:"required,min=4,max=16" example:"ABC123"`
}

type UpdateLoadoutRequest struct {
	SitoneIDs []string `json:"sitoneIds" validate:"required,min=1,max=5" example:"stone_engineering_base,stone_explorer_base"`
}

type AnswerRequest struct {
	QuestionID string `json:"questionId" validate:"required" example:"quiz-001"`
	Choice     string `json:"choice" validate:"required,oneof=A B C D" example:"A"`
}

type AnswerAcceptedResponse struct {
	Accepted bool `json:"accepted" example:"true"`
}

type ComputerBattleSettingsResponse struct {
	Enabled bool `json:"enabled" example:"true"`
}

type MatchStateResponse struct {
	MatchID               string                 `json:"matchId" example:"match_7H9K2Q"`
	Code                  string                 `json:"code,omitempty" example:"ABC123"`
	Mode                  string                 `json:"mode" example:"pvp"`
	Status                string                 `json:"status" example:"active"`
	Phase                 string                 `json:"phase,omitempty" example:"answering"`
	HostPlayerID          string                 `json:"hostPlayerId" example:"7H9K2Q"`
	Players               []MatchPlayerResponse  `json:"players"`
	CurrentQuestionIndex  int                    `json:"currentQuestionIndex,omitempty" example:"0"`
	QuestionCount         int                    `json:"questionCount,omitempty" example:"10"`
	CurrentQuestion       *MatchQuestionResponse `json:"currentQuestion,omitempty"`
	CurrentQuestionResult *MatchQuestionResult   `json:"currentQuestionResult,omitempty"`
	RoundStartedAt        *time.Time             `json:"roundStartedAt,omitempty"`
	RoundEndsAt           *time.Time             `json:"roundEndsAt,omitempty"`
	RevealEndsAt          *time.Time             `json:"revealEndsAt,omitempty"`
	CreatedAt             time.Time              `json:"createdAt"`
	StartedAt             *time.Time             `json:"startedAt,omitempty"`
	CompletedAt           *time.Time             `json:"completedAt,omitempty"`
	Results               []MatchQuestionResult  `json:"results,omitempty"`
}

type MatchPlayerResponse struct {
	PlayerID                 string                 `json:"playerId" example:"7H9K2Q"`
	Nickname                 string                 `json:"nickname" example:"Alice"`
	Kind                     string                 `json:"kind" example:"human"`
	Ready                    bool                   `json:"ready" example:"true"`
	AnsweredCurrentQuestion  bool                   `json:"answeredCurrentQuestion,omitempty" example:"true"`
	SitoneIDs                []string               `json:"sitoneIds,omitempty" example:"stone_engineering_base,stone_explorer_base"`
	Score                    *int                   `json:"score,omitempty" example:"850"`
	MaxScore                 *int                   `json:"maxScore,omitempty" example:"2250"`
	AnswerScoreBonusPercent  int                    `json:"answerScoreBonusPercent,omitempty" example:"10"`
	OpenPowerBonusPercent    int                    `json:"openPowerBonusPercent,omitempty" example:"20"`
	MaterialDropBonusPercent int                    `json:"materialDropBonusPercent,omitempty" example:"15"`
	EliminateChancePercent   int                    `json:"eliminateChancePercent,omitempty" example:"25"`
	EliminateCount           int                    `json:"eliminateCount,omitempty" example:"1"`
	EliminatedChoices        []string               `json:"eliminatedChoices,omitempty" example:"B,C"`
	EliminatedBy             []string               `json:"eliminatedBy,omitempty" example:"靈光型小石"`
	BaseOpenPowerReward      *int                   `json:"baseOpenPowerReward,omitempty" example:"105"`
	OpenPowerReward          *int                   `json:"openPowerReward,omitempty" example:"126"`
	MaterialDrop             *MatchItemDropResponse `json:"materialDrop,omitempty"`
}

type MatchQuestionResponse struct {
	QuestionID string `json:"questionId" example:"quiz-001"`
	Prompt     string `json:"prompt" example:"哪個指令可以初始化新的 Git 儲存庫？"`
	ChoiceA    string `json:"choiceA" example:"git init"`
	ChoiceB    string `json:"choiceB" example:"git clone"`
	ChoiceC    string `json:"choiceC" example:"git status"`
	ChoiceD    string `json:"choiceD" example:"git add"`
}

type MatchQuestionResult struct {
	QuestionID    string                `json:"questionId" example:"quiz-001"`
	Prompt        string                `json:"prompt" example:"哪個指令可以初始化新的 Git 儲存庫？"`
	ChoiceA       string                `json:"choiceA" example:"git init"`
	ChoiceB       string                `json:"choiceB" example:"git clone"`
	ChoiceC       string                `json:"choiceC" example:"git status"`
	ChoiceD       string                `json:"choiceD" example:"git add"`
	CorrectChoice string                `json:"correctChoice" example:"A"`
	Explanation   string                `json:"explanation" example:"git init 會在目前目錄建立新的 Git 儲存庫。"`
	Answers       []MatchAnswerResponse `json:"answers"`
}

type MatchAnswerResponse struct {
	PlayerID      string     `json:"playerId" example:"7H9K2Q"`
	Nickname      string     `json:"nickname" example:"Alice"`
	Choice        string     `json:"choice,omitempty" example:"A"`
	Correct       bool       `json:"correct" example:"true"`
	BaseScore     int        `json:"baseScore" example:"150"`
	BonusScore    int        `json:"bonusScore" example:"15"`
	Score         int        `json:"score" example:"150"`
	ElapsedMillis int64      `json:"elapsedMillis" example:"3200"`
	AnsweredAt    *time.Time `json:"answeredAt,omitempty"`
}

type MatchItemDropResponse struct {
	Dropped    bool   `json:"dropped" example:"true"`
	ItemID     string `json:"itemId,omitempty"`
	ItemName   string `json:"itemName,omitempty"`
	SitoneID   string `json:"sitoneId,omitempty" example:"stone_explorer_base"`
	SitoneName string `json:"sitoneName,omitempty" example:"探索型小石"`
	Quantity   int    `json:"quantity,omitempty" example:"1"`
	DropRate   int    `json:"dropRate" example:"25"`
}
