package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/config"
	"github.com/sitcon-tw/camp2026-game/internal/content"
	httpserver "github.com/sitcon-tw/camp2026-game/internal/http"
	"github.com/sitcon-tw/camp2026-game/internal/mongodb"
)

type Application struct {
	Config      config.Config
	Log         *slog.Logger
	HTTPServer  *http.Server
	Content     *content.Store
	MongoClient *drivermongo.Client
	MongoDB     *drivermongo.Database
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	contentStore, err := content.Load(cfg.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("load content: %w", err)
	}

	mongoClient, err := mongodb.NewClient(ctx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}
	mongoDB := mongoClient.Database(cfg.MongoDatabase)
	if err := mongodb.EnsureIndexes(ctx, mongoDB); err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()
		_ = mongoClient.Disconnect(disconnectCtx)

		return nil, err
	}

	handler := httpserver.NewRouter(httpserver.Dependencies{
		Log:               log,
		RequestTimeout:    cfg.HTTP.RequestTimeout,
		Content:           contentStore,
		MongoClient:       mongoClient,
		MongoDB:           mongoDB,
		AdminPassword:     cfg.AdminPassword,
		AdminCookieSecure: cfg.AdminCookieSecure,
	})

	return &Application{
		Config:      cfg,
		Log:         log,
		HTTPServer:  httpserver.NewServer(cfg.HTTP, handler),
		Content:     contentStore,
		MongoClient: mongoClient,
		MongoDB:     mongoDB,
	}, nil
}
