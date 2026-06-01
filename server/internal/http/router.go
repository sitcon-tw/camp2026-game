package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"

	authhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/auth"
	bingohandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/bingo"
	cataloghandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/catalog"
	homehandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/home"
	matcheshandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/matches"
	qrcodehandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/qrcode"
	staffhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/staff"
	storagehandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/storage"
	systemhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/system"
	worldbosshandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/worldboss"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

type Dependencies struct {
	Log            *slog.Logger
	RequestTimeout time.Duration
	MongoClient    *mongo.Client
}

func NewRouter(dep Dependencies) http.Handler {
	if dep.Log == nil {
		dep.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if dep.RequestTimeout <= 0 {
		dep.RequestTimeout = 10 * time.Second
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.StripSlashes)
	r.Use(chimw.Timeout(dep.RequestTimeout))
	r.Use(recoverer(dep.Log))
	r.Use(requestLogger(dep.Log))

	r.Route("/api", func(api chi.Router) {
		registerRoutes(api, dep)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, httpx.NotFound("not found"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusMethodNotAllowed, "method not allowed"))
	})

	return r
}

func registerRoutes(api chi.Router, dep Dependencies) {
	systemhandler.New(systemhandler.Dependencies{
		MongoClient: dep.MongoClient,
	}).RegisterRoutes(api)
	authhandler.New().RegisterRoutes(api)
	homehandler.New().RegisterRoutes(api)
	bingohandler.New().RegisterRoutes(api)
	matcheshandler.New().RegisterRoutes(api)
	qrcodehandler.New().RegisterRoutes(api)
	worldbosshandler.New().RegisterRoutes(api)
	storagehandler.New().RegisterRoutes(api)
	cataloghandler.New().RegisterRoutes(api)
	staffhandler.New().RegisterRoutes(api)

	registerSwaggerRoutes(api)
}
