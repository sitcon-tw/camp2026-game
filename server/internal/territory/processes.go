package territory

// Yansan process flows (design §8-§10): captive convert, unique redeem and
// missing-stone rescue. All of them charge the executor up front (cost by
// beneficiary, design §5), then resolve later via the central sweeper.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ErrBossUnderAttack is returned when yansan services are suspended.
var ErrBossUnderAttack = errors.New("yansan services suspended: boss under attack")

// ErrCaptiveNotConvertible is returned for cooldown / already-converted
// captives.
var ErrCaptiveNotConvertible = errors.New("captive is not convertible")

// ErrProcessAlreadyPending is returned when the target already has a
// pending process.
var ErrProcessAlreadyPending = errors.New("a pending process already exists for this target")

// ErrNotYourTeam is returned when the target does not belong to the
// executor's team.
var ErrNotYourTeam = errors.New("target does not belong to your team")

// ErrWrongState is returned when the target is not in the required state.
var ErrWrongState = errors.New("target is not in the required state")

// StartConvert begins converting a captive the executor's team holds. The
// executor pays; success chance decreases with the team's prior successful
// converts and can be boosted with extra invested open power (also paid).
func (s *Service) StartConvert(ctx context.Context, executor mongomodel.Player, captive mongomodel.CaptiveRecord, executorRank int, teamSize int, extraOpenPower int) (mongomodel.YansanProcess, error) {
	now := time.Now().UTC()
	if captive.CurrentTeamID != executor.TeamID {
		return mongomodel.YansanProcess{}, ErrNotYourTeam
	}
	if captive.Status == mongomodel.CaptiveRecordStatusCooldown && now.After(captive.CooldownUntil) {
		captive.Status = mongomodel.CaptiveRecordStatusConvertible
	}
	if captive.Status != mongomodel.CaptiveRecordStatusConvertible {
		return mongomodel.YansanProcess{}, ErrCaptiveNotConvertible
	}
	if err := s.requireYansanAvailable(ctx); err != nil {
		return mongomodel.YansanProcess{}, err
	}

	priorConverts, err := s.countProcesses(ctx, executor.TeamID, mongomodel.YansanProcessKindConvert, mongomodel.YansanProcessStatusSucceeded)
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}
	priorAttempts, err := s.countProcesses(ctx, executor.TeamID, mongomodel.YansanProcessKindConvert, "")
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}

	seedRef := NewSeedRef()
	roll := NewRoll(seedRef)
	cost, err := s.Costs.Roll(CostKindConvert, executorRank, teamSize, CostExtras{TeamConvertCount: priorAttempts}, roll)
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}
	if extraOpenPower < 0 {
		extraOpenPower = 0
	}
	successChance := ConvertSuccessChance(s.Engine, priorConverts, extraOpenPower)
	resolveAt := now.Add(RollProcessDuration(s.Engine.ConvertResolveMin, s.Engine.ConvertResolveMax, roll))

	process := mongomodel.YansanProcess{
		ID:                  NewID("yansan_process"),
		Kind:                mongomodel.YansanProcessKindConvert,
		SitoneID:            captive.SitoneID,
		InstanceID:          captive.InstanceID,
		ActorPlayerID:       executor.ID,
		BeneficiaryPlayerID: executor.ID,
		TeamID:              executor.TeamID,
		CostOpenPower:       cost + extraOpenPower,
		Status:              mongomodel.YansanProcessStatusPending,
		StartedAt:           now,
		ResolveAt:           resolveAt,
		Result: bson.M{
			"seed_ref":          seedRef,
			"success_chance":    successChance,
			"captive_record_id": captive.ID,
			"extra_open_power":  extraOpenPower,
		},
	}

	err = s.WithLock(ctx, "yansan:"+executor.TeamID, func(ctx context.Context) error {
		return s.InTransaction(ctx, func(ctx context.Context) error {
			// Ensure the captive has no pending process and mark the
			// instance convert_pending.
			pending, err := s.hasPendingProcess(ctx, captive.InstanceID)
			if err != nil {
				return err
			}
			if pending {
				return ErrProcessAlreadyPending
			}
			if err := s.SpendOpenPower(ctx, executor.ID, process.CostOpenPower, "yansan_convert", process.ID); err != nil {
				return err
			}
			result, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).UpdateOne(
				ctx,
				bson.M{"_id": captive.InstanceID, "status": mongomodel.SitoneInstanceStatusCaptured},
				bson.M{"$set": bson.M{
					"status":     mongomodel.SitoneInstanceStatusConvertPending,
					"updated_at": now,
				}},
			)
			if err != nil {
				return err
			}
			if result.ModifiedCount == 0 {
				return ErrCaptiveNotConvertible
			}
			if captive.Status == mongomodel.CaptiveRecordStatusConvertible {
				_, err = s.DB.Collection(mongomodel.CaptiveRecordsCollection).UpdateOne(
					ctx,
					bson.M{"_id": captive.ID},
					bson.M{"$set": bson.M{"status": mongomodel.CaptiveRecordStatusConvertible}},
				)
				if err != nil {
					return err
				}
			}
			_, err = s.DB.Collection(mongomodel.YansanProcessesCollection).InsertOne(ctx, process)
			return err
		})
	})
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}

	s.logEvent(ctx, mongomodel.EventLog{
		EventType:       mongomodel.EventTypeConvertStarted,
		ActorPlayerID:   executor.ID,
		TeamID:          executor.TeamID,
		RelatedSitoneID: captive.SitoneID,
		Payload:         bson.M{"process_id": process.ID, "cost": process.CostOpenPower},
	})
	return process, nil
}

