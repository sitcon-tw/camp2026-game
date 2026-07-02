package model

import "time"

const MatchesCollection = "matches"

const (
	MatchStatusWaiting   = "waiting"
	MatchStatusActive    = "active"
	MatchStatusCompleted = "completed"
)

const (
	MatchModePVP      = "pvp"
	MatchModeComputer = "computer"
)

const (
	MatchPhaseAnswering = "answering"
	MatchPhaseRevealing = "revealing"
)

const (
	MatchPlayerKindHuman    = "human"
	MatchPlayerKindComputer = "computer"
)

type Match struct {
	ID                   string                  `bson:"_id"`
	Revision             int64                   `bson:"revision,omitempty"`
	Code                 string                  `bson:"code"`
	Mode                 string                  `bson:"mode,omitempty"`
	Status               string                  `bson:"status"`
	Phase                string                  `bson:"phase,omitempty"`
	HostPlayerID         string                  `bson:"host_player_id"`
	OpenHostLock         string                  `bson:"open_host_lock,omitempty"`
	Players              []MatchPlayer           `bson:"players"`
	QuestionIDs          []string                `bson:"question_ids,omitempty"`
	CurrentQuestionIndex int                     `bson:"current_question_index,omitempty"`
	EliminatedChoices    []MatchEliminatedChoice `bson:"eliminated_choices,omitempty"`
	RoundStartedAt       time.Time               `bson:"round_started_at,omitempty"`
	RoundEndsAt          time.Time               `bson:"round_ends_at,omitempty"`
	RevealEndsAt         time.Time               `bson:"reveal_ends_at,omitempty"`
	CreatedAt            time.Time               `bson:"created_at"`
	StartedAt            time.Time               `bson:"started_at,omitempty"`
	CompletedAt          time.Time               `bson:"completed_at,omitempty"`
}

type MatchPlayer struct {
	PlayerID      string              `bson:"player_id"`
	Nickname      string              `bson:"nickname"`
	Kind          string              `bson:"kind,omitempty"`
	Ready         bool                `bson:"ready"`
	Score         int                 `bson:"score"`
	SitoneIDs     []string            `bson:"sitone_ids,omitempty"`
	BattleEffects *MatchBattleEffects `bson:"battle_effects,omitempty"`
}

type MatchBattleEffects struct {
	MaterialDropBonusPercent int      `bson:"material_drop_bonus_percent,omitempty"`
	AnswerScoreBonusPercent  int      `bson:"answer_score_bonus_percent,omitempty"`
	OpenPowerBonusPercent    int      `bson:"open_power_bonus_percent,omitempty"`
	EliminateChancePercent   int      `bson:"eliminate_chance_percent,omitempty"`
	EliminateCount           int      `bson:"eliminate_count,omitempty"`
	EliminateSourceNames     []string `bson:"eliminate_source_names,omitempty"`
}

type MatchEliminatedChoice struct {
	QuestionID        string   `bson:"question_id"`
	PlayerID          string   `bson:"player_id"`
	Choices           []string `bson:"choices,omitempty"`
	SourceSitoneNames []string `bson:"source_sitone_names,omitempty"`
}
