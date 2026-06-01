package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/config"
	httpserver "github.com/sitcon-tw/camp2026-game/internal/http"
	"github.com/sitcon-tw/camp2026-game/internal/mongodb"
)

type Application struct {
	Config      config.Config
	Log         *slog.Logger
	HTTPServer  *http.Server
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

	mongoClient, err := mongodb.NewClient(ctx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}
	mongoDB := mongoClient.Database(cfg.MongoDatabase)

	handler := httpserver.NewRouter(httpserver.Dependencies{
		Log:            log,
		RequestTimeout: cfg.HTTP.RequestTimeout,
		MongoClient:    mongoClient,
		MongoDB:        mongoDB,
	})

	return &Application{
		Config:      cfg,
		Log:         log,
		HTTPServer:  httpserver.NewServer(cfg.HTTP, handler),
		MongoClient: mongoClient,
		MongoDB:     mongoDB,
	}, nil
}