// StartRedeem begins redeeming a unique stone of the executor's team held
// at yansan. Cost and success rate scale with the team's redeem count.
func (s *Service) StartRedeem(ctx context.Context, executor mongomodel.Player, instance mongomodel.SitoneInstance, beneficiaryRank int, teamSize int) (mongomodel.YansanProcess, error) {
	if instance.Status != mongomodel.SitoneInstanceStatusProtectedAtYansan {
		return mongomodel.YansanProcess{}, ErrWrongState
	}
	if instance.OriginTeamID != executor.TeamID {
		return mongomodel.YansanProcess{}, ErrNotYourTeam
	}
	if err := s.requireYansanAvailable(ctx); err != nil {
		return mongomodel.YansanProcess{}, err
	}

	teamRedeemCount, err := s.TeamRedeemCount(ctx, executor.TeamID)
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}

	now := time.Now().UTC()
	seedRef := NewSeedRef()
	roll := NewRoll(seedRef)
	cost, err := s.Costs.Roll(CostKindRedeem, beneficiaryRank, teamSize, CostExtras{TeamRedeemCount: teamRedeemCount}, roll)
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}
	successChance := RedeemSuccessChance(s.Engine, teamRedeemCount)
	resolveAt := now.Add(RollProcessDuration(s.Engine.RedeemResolveMin, s.Engine.RedeemResolveMax, roll))

	process := mongomodel.YansanProcess{
		ID:                  NewID("yansan_process"),
		Kind:                mongomodel.YansanProcessKindRedeem,
		SitoneID:            instance.SitoneID,
		InstanceID:          instance.ID,
		ActorPlayerID:       executor.ID,
		BeneficiaryPlayerID: instance.OriginPlayerID,
		TeamID:              executor.TeamID,
		CostOpenPower:       cost,
		Status:              mongomodel.YansanProcessStatusPending,
		StartedAt:           now,
		ResolveAt:           resolveAt,
		Result: bson.M{
			"seed_ref":       seedRef,
			"success_chance": successChance,
		},
	}

	err = s.WithLock(ctx, "yansan:"+executor.TeamID, func(ctx context.Context) error {
		return s.InTransaction(ctx, func(ctx context.Context) error {
			pending, err := s.hasPendingProcess(ctx, instance.ID)
			if err != nil {
				return err
			}
			if pending {
				return ErrProcessAlreadyPending
			}
			if err := s.SpendOpenPower(ctx, executor.ID, cost, "yansan_redeem", process.ID); err != nil {
				return err
			}
			_, err = s.DB.Collection(mongomodel.YansanProcessesCollection).InsertOne(ctx, process)
			return err
		})
	})
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}

	s.logEvent(ctx, mongomodel.EventLog{
		EventType:       mongomodel.EventTypeRedeemStarted,
		ActorPlayerID:   executor.ID,
		TeamID:          executor.TeamID,
		RelatedSitoneID: instance.SitoneID,
		Payload:         bson.M{"process_id": process.ID, "cost": cost},
	})
	return process, nil
}

