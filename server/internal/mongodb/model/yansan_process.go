package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const YansanProcessesCollection = "yansan_processes"

const (
	YansanProcessKindConvert   = "convert"
	YansanProcessKindRedeem    = "redeem"
	YansanProcessKindCalibrate = "calibrate"
	// YansanProcessKindRescue is a rescue of a missing sitone. It shares
	// the process/sweeper machinery but is NOT a yansan service: a boss
	// attack does not abort pending rescues.
	YansanProcessKindRescue = "rescue"
)

const (
	YansanProcessStatusPending     = "pending"
	YansanProcessStatusSucceeded   = "succeeded"
	YansanProcessStatusFailed      = "failed"
	YansanProcessStatusAbortedBoss = "aborted_boss"
)

type YansanProcess struct {
	ID                  string    `bson:"_id"`
	Kind                string    `bson:"kind"`
	SitoneID            string    `bson:"sitone_id"`
	InstanceID          string    `bson:"instance_id,omitempty"`
	ActorPlayerID       string    `bson:"actor_player_id"`
	BeneficiaryPlayerID string    `bson:"beneficiary_player_id,omitempty"`
	TeamID              string    `bson:"team_id"`
	CostOpenPower       int       `bson:"cost_open_power"`
	Status              string    `bson:"status"`
	StartedAt           time.Time `bson:"started_at"`
	ResolveAt           time.Time `bson:"resolve_at,omitempty"`
	ResolvedAt          time.Time `bson:"resolved_at,omitempty"`
	Result              bson.M    `bson:"result,omitempty"`
}
