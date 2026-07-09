package territory

// Central sweeper (design §0.1 decision 5): a single background goroutine
// scans for overdue votes, due mission resolutions and due yansan
// processes. Everything survives restarts naturally because state lives in
// MongoDB and the sweeper re-picks up due documents on boot.

import (
	"context"
	"io"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// DefaultSweepInterval is how often the sweeper scans for due work.
const DefaultSweepInterval = 15 * time.Second

const sweeperOperationTimeout = 30 * time.Second

// Sweeper runs the periodic territory resolution loop.
type Sweeper struct {
	service  *Service
	log      *slog.Logger
	interval time.Duration
	stop     chan struct{}
}

// StartSweeper launches the background sweeper goroutine and returns its
// handle. Call Stop to terminate it.
func StartSweeper(service *Service, log *slog.Logger, interval time.Duration) *Sweeper {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if interval <= 0 {
		interval = DefaultSweepInterval
	}
	sweeper := &Sweeper{
		service:  service,
		log:      log,
		interval: interval,
		stop:     make(chan struct{}),
	}
	go sweeper.run()
	return sweeper
}

// Stop terminates the sweeper loop.
func (s *Sweeper) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

func (s *Sweeper) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Sweep once at boot to catch up on work that came due while down.
	s.sweepOnce()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.sweepOnce()
		}
	}
}

func (s *Sweeper) sweepOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), sweeperOperationTimeout)
	defer cancel()

	if err := s.expireOverdueVotes(ctx); err != nil {
		s.log.Error("territory sweeper: expire votes", "error", err)
	}
	if err := s.resolveDueMissions(ctx); err != nil {
		s.log.Error("territory sweeper: resolve missions", "error", err)
	}
	if err := s.resolveDueProcesses(ctx); err != nil {
		s.log.Error("territory sweeper: resolve processes", "error", err)
	}
}

func (s *Sweeper) expireOverdueVotes(ctx context.Context) error {
	cursor, err := s.service.DB.Collection(mongomodel.AttackMissionsCollection).Find(ctx, bson.M{
		"status":        mongomodel.AttackMissionStatusVoting,
		"vote_deadline": bson.M{"$lte": time.Now()},
	})
	if err != nil {
		return err
	}
	var missions []mongomodel.AttackMission
	if err := cursor.All(ctx, &missions); err != nil {
		return err
	}
	for _, mission := range missions {
		if _, err := s.service.ExpireMissionIfDue(ctx, mission); err != nil {
			s.log.Error("territory sweeper: expire mission", "mission_id", mission.ID, "error", err)
		}
	}
	return nil
}

func (s *Sweeper) resolveDueMissions(ctx context.Context) error {
	cursor, err := s.service.DB.Collection(mongomodel.AttackMissionsCollection).Find(ctx, bson.M{
		"status":     mongomodel.AttackMissionStatusDeployed,
		"resolve_at": bson.M{"$lte": time.Now()},
	})
	if err != nil {
		return err
	}
	var missions []mongomodel.AttackMission
	if err := cursor.All(ctx, &missions); err != nil {
		return err
	}
	for _, mission := range missions {
		err := s.service.WithLock(ctx, "mission:"+mission.ID, func(ctx context.Context) error {
			return s.service.ResolveDueMission(ctx, mission)
		})
		if err != nil {
			s.log.Error("territory sweeper: resolve mission", "mission_id", mission.ID, "error", err)
		} else {
			s.log.Info("territory sweeper: mission resolved", "mission_id", mission.ID)
		}
	}
	return nil
}

func (s *Sweeper) resolveDueProcesses(ctx context.Context) error {
	cursor, err := s.service.DB.Collection(mongomodel.YansanProcessesCollection).Find(ctx, bson.M{
		"status":     mongomodel.YansanProcessStatusPending,
		"resolve_at": bson.M{"$lte": time.Now()},
	})
	if err != nil {
		return err
	}
	var processes []mongomodel.YansanProcess
	if err := cursor.All(ctx, &processes); err != nil {
		return err
	}
	for _, process := range processes {
		err := s.service.WithLock(ctx, "process:"+process.ID, func(ctx context.Context) error {
			return s.service.ResolveDueProcess(ctx, process)
		})
		if err != nil {
			s.log.Error("territory sweeper: resolve process", "process_id", process.ID, "kind", process.Kind, "error", err)
		} else {
			s.log.Info("territory sweeper: process resolved", "process_id", process.ID, "kind", process.Kind)
		}
	}
	return nil
}
