package territory

// Yansan boss state machine (design §12): staff-only open, fixed
// requirement of all 9 player teams registering to launch the attack.
// While under attack, every pending yansan service fails without refund.

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ErrBossNotOpen is returned when a boss attack is registered while the
// boss is not open.
var ErrBossNotOpen = errors.New("boss is not open for attack")

// ErrNotTerritoryParticipant is returned when a non-playing team attempts
// to join territory-only flows.
var ErrNotTerritoryParticipant = errors.New("team does not participate in territory")

// BossState reads the current boss state, defaulting to closed.
func (s *Service) BossState(ctx context.Context) (mongomodel.BossState, error) {
	var state mongomodel.BossState
	err := s.DB.Collection(mongomodel.BossStatesCollection).FindOne(ctx, bson.M{"_id": mongomodel.BossStateID}).Decode(&state)
	if err == mongo.ErrNoDocuments {
		return mongomodel.BossState{
			ID:     mongomodel.BossStateID,
			Status: mongomodel.BossStatusClosed,
		}, nil
	}
	if err != nil {
		return mongomodel.BossState{}, err
	}
	if state.Status == "" {
		state.Status = mongomodel.BossStatusClosed
	}
	return state, nil
}

func (s *Service) requireYansanAvailable(ctx context.Context) error {
	state, err := s.BossState(ctx)
	if err != nil {
		return err
	}
	if state.Status == mongomodel.BossStatusUnderAttack {
		return ErrBossUnderAttack
	}
	return nil
}