// StartRescue begins a rescue of a missing instance of the executor's
// team. The executor pays; cost is keyed on the original holder's rank.
// Rescues are not a yansan service, so the boss state does not gate them.
func (s *Service) StartRescue(ctx context.Context, executor mongomodel.Player, instance mongomodel.SitoneInstance, beneficiaryRank int, teamSize int) (mongomodel.YansanProcess, error) {
	if instance.Status != mongomodel.SitoneInstanceStatusMissing {
		return mongomodel.YansanProcess{}, ErrWrongState
	}
	if instance.TeamID != executor.TeamID {
		return mongomodel.YansanProcess{}, ErrNotYourTeam
	}

	now := time.Now().UTC()
	seedRef := NewSeedRef()
	roll := NewRoll(seedRef)
	cost, err := s.Costs.Roll(CostKindRescue, beneficiaryRank, teamSize, CostExtras{}, roll)
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}
	resolveAt := now.Add(RollProcessDuration(s.Engine.RescueResolveMin, s.Engine.RescueResolveMax, roll))

	process := mongomodel.YansanProcess{
		ID:                  NewID("yansan_process"),
		Kind:                mongomodel.YansanProcessKindRescue,
		SitoneID:            instance.SitoneID,
		InstanceID:          instance.ID,
		ActorPlayerID:       executor.ID,
		BeneficiaryPlayerID: instance.OriginPlayerID,
		TeamID:              executor.TeamID,
		CostOpenPower:       cost,
		Status:              mongomodel.YansanProcessStatusPending,
		StartedAt:           now,
		ResolveAt:           resolveAt,
		Result:              bson.M{"seed_ref": seedRef},
	}

	err = s.WithLock(ctx, "rescue:"+executor.TeamID, func(ctx context.Context) error {
		return s.InTransaction(ctx, func(ctx context.Context) error {
			pending, err := s.hasPendingProcess(ctx, instance.ID)
			if err != nil {
				return err
			}
			if pending {
				return ErrProcessAlreadyPending
			}
			if err := s.SpendOpenPower(ctx, executor.ID, cost, "territory_rescue", process.ID); err != nil {
				return err
			}
			_, err = s.DB.Collection(mongomodel.YansanProcessesCollection).InsertOne(ctx, process)
			return err
		})
	})
	if err != nil {
		return mongomodel.YansanProcess{}, err
	}

	s.logEvent(ctx, mongomodel.EventLog{
		EventType:       mongomodel.EventTypeRescueStarted,
		ActorPlayerID:   executor.ID,
		TeamID:          executor.TeamID,
		RelatedSitoneID: instance.SitoneID,
		Payload:         bson.M{"process_id": process.ID, "cost": cost},
	})
	return process, nil
}

// ResolveDueProcess resolves one pending yansan process whose resolve_at
// has passed.
func (s *Service) ResolveDueProcess(ctx context.Context, process mongomodel.YansanProcess) error {
	switch process.Kind {
	case mongomodel.YansanProcessKindConvert:
		return s.resolveConvert(ctx, process)
	case mongomodel.YansanProcessKindRedeem:
		return s.resolveRedeem(ctx, process)
	case mongomodel.YansanProcessKindRescue:
		return s.resolveRescue(ctx, process)
	default:
		// Unknown kind: fail it so the sweeper does not spin on it.
		return s.finishProcess(ctx, process, mongomodel.YansanProcessStatusFailed, bson.M{"error": "unsupported kind"})
	}
}

