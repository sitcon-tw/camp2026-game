package territory

// Attack mission lifecycle (design §7): voting → deployed → resolved, with
// lazy expiry of overdue votes and central-sweeper resolution.

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/eventlog"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ExpireMissionIfDue lazily flips an overdue voting mission to expired.
// Returns the (possibly updated) mission.
func (s *Service) ExpireMissionIfDue(ctx context.Context, mission mongomodel.AttackMission) (mongomodel.AttackMission, error) {
	if mission.Status != mongomodel.AttackMissionStatusVoting || time.Now().Before(mission.VoteDeadline) {
		return mission, nil
	}
	now := time.Now().UTC()
	result, err := s.DB.Collection(mongomodel.AttackMissionsCollection).UpdateOne(
		ctx,
		bson.M{"_id": mission.ID, "status": mongomodel.AttackMissionStatusVoting},
		bson.M{"$set": bson.M{
			"status":     mongomodel.AttackMissionStatusExpired,
			"updated_at": now,
		}},
	)
	if err != nil {
		return mission, err
	}
	if result.ModifiedCount > 0 {
		mission.Status = mongomodel.AttackMissionStatusExpired
		mission.UpdatedAt = now
		_, _ = eventlog.Insert(ctx, s.DB, mongomodel.EventLog{
			EventType:        mongomodel.EventTypeAttackVoteExpired,
			TeamID:           mission.AttackerTeamID,
			RelatedMissionID: mission.ID,
			Payload:          bson.M{"vote_deadline": mission.VoteDeadline},
		})
	}
	return mission, nil
}

// CommitMission executes the "threshold reached" transition: the executor
// (initiator) pays the rolled open power cost, the selected sitones become
// deployed instances (quantity locked in the same transaction, design §2),
// and the mission gets its resolve_at and random seed.
func (s *Service) CommitMission(ctx context.Context, mission mongomodel.AttackMission, beneficiaryRank int, teamSize int) (mongomodel.AttackMission, error) {
	seedRef := NewSeedRef()
	roll := NewRoll(seedRef)
	cost, err := s.Costs.Roll(CostKindAttack, beneficiaryRank, teamSize, CostExtras{}, roll)
	if err != nil {
		return mission, err
	}
	duration := RollMissionDuration(s.Engine, roll)

	now := time.Now().UTC()
	resolveAt := now.Add(duration)

	err = s.InTransaction(ctx, func(ctx context.Context) error {
		// Deduct executor open power.
		if err := s.SpendOpenPower(ctx, mission.InitiatorPlayerID, cost, "territory_attack", mission.ID); err != nil {
			return err
		}

		// Lock quantities and create deployed instances (hybrid model §2:
		// instance creation and quantity decrement in one transaction).
		counts := countByID(mission.SelectedSitoneIDs)
		for sitoneID, n := range counts {
			if err := s.DecrementQuantity(ctx, mission.InitiatorPlayerID, sitoneID, n); err != nil {
				return err
			}
		}
		instances := make([]any, 0, len(mission.SelectedSitoneIDs))
		for _, sitoneID := range mission.SelectedSitoneIDs {
			instances = append(instances, mongomodel.SitoneInstance{
				ID:             NewID("sitone_instance"),
				PlayerID:       mission.InitiatorPlayerID,
				TeamID:         mission.AttackerTeamID,
				OriginPlayerID: mission.InitiatorPlayerID,
				OriginTeamID:   mission.AttackerTeamID,
				SitoneID:       sitoneID,
				Status:         mongomodel.SitoneInstanceStatusDeployed,
				MissionID:      mission.ID,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
		if len(instances) > 0 {
			if _, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).InsertMany(ctx, instances); err != nil {
				return err
			}
		}

		result, err := s.DB.Collection(mongomodel.AttackMissionsCollection).UpdateOne(
			ctx,
			bson.M{"_id": mission.ID, "status": mongomodel.AttackMissionStatusVoting},
			bson.M{"$set": bson.M{
				"status":          mongomodel.AttackMissionStatusDeployed,
				"cost_open_power": cost,
				"started_at":      now,
				"resolve_at":      resolveAt,
				"random_seed_ref": seedRef,
				"updated_at":      now,
			}},
		)
		if err != nil {
			return err
		}
		if result.ModifiedCount == 0 {
			return fmt.Errorf("mission %s no longer in voting", mission.ID)
		}
		return nil
	})
	if err != nil {
		return mission, err
	}

	mission.Status = mongomodel.AttackMissionStatusDeployed
	mission.CostOpenPower = cost
	mission.StartedAt = now
	mission.ResolveAt = resolveAt
	mission.RandomSeedRef = seedRef
	mission.UpdatedAt = now

	s.logEvent(ctx, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeOpenPowerSpent,
		ActorPlayerID:    mission.InitiatorPlayerID,
		TeamID:           mission.AttackerTeamID,
		RelatedMissionID: mission.ID,
		Payload:          bson.M{"amount": cost, "reason": "territory_attack"},
	})
	s.logEvent(ctx, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeSitonesDeployed,
		ActorPlayerID:    mission.InitiatorPlayerID,
		TeamID:           mission.AttackerTeamID,
		RelatedMissionID: mission.ID,
		Payload: bson.M{
			"sitone_ids": mission.SelectedSitoneIDs,
			"resolve_at": resolveAt,
		},
	})
	return mission, nil
}