// OpenBoss transitions the boss to open (staff/admin action).
func (s *Service) OpenBoss(ctx context.Context, openedBy string) (mongomodel.BossState, error) {
	now := time.Now().UTC()
	_, err := s.DB.Collection(mongomodel.BossStatesCollection).UpdateOne(
		ctx,
		bson.M{"_id": mongomodel.BossStateID},
		bson.M{"$set": bson.M{
			"status":                 mongomodel.BossStatusOpen,
			"opened_by":              openedBy,
			"opened_at":              now,
			"participating_team_ids": []string{},
			"updated_at":             now,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return mongomodel.BossState{}, err
	}
	s.logEvent(ctx, mongomodel.EventLog{
		EventType: mongomodel.EventTypeBossOpened,
		Payload:   bson.M{"opened_by": openedBy},
	})
	return s.BossState(ctx)
}

// CloseBoss ends the boss fight: services resume.
func (s *Service) CloseBoss(ctx context.Context, closedBy string) (mongomodel.BossState, error) {
	now := time.Now().UTC()
	_, err := s.DB.Collection(mongomodel.BossStatesCollection).UpdateOne(
		ctx,
		bson.M{"_id": mongomodel.BossStateID},
		bson.M{"$set": bson.M{
			"status":     mongomodel.BossStatusFinished,
			"closed_at":  now,
			"updated_at": now,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return mongomodel.BossState{}, err
	}
	s.logEvent(ctx, mongomodel.EventLog{
		EventType: mongomodel.EventTypeBossClosed,
		Payload:   bson.M{"closed_by": closedBy},
	})
	return s.BossState(ctx)
}

// RegisterBossAttack records a team's participation. When every player
// team (all 9) has registered, the boss transitions to under_attack and
// every pending yansan process immediately fails (aborted_boss): stones
// revert to their prior state and open power is NOT refunded (design §12).
func (s *Service) RegisterBossAttack(ctx context.Context, player mongomodel.Player) (mongomodel.BossState, bool, error) {
	if !IsTerritoryParticipantTeam(player.TeamID) {
		return mongomodel.BossState{}, false, ErrNotTerritoryParticipant
	}

	var launched bool
	var state mongomodel.BossState

	err := s.WithLock(ctx, "boss", func(ctx context.Context) error {
		current, err := s.BossState(ctx)
		if err != nil {
			return err
		}
		if current.Status != mongomodel.BossStatusOpen {
			if current.Status == mongomodel.BossStatusUnderAttack {
				state = current
				return nil
			}
			return ErrBossNotOpen
		}

		now := time.Now().UTC()
		_, err = s.DB.Collection(mongomodel.BossStatesCollection).UpdateOne(
			ctx,
			bson.M{"_id": mongomodel.BossStateID},
			bson.M{
				"$addToSet": bson.M{"participating_team_ids": player.TeamID},
				"$set":      bson.M{"updated_at": now},
			},
		)
		if err != nil {
			return err
		}
		s.logEvent(ctx, mongomodel.EventLog{
			EventType:     mongomodel.EventTypeBossAttacked,
			ActorPlayerID: player.ID,
			TeamID:        player.TeamID,
			Payload:       bson.M{"phase": "registered"},
		})

		state, err = s.BossState(ctx)
		if err != nil {
			return err
		}
		required, err := s.playerTeamCount(ctx)
		if err != nil {
			return err
		}
		if len(state.ParticipatingTeamIDs) < required {
			return nil
		}

		// All teams in: launch the boss fight.
		_, err = s.DB.Collection(mongomodel.BossStatesCollection).UpdateOne(
			ctx,
			bson.M{"_id": mongomodel.BossStateID, "status": mongomodel.BossStatusOpen},
			bson.M{"$set": bson.M{
				"status":            mongomodel.BossStatusUnderAttack,
				"attack_started_at": now,
				"updated_at":        now,
			}},
		)
		if err != nil {
			return err
		}
		launched = true
		state.Status = mongomodel.BossStatusUnderAttack
		state.AttackStartedAt = now

		if err := s.abortPendingYansanProcesses(ctx, now); err != nil {
			return err
		}
		s.logEvent(ctx, mongomodel.EventLog{
			EventType:     mongomodel.EventTypeBossAttacked,
			ActorPlayerID: player.ID,
			TeamID:        player.TeamID,
			Payload: bson.M{
				"phase":               "launched",
				"participating_teams": state.ParticipatingTeamIDs,
			},
		})
		return nil
	})
	if err != nil {
		return mongomodel.BossState{}, false, err
	}
	return state, launched, nil
}

// abortPendingYansanProcesses fails every in-flight yansan service
// (convert / redeem / calibrate — rescues are unaffected). Stones revert:
// convert_pending instances go back to captured (captive record stays
// convertible), protected unique stones simply stay at yansan. Open power
// is NOT refunded.
func (s *Service) abortPendingYansanProcesses(ctx context.Context, now time.Time) error {
	cursor, err := s.DB.Collection(mongomodel.YansanProcessesCollection).Find(ctx, bson.M{
		"status": mongomodel.YansanProcessStatusPending,
		"kind": bson.M{"$in": bson.A{
			mongomodel.YansanProcessKindConvert,
			mongomodel.YansanProcessKindRedeem,
			mongomodel.YansanProcessKindCalibrate,
		}},
	})
	if err != nil {
		return err
	}
	var processes []mongomodel.YansanProcess
	err = cursor.All(ctx, &processes)
	if err != nil {
		return err
	}

	for _, process := range processes {
		_, err := s.DB.Collection(mongomodel.YansanProcessesCollection).UpdateOne(
			ctx,
			bson.M{"_id": process.ID, "status": mongomodel.YansanProcessStatusPending},
			bson.M{"$set": bson.M{
				"status":      mongomodel.YansanProcessStatusAbortedBoss,
				"resolved_at": now,
			}},
		)
		if err != nil {
			return err
		}
		// Revert the stone to its prior state.
		if process.Kind == mongomodel.YansanProcessKindConvert && process.InstanceID != "" {
			_, err = s.DB.Collection(mongomodel.SitoneInstancesCollection).UpdateOne(
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
		s.logEvent(ctx, mongomodel.EventLog{
			EventType:       mongomodel.EventTypeBossInterruptedYansan,
			ActorPlayerID:   process.ActorPlayerID,
			TeamID:          process.TeamID,
			RelatedSitoneID: process.SitoneID,
			Payload: bson.M{
				"process_id": process.ID,
				"kind":       process.Kind,
				"refunded":   false,
			},
		})
	}
	return nil
}

// playerTeamCount counts non-system teams (the fixed boss threshold is
// "all player teams participate", design §12 decision 11).
func (s *Service) playerTeamCount(ctx context.Context) (int, error) {
	count, err := s.DB.Collection(mongomodel.TeamsCollection).CountDocuments(ctx, bson.M{
		"_id":       bson.M{"$nin": bson.A{TeamYansanID, TeamStaffID}},
		"is_system": bson.M{"$ne": true},
	})
	return int(count), err
}