func (s *Service) resolveConvert(ctx context.Context, process mongomodel.YansanProcess) error {
	now := time.Now().UTC()
	chance := resultFloat(process.Result, "success_chance")
	roll := NewRoll(resultString(process.Result, "seed_ref") + ":resolve")
	success := roll.Float() < chance

	captiveRecordID := resultString(process.Result, "captive_record_id")

	err := s.InTransaction(ctx, func(ctx context.Context) error {
		if success {
			// Captive joins the new team at full value: record converted,
			// instance dissolved into the beneficiary's free stock.
			_, err := s.DB.Collection(mongomodel.CaptiveRecordsCollection).UpdateOne(
				ctx,
				bson.M{"_id": captiveRecordID},
				bson.M{"$set": bson.M{
					"status":       mongomodel.CaptiveRecordStatusConverted,
					"converted_at": now,
				}},
			)
			if err != nil {
				return err
			}
			if _, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).DeleteOne(ctx, bson.M{"_id": process.InstanceID}); err != nil {
				return err
			}
			if err := s.IncrementQuantity(ctx, process.BeneficiaryPlayerID, process.SitoneID, 1); err != nil {
				return err
			}
		} else {
			// Failure: the captive stays captured/convertible.
			_, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).UpdateOne(
				ctx,
				bson.M{"_id": process.InstanceID, "status": mongomodel.SitoneInstanceStatusConvertPending},
				bson.M{"$set": bson.M{
					"status":     mongomodel.SitoneInstanceStatusCaptured,
					"updated_at": now,
				}},
			)
			if err != nil {
				return err
			}
		}
		status := mongomodel.YansanProcessStatusFailed
		if success {
			status = mongomodel.YansanProcessStatusSucceeded
		}
		return s.finishProcess(ctx, process, status, bson.M{"success": success})
	})
	if err != nil {
		return err
	}

	eventType := mongomodel.EventTypeConvertFailed
	if success {
		eventType = mongomodel.EventTypeConvertSucceeded
	}
	s.logEvent(ctx, mongomodel.EventLog{
		EventType:       eventType,
		ActorPlayerID:   process.ActorPlayerID,
		TeamID:          process.TeamID,
		RelatedSitoneID: process.SitoneID,
		Payload:         bson.M{"process_id": process.ID},
	})
	return nil
}

func (s *Service) resolveRedeem(ctx context.Context, process mongomodel.YansanProcess) error {
	chance := resultFloat(process.Result, "success_chance")
	roll := NewRoll(resultString(process.Result, "seed_ref") + ":resolve")
	success := roll.Float() < chance

	err := s.InTransaction(ctx, func(ctx context.Context) error {
		if success {
			// The unique stone is released back to its original holder.
			result, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).DeleteOne(ctx, bson.M{
				"_id":    process.InstanceID,
				"status": mongomodel.SitoneInstanceStatusProtectedAtYansan,
			})
			if err != nil {
				return err
			}
			if result.DeletedCount > 0 {
				if err := s.IncrementQuantity(ctx, process.BeneficiaryPlayerID, process.SitoneID, 1); err != nil {
					return err
				}
			}
		}
		// Failure: the stone stays protected at yansan (design §9).
		status := mongomodel.YansanProcessStatusFailed
		if success {
			status = mongomodel.YansanProcessStatusSucceeded
		}
		return s.finishProcess(ctx, process, status, bson.M{"success": success})
	})
	if err != nil {
		return err
	}

	eventType := mongomodel.EventTypeRedeemFailed
	if success {
		eventType = mongomodel.EventTypeRedeemSucceeded
	}
	s.logEvent(ctx, mongomodel.EventLog{
		EventType:       eventType,
		ActorPlayerID:   process.ActorPlayerID,
		TeamID:          process.TeamID,
		RelatedSitoneID: process.SitoneID,
		Payload:         bson.M{"process_id": process.ID},
	})
	return nil
}

