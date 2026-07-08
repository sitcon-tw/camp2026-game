package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/sitcon-tw/camp2026-game/tgbot/internal/bot"
	"github.com/sitcon-tw/camp2026-game/tgbot/internal/config"
	"github.com/sitcon-tw/camp2026-game/tgbot/internal/store"
	"github.com/sitcon-tw/camp2026-game/tgbot/internal/telegram"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		_, _ = fmt.Fprintf(os.Stderr, "tgbot failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoClient, err := connectMongo(ctx, cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		_ = mongoClient.Disconnect(shutdownCtx)
	}()

	mongoStore := store.NewMongoStore(mongoClient.Database(cfg.MongoDatabase))
	if err := mongoStore.EnsureIndexes(ctx); err != nil {
		return err
	}

	tgClient := telegram.NewClient(&http.Client{
		Timeout: cfg.HTTPClientTimeout,
	}, cfg.BotToken, cfg.APIBaseURL)
	me, err := getTelegramBotProfile(ctx, tgClient, log)
	if err != nil {
		return err
	}
	if me.Username == "" {
		return errors.New("telegram bot profile is missing username")
	}

	service, err := bot.NewService(bot.Dependencies{
		Store:            mongoStore,
		Messenger:        tgClient,
		BotUsername:      me.Username,
		LoginBaseURL:     cfg.LoginBaseURL,
		GroupTeamMap:     cfg.GroupTeamMap,
		InitialSitoneIDs: cfg.InitialSitoneIDs,
		RequestTTL:       cfg.RequestTTL,
		Log:              log,
	})
	if err != nil {
		return err
	}

	log.Info("telegram bot started",
		"bot_username", me.Username,
		"group_count", len(cfg.GroupTeamMap),
	)
	return bot.NewRunner(tgClient, service, cfg.PollTimeout, log).Run(ctx)
}

type botProfileGetter interface {
	GetMe(ctx context.Context) (telegram.User, error)
}

func getTelegramBotProfile(ctx context.Context, getter botProfileGetter, log *slog.Logger) (telegram.User, error) {
	backoff := time.Second
	for {
		me, err := getter.GetMe(ctx)
		if err == nil {
			return me, nil
		}
		if ctx.Err() != nil {
			return telegram.User{}, ctx.Err()
		}
		if !transientTelegramError(err) {
			return telegram.User{}, fmt.Errorf("get telegram bot profile: %w", err)
		}
		log.Warn("telegram getMe failed; retrying", "error", err, "retry_after", backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return telegram.User{}, ctx.Err()
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func transientTelegramError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "bad gateway") ||
		strings.Contains(message, "gateway timeout") ||
		strings.Contains(message, "service unavailable") ||
		strings.Contains(message, "too many requests") ||
		strings.Contains(message, "telegram getme request failed") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "temporary") ||
		strings.Contains(message, "connection reset") ||
		strings.Contains(message, "eof")
}

func connectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	return client, nil
}
