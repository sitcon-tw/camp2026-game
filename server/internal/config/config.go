package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv          = "local"
	defaultServiceName     = "camp2026-game-api"
	defaultAppVersion      = "0.1.0"
	defaultLogLevel        = "info"
	defaultHTTPAddr        = ":8080"
	defaultMongoURI        = "mongodb://camp2026:camp2026@localhost:27017/camp2026?authSource=admin"
	defaultMongoDatabase   = "camp2026"
	defaultRequestTimeout  = 10 * time.Second
	defaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	Env             string
	ServiceName     string
	Version         string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	HTTP            HTTPConfig
	MongoURI        string
	MongoDatabase   string
}

type HTTPConfig struct {
	Addr              string
	RequestTimeout    time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func Load() (Config, error) {
	loadDotEnv()

	cfg := Config{
		Env:             stringValue("APP_ENV", defaultAppEnv),
		ServiceName:     stringValue("SERVICE_NAME", defaultServiceName),
		Version:         stringValue("APP_VERSION", defaultAppVersion),
		ShutdownTimeout: durationValue("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		HTTP: HTTPConfig{
			Addr:              stringValue("HTTP_ADDR", defaultHTTPAddr),
			RequestTimeout:    durationValue("REQUEST_TIMEOUT", defaultRequestTimeout),
			ReadHeaderTimeout: durationValue("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       durationValue("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      durationValue("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:       durationValue("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		MongoURI:      stringValue("MONGODB_URI", defaultMongoURI),
		MongoDatabase: stringValue("MONGODB_DATABASE", defaultMongoDatabase),
	}

	level, err := parseLogLevel(stringValue("LOG_LEVEL", defaultLogLevel))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) validate() error {
	var errs []error
	if strings.TrimSpace(cfg.Env) == "" {
		errs = append(errs, errors.New("APP_ENV is required"))
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		errs = append(errs, errors.New("SERVICE_NAME is required"))
	}
	if strings.TrimSpace(cfg.Version) == "" {
		errs = append(errs, errors.New("APP_VERSION is required"))
	}
	if strings.TrimSpace(cfg.HTTP.Addr) == "" {
		errs = append(errs, errors.New("HTTP_ADDR is required"))
	}
	if strings.TrimSpace(cfg.MongoURI) == "" {
		errs = append(errs, errors.New("MONGODB_URI is required"))
	}
	if strings.TrimSpace(cfg.MongoDatabase) == "" {
		errs = append(errs, errors.New("MONGODB_DATABASE is required"))
	}
	if cfg.HTTP.RequestTimeout <= 0 {
		errs = append(errs, errors.New("REQUEST_TIMEOUT must be positive"))
	}
	if cfg.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("SHUTDOWN_TIMEOUT must be positive"))
	}
	return errors.Join(errs...)
}

func loadDotEnv() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("server/.env")
}

func stringValue(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func durationValue(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}