func (s *Service) resolveRescue(ctx context.Context, process mongomodel.YansanProcess) error {
	now := time.Now().UTC()
	sitone, _ := s.Content.GetSitone(process.SitoneID)
	roll := NewRoll(resultString(process.Result, "seed_ref") + ":resolve")
	outcome := RollRescueOutcome(s.Engine, sitone, roll)
	// Dead only applies to repeatable stones; a doomed unique stone goes to
	// yansan protection instead (design §9).
	redirectedToYansan := false
	if outcome == RescueOutcomeDead && !sitone.Repeatable {
		if sitone.Unique {
			redirectedToYansan = true
		} else {
			outcome = RescueOutcomeFail
		}
	}

	fatigueGain := s.Engine.RescueFatigueGain

	err := s.InTransaction(ctx, func(ctx context.Context) error {
		switch {
		case outcome == RescueOutcomeSuccess:
			result, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).DeleteOne(ctx, bson.M{
				"_id":    process.InstanceID,
				"status": mongomodel.SitoneInstanceStatusMissing,
			})
			if err != nil {
				return err
			}
			if result.DeletedCount > 0 {
				if err := s.IncrementQuantity(ctx, process.BeneficiaryPlayerID, process.SitoneID, 1); err != nil {
					return err
				}
			}
		case outcome == RescueOutcomeDead && redirectedToYansan:
			if err := s.setInstanceStatus(ctx, process.InstanceID, mongomodel.SitoneInstanceStatusProtectedAtYansan, now, nil); err != nil {
				return err
			}
		case outcome == RescueOutcomeDead:
			if err := s.setInstanceStatus(ctx, process.InstanceID, mongomodel.SitoneInstanceStatusDead, now, nil); err != nil {
				return err
			}
		default:
			// fail / still_missing: stone stays missing, fatigue bumps.
			_, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).UpdateOne(
				ctx,
				bson.M{"_id": process.InstanceID},
				bson.M{
					"$set": bson.M{"updated_at": now},
					"$inc": bson.M{"fatigue": fatigueGain},
				},
			)
			if err != nil {
				return err
			}
		}
		status := mongomodel.YansanProcessStatusFailed
		if outcome == RescueOutcomeSuccess {
			status = mongomodel.YansanProcessStatusSucceeded
		}
		return s.finishProcess(ctx, process, status, bson.M{"outcome": outcome})
	})
	if err != nil {
		return err
	}

	s.logEvent(ctx, mongomodel.EventLog{
		EventType:       mongomodel.EventTypeRescueResolved,
		ActorPlayerID:   process.ActorPlayerID,
		TeamID:          process.TeamID,
		RelatedSitoneID: process.SitoneID,
		Payload:         bson.M{"process_id": process.ID, "outcome": outcome},
	})
	if outcome == RescueOutcomeDead && !redirectedToYansan {
		s.logEvent(ctx, mongomodel.EventLog{
			EventType:       mongomodel.EventTypeSitoneDied,
			TeamID:          process.TeamID,
			RelatedSitoneID: process.SitoneID,
			Payload:         bson.M{"process_id": process.ID},
		})
	}
	return nil
}

func (s *Service) finishProcess(ctx context.Context, process mongomodel.YansanProcess, status string, result bson.M) error {
	merged := bson.M{}
	for key, value := range process.Result {
		merged[key] = value
	}
	for key, value := range result {
		merged[key] = value
	}
	update, err := s.DB.Collection(mongomodel.YansanProcessesCollection).UpdateOne(
		ctx,
		bson.M{"_id": process.ID, "status": mongomodel.YansanProcessStatusPending},
		bson.M{"$set": bson.M{
			"status":      status,
			"resolved_at": time.Now().UTC(),
			"result":      merged,
		}},
	)
	if err != nil {
		return err
	}
	if update.ModifiedCount == 0 {
		return fmt.Errorf("process %s no longer pending", process.ID)
	}
	return nil
}

func (s *Service) hasPendingProcess(ctx context.Context, instanceID string) (bool, error) {
	if instanceID == "" {
		return false, nil
	}
	err := s.DB.Collection(mongomodel.YansanProcessesCollection).FindOne(ctx, bson.M{
		"instance_id": instanceID,
		"status":      mongomodel.YansanProcessStatusPending,
	}).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) countProcesses(ctx context.Context, teamID string, kind string, status string) (int, error) {
	filter := bson.M{
		"team_id": teamID,
		"kind":    kind,
	}
	if status != "" {
		filter["status"] = status
	} else {
		filter["status"] = bson.M{"$ne": mongomodel.YansanProcessStatusAbortedBoss}
	}
	count, err := s.DB.Collection(mongomodel.YansanProcessesCollection).CountDocuments(ctx, filter)
	return int(count), err
}

// TeamRedeemCount counts the team's redeem attempts (boss-aborted attempts
// excluded), used for cost and success-rate scaling (design §9: 贖回次數整
// 隊計算).
func (s *Service) TeamRedeemCount(ctx context.Context, teamID string) (int, error) {
	return s.countProcesses(ctx, teamID, mongomodel.YansanProcessKindRedeem, "")
}

// TeamConvertSuccessCount counts the team's successful converts.
func (s *Service) TeamConvertSuccessCount(ctx context.Context, teamID string) (int, error) {
	return s.countProcesses(ctx, teamID, mongomodel.YansanProcessKindConvert, mongomodel.YansanProcessStatusSucceeded)
}

func resultFloat(result bson.M, key string) float64 {
	if result == nil {
		return 0
	}
	switch value := result[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func resultString(result bson.M, key string) string {
	if result == nil {
		return ""
	}
	if value, ok := result[key].(string); ok {
		return value
	}
	return ""
}