// ResolveDueMission resolves one deployed mission whose resolve_at has
// passed: compute A/D, run steal + risk passes, apply every state change
// and write the public-safe result summary plus private event payloads.
func (s *Service) ResolveDueMission(ctx context.Context, mission mongomodel.AttackMission) error {
	now := time.Now().UTC()

	instances, err := s.missionInstances(ctx, mission.ID)
	if err != nil {
		return fmt.Errorf("mission %s instances: %w", mission.ID, err)
	}

	attackerMembers, err := s.TeamMembers(ctx, mission.AttackerTeamID)
	if err != nil {
		return err
	}
	defenderMembers, err := s.TeamMembers(ctx, mission.DefenderTeamID)
	if err != nil {
		return err
	}
	defenderIDs := playerIDs(defenderMembers)

	input := MissionInput{
		AttackerTeamSize: len(attackerMembers),
		DefenderTeamSize: len(defenderMembers),
		CostPaid:         mission.CostOpenPower,
	}
	for _, instance := range instances {
		sitone, ok := s.Content.GetSitone(instance.SitoneID)
		if !ok {
			continue
		}
		input.Attackers = append(input.Attackers, DeployedStone{
			InstanceID:     instance.ID,
			SitoneID:       instance.SitoneID,
			OriginPlayerID: instance.OriginPlayerID,
			Fatigue:        instance.Fatigue,
			Sitone:         sitone,
		})
	}

	// Defense is read at RESOLVE time (design §0.1 decision 7).
	defenseDoc, err := s.teamDefense(ctx, mission.DefenderTeamID)
	if err != nil {
		return err
	}
	defenderCaptives, err := s.captiveSitoneSet(ctx, mission.DefenderTeamID)
	if err != nil {
		return err
	}
	defenderQuantities, err := s.TeamQuantities(ctx, defenderIDs)
	if err != nil {
		return err
	}
	for _, sitoneID := range defenseDoc.SitoneIDs {
		sitone, ok := s.Content.GetSitone(sitoneID)
		if !ok {
			continue
		}
		// A defense stone the team only holds as a captive counts at 50%.
		captiveOnly := defenderCaptives[sitoneID] && defenderQuantities[sitoneID] == 0
		input.Defense = append(input.Defense, DefenseStone{Sitone: sitone, Captive: captiveOnly})
	}

	input.StealPool, err = s.buildStealPool(ctx, mission, defenderMembers, defenderQuantities)
	if err != nil {
		return err
	}

	roll := NewRoll(mission.RandomSeedRef + ":resolve")
	resolution := ResolveMission(s.Engine, input, roll)

	summary := buildResultSummary(s, resolution)

	err = s.InTransaction(ctx, func(ctx context.Context) error {
		if err := s.applyStealResults(ctx, mission, resolution, now); err != nil {
			return err
		}
		if err := s.applyStoneResults(ctx, mission, resolution, now); err != nil {
			return err
		}
		if resolution.OpRefund > 0 {
			if err := s.GrantOpenPower(ctx, mission.InitiatorPlayerID, resolution.OpRefund, "territory_attack_refund", mission.ID); err != nil {
				return err
			}
		}
		result, err := s.DB.Collection(mongomodel.AttackMissionsCollection).UpdateOne(
			ctx,
			bson.M{"_id": mission.ID, "status": mongomodel.AttackMissionStatusDeployed},
			bson.M{"$set": bson.M{
				"status":         mongomodel.AttackMissionStatusResolved,
				"resolved_at":    now,
				"result_summary": summary,
				"updated_at":     now,
			}},
		)
		if err != nil {
			return err
		}
		if result.ModifiedCount == 0 {
			return fmt.Errorf("mission %s no longer deployed", mission.ID)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Private full detail lives only in the event payload (staff-visible).
	s.logEvent(ctx, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeMissionResolved,
		ActorPlayerID:    mission.InitiatorPlayerID,
		TeamID:           mission.AttackerTeamID,
		RelatedMissionID: mission.ID,
		Payload:          resolutionDetailPayload(mission, resolution),
	})
	if resolution.StealHappened {
		s.logEvent(ctx, mongomodel.EventLog{
			EventType:        mongomodel.EventTypeStealSuccess,
			TeamID:           mission.AttackerTeamID,
			RelatedMissionID: mission.ID,
			Payload:          bson.M{"stolen": stolenPayload(resolution.Stolen), "defender_team_id": mission.DefenderTeamID},
		})
	}
	for _, stone := range resolution.Stones {
		eventType := ""
		targetTeam := mission.AttackerTeamID
		switch stone.Outcome {
		case StoneOutcomeMissing:
			eventType = mongomodel.EventTypeSitoneMissing
		case StoneOutcomeDead:
			eventType = mongomodel.EventTypeSitoneDied
		case StoneOutcomeProtected:
			eventType = mongomodel.EventTypeSitoneSentYansan
		case StoneOutcomeCaptured:
			eventType = mongomodel.EventTypeSitoneCaptured
			targetTeam = mission.DefenderTeamID
		default:
			continue
		}
		s.logEvent(ctx, mongomodel.EventLog{
			EventType:        eventType,
			TeamID:           targetTeam,
			TargetPlayerID:   stone.Stone.OriginPlayerID,
			RelatedSitoneID:  stone.Stone.SitoneID,
			RelatedMissionID: mission.ID,
			Payload:          bson.M{"instance_id": stone.Stone.InstanceID},
		})
	}
	return nil
}

func (s *Service) missionInstances(ctx context.Context, missionID string) ([]mongomodel.SitoneInstance, error) {
	cursor, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).Find(ctx, bson.M{
		"mission_id": missionID,
		"status":     mongomodel.SitoneInstanceStatusDeployed,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var instances []mongomodel.SitoneInstance
	if err := cursor.All(ctx, &instances); err != nil {
		return nil, err
	}
	return instances, nil
}

func (s *Service) teamDefense(ctx context.Context, teamID string) (mongomodel.TeamDefense, error) {
	var defense mongomodel.TeamDefense
	err := s.DB.Collection(mongomodel.TeamDefensesCollection).FindOne(ctx, bson.M{"_id": teamID}).Decode(&defense)
	if err == mongo.ErrNoDocuments {
		return mongomodel.TeamDefense{ID: teamID, SitoneIDs: []string{}}, nil
	}
	if err != nil {
		return mongomodel.TeamDefense{}, err
	}
	return defense, nil
}

// captiveSitoneSet returns the sitone ids the team currently holds as
// captives (cooldown or convertible).
func (s *Service) captiveSitoneSet(ctx context.Context, teamID string) (map[string]bool, error) {
	cursor, err := s.DB.Collection(mongomodel.CaptiveRecordsCollection).Find(ctx, bson.M{
		"current_team_id": teamID,
		"status": bson.M{"$in": bson.A{
			mongomodel.CaptiveRecordStatusCooldown,
			mongomodel.CaptiveRecordStatusConvertible,
		}},
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var records []mongomodel.CaptiveRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(records))
	for _, record := range records {
		set[record.SitoneID] = true
	}
	return set, nil
}

// buildStealPool lists what the attacker can steal from the defender:
// recapture candidates first (the attacker's own captives held by the
// defender, design §8), then the defender members' repeatable stones.
// Unique stones are never stealable (design §9).
func (s *Service) buildStealPool(ctx context.Context, mission mongomodel.AttackMission, defenderMembers []mongomodel.Player, defenderQuantities map[string]int) ([]StealCandidate, error) {
	pool := make([]StealCandidate, 0)

	// Recapture: captives held by the defender that originally belong to
	// the attacker.
	cursor, err := s.DB.Collection(mongomodel.CaptiveRecordsCollection).Find(ctx, bson.M{
		"current_team_id":  mission.DefenderTeamID,
		"original_team_id": mission.AttackerTeamID,
		"status": bson.M{"$in": bson.A{
			mongomodel.CaptiveRecordStatusCooldown,
			mongomodel.CaptiveRecordStatusConvertible,
		}},
	})
	if err != nil {
		return nil, err
	}
	var recaptures []mongomodel.CaptiveRecord
	err = cursor.All(ctx, &recaptures)
	if err != nil {
		return nil, err
	}
	for _, record := range recaptures {
		pool = append(pool, StealCandidate{
			SitoneID:          record.SitoneID,
			CaptiveInstanceID: record.InstanceID,
			CaptiveRecordID:   record.ID,
			Recapture:         true,
		})
	}

	// Quantity-based repeatable stones of defender members. One candidate
	// per (owner, sitone) unit up to quantity, flattened.
	for _, member := range defenderMembers {
		quantities, err := s.PlayerQuantities(ctx, member.ID)
		if err != nil {
			return nil, err
		}
		for sitoneID, quantity := range quantities {
			sitone, ok := s.Content.GetSitone(sitoneID)
			if !ok || !sitone.Repeatable || sitone.Unique {
				continue
			}
			for i := 0; i < quantity; i++ {
				pool = append(pool, StealCandidate{
					SitoneID:      sitoneID,
					OwnerPlayerID: member.ID,
				})
			}
		}
	}
	_ = defenderQuantities
	return pool, nil
}

func (s *Service) applyStealResults(ctx context.Context, mission mongomodel.AttackMission, resolution MissionResolution, now time.Time) error {
	for _, stolen := range resolution.Stolen {
		if stolen.Recapture {
			// 原隊搶回: the captive returns home at full value.
			if err := s.recaptureCaptive(ctx, mission, stolen, now); err != nil {
				return err
			}
			continue
		}

		// Regular steal: decrement the defender member's quantity and
		// create a captured instance under the attacker.
		if err := s.DecrementQuantity(ctx, stolen.OwnerPlayerID, stolen.SitoneID, 1); err != nil {
			if err == ErrInsufficientSitones {
				continue // stock changed since pool snapshot; skip
			}
			return err
		}
		instanceID := NewID("sitone_instance")
		_, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).InsertOne(ctx, mongomodel.SitoneInstance{
			ID:             instanceID,
			PlayerID:       mission.InitiatorPlayerID,
			TeamID:         mission.AttackerTeamID,
			OriginPlayerID: stolen.OwnerPlayerID,
			OriginTeamID:   mission.DefenderTeamID,
			SitoneID:       stolen.SitoneID,
			Status:         mongomodel.SitoneInstanceStatusCaptured,
			MissionID:      mission.ID,
			CooldownUntil:  now.Add(CaptiveCooldownDuration),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
		if err != nil {
			return err
		}
		_, err = s.DB.Collection(mongomodel.CaptiveRecordsCollection).InsertOne(ctx, mongomodel.CaptiveRecord{
			ID:             NewID("captive"),
			SitoneID:       stolen.SitoneID,
			OriginalTeamID: mission.DefenderTeamID,
			CurrentTeamID:  mission.AttackerTeamID,
			InstanceID:     instanceID,
			CapturedAt:     now,
			CooldownUntil:  now.Add(CaptiveCooldownDuration),
			Status:         mongomodel.CaptiveRecordStatusCooldown,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) recaptureCaptive(ctx context.Context, mission mongomodel.AttackMission, stolen StealCandidate, now time.Time) error {
	result, err := s.DB.Collection(mongomodel.CaptiveRecordsCollection).UpdateOne(
		ctx,
		bson.M{
			"_id": stolen.CaptiveRecordID,
			"status": bson.M{"$in": bson.A{
				mongomodel.CaptiveRecordStatusCooldown,
				mongomodel.CaptiveRecordStatusConvertible,
			}},
		},
		bson.M{"$set": bson.M{"status": mongomodel.CaptiveRecordStatusRecaptured}},
	)
	if err != nil {
		return err
	}
	if result.ModifiedCount == 0 {
		return nil // already converted or recaptured; skip
	}

	// Find the captive instance to learn the original holder, then free it
	// back into the attacker's stock.
	var instance mongomodel.SitoneInstance
	err = s.DB.Collection(mongomodel.SitoneInstancesCollection).FindOne(ctx, bson.M{"_id": stolen.CaptiveInstanceID}).Decode(&instance)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	holder := instance.OriginPlayerID
	if holder == "" {
		holder = mission.InitiatorPlayerID
	}
	if instance.ID != "" {
		if _, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).DeleteOne(ctx, bson.M{"_id": instance.ID}); err != nil {
			return err
		}
	}
	if err := s.IncrementQuantity(ctx, holder, stolen.SitoneID, 1); err != nil {
		return err
	}
	s.logEvent(ctx, mongomodel.EventLog{
		EventType:        mongomodel.EventTypeCaptiveRecaptured,
		TeamID:           mission.AttackerTeamID,
		TargetPlayerID:   holder,
		RelatedSitoneID:  stolen.SitoneID,
		RelatedMissionID: mission.ID,
	})
	return nil
}

func (s *Service) applyStoneResults(ctx context.Context, mission mongomodel.AttackMission, resolution MissionResolution, now time.Time) error {
	instanceCollection := s.DB.Collection(mongomodel.SitoneInstancesCollection)
	for _, stone := range resolution.Stones {
		switch stone.Outcome {
		case StoneOutcomeReturned:
			// Survivor: instance released, quantity restored.
			if _, err := instanceCollection.DeleteOne(ctx, bson.M{"_id": stone.Stone.InstanceID}); err != nil {
				return err
			}
			if err := s.IncrementQuantity(ctx, stone.Stone.OriginPlayerID, stone.Stone.SitoneID, 1); err != nil {
				return err
			}
		case StoneOutcomeMissing:
			if err := s.setInstanceStatus(ctx, stone.Stone.InstanceID, mongomodel.SitoneInstanceStatusMissing, now, nil); err != nil {
				return err
			}
		case StoneOutcomeDead:
			// Dead: instance kept, quantity NOT restored (design §10).
			if err := s.setInstanceStatus(ctx, stone.Stone.InstanceID, mongomodel.SitoneInstanceStatusDead, now, nil); err != nil {
				return err
			}
		case StoneOutcomeProtected:
			if err := s.setInstanceStatus(ctx, stone.Stone.InstanceID, mongomodel.SitoneInstanceStatusProtectedAtYansan, now, nil); err != nil {
				return err
			}
		case StoneOutcomeCaptured:
			// Captured by the defender.
			extra := bson.M{
				"player_id":      "",
				"team_id":        mission.DefenderTeamID,
				"cooldown_until": now.Add(CaptiveCooldownDuration),
			}
			if err := s.setInstanceStatus(ctx, stone.Stone.InstanceID, mongomodel.SitoneInstanceStatusCaptured, now, extra); err != nil {
				return err
			}
			_, err := s.DB.Collection(mongomodel.CaptiveRecordsCollection).InsertOne(ctx, mongomodel.CaptiveRecord{
				ID:             NewID("captive"),
				SitoneID:       stone.Stone.SitoneID,
				OriginalTeamID: mission.AttackerTeamID,
				CurrentTeamID:  mission.DefenderTeamID,
				InstanceID:     stone.Stone.InstanceID,
				CapturedAt:     now,
				CooldownUntil:  now.Add(CaptiveCooldownDuration),
				Status:         mongomodel.CaptiveRecordStatusCooldown,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) setInstanceStatus(ctx context.Context, instanceID string, status string, now time.Time, extra bson.M) error {
	set := bson.M{
		"status":     status,
		"updated_at": now,
	}
	for key, value := range extra {
		set[key] = value
	}
	_, err := s.DB.Collection(mongomodel.SitoneInstancesCollection).UpdateOne(
		ctx,
		bson.M{"_id": instanceID},
		bson.M{"$set": set},
	)
	return err
}

// buildResultSummary produces the PUBLIC-safe summary stored on the
// mission: outcome labels, stolen stone names, missing/dead/fatigued lists
// — never rolls, chances or A/D values (design §14).
func buildResultSummary(s *Service, resolution MissionResolution) bson.M {
	outcome := "returned"
	switch {
	case resolution.StealHappened:
		outcome = "steal_success"
	case resolution.AttackerWon:
		outcome = "success"
	case resolution.G < 0:
		outcome = "repelled"
	default:
		outcome = "stalemate"
	}

	stolenNames := make([]string, 0, len(resolution.Stolen))
	stolenIDs := make([]string, 0, len(resolution.Stolen))
	for _, stolen := range resolution.Stolen {
		stolenIDs = append(stolenIDs, stolen.SitoneID)
		if sitone, ok := s.Content.GetSitone(stolen.SitoneID); ok {
			stolenNames = append(stolenNames, sitone.Name)
		}
	}

	missing := make([]string, 0)
	dead := make([]string, 0)
	protected := make([]string, 0)
	captured := make([]string, 0)
	fatigued := make([]string, 0)
	for _, stone := range resolution.Stones {
		switch stone.Outcome {
		case StoneOutcomeMissing:
			missing = append(missing, stone.Stone.SitoneID)
		case StoneOutcomeDead:
			dead = append(dead, stone.Stone.SitoneID)
		case StoneOutcomeProtected:
			protected = append(protected, stone.Stone.SitoneID)
		case StoneOutcomeCaptured:
			captured = append(captured, stone.Stone.SitoneID)
		default:
			if stone.FatigueGain >= s.Engine.FatiguedLabelThreshold {
				fatigued = append(fatigued, stone.Stone.SitoneID)
			}
		}
	}

	summary := bson.M{
		"outcome":                   outcome,
		"stolen_sitone_names":       stolenNames,
		"stolen_sitone_ids":         stolenIDs,
		"missing_sitone_ids":        missing,
		"dead_sitone_ids":           dead,
		"protected_sitone_ids":      protected,
		"captured_sitone_ids":       captured,
		"fatigued_sitone_ids":       fatigued,
		"open_power_refund_granted": resolution.OpRefund > 0,
	}
	// scout_recon reveals a coarse defense-strength band even on failure.
	if resolution.HasScoutRecon {
		summary["defense_band"] = resolution.DefenseBand
	}
	return summary
}

// resolutionDetailPayload is the PRIVATE (staff-only) detail written into
// the event log.
func resolutionDetailPayload(mission mongomodel.AttackMission, resolution MissionResolution) bson.M {
	stones := make([]bson.M, 0, len(resolution.Stones))
	for _, stone := range resolution.Stones {
		stones = append(stones, bson.M{
			"instance_id":  stone.Stone.InstanceID,
			"sitone_id":    stone.Stone.SitoneID,
			"outcome":      stone.Outcome,
			"fatigue_gain": stone.FatigueGain,
		})
	}
	return bson.M{
		"seed_ref":       mission.RandomSeedRef,
		"a":              resolution.A,
		"d":              resolution.D,
		"g":              resolution.G,
		"attacker_won":   resolution.AttackerWon,
		"steal_happened": resolution.StealHappened,
		"stolen":         stolenPayload(resolution.Stolen),
		"stones":         stones,
		"op_refund":      resolution.OpRefund,
		"defense_band":   resolution.DefenseBand,
	}
}

func stolenPayload(stolen []StealCandidate) []bson.M {
	out := make([]bson.M, 0, len(stolen))
	for _, candidate := range stolen {
		out = append(out, bson.M{
			"sitone_id":       candidate.SitoneID,
			"owner_player_id": candidate.OwnerPlayerID,
			"recapture":       candidate.Recapture,
		})
	}
	return out
}

func (s *Service) logEvent(ctx context.Context, event mongomodel.EventLog) {
	_, _ = eventlog.Insert(ctx, s.DB, event)
}

func countByID(ids []string) map[string]int {
	counts := make(map[string]int, len(ids))
	for _, id := range ids {
		counts[id]++
	}
	return counts
}

func playerIDs(players []mongomodel.Player) []string {
	ids := make([]string, 0, len(players))
	for _, player := range players {
		ids = append(ids, player.ID)
	}
	return ids
}

// ActiveMissionForTeam returns the team's mission currently in voting or
// deployed state, if any.
func (s *Service) ActiveMissionForTeam(ctx context.Context, teamID string) (*mongomodel.AttackMission, error) {
	var mission mongomodel.AttackMission
	err := s.DB.Collection(mongomodel.AttackMissionsCollection).FindOne(
		ctx,
		bson.M{
			"attacker_team_id": teamID,
			"status": bson.M{"$in": bson.A{
				mongomodel.AttackMissionStatusVoting,
				mongomodel.AttackMissionStatusDeployed,
			}},
		},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&mission)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mission, nil
}
