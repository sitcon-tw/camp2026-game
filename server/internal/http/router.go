package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	adminhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/admin"
	authhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/auth"
	cataloghandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/catalog"
	communitystandshandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/communitystands"
	fusionshandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/fusions"
	leaderboardshandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/leaderboards"
	matcheshandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/matches"
	mehandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/me"
	qrhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/qr"
	shophandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/shop"
	staffhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/staff"
	systemhandler "github.com/sitcon-tw/camp2026-game/internal/http/handler/system"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
)

type Dependencies struct {
	Log                  *slog.Logger
	RequestTimeout       time.Duration
	Content              *content.Store
	MongoClient          *mongo.Client
	MongoDB              *mongo.Database
	AdminPassword        string
	AdminCookieSecure    bool
	RecoverMatchSessions bool
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
	r.Use(timeoutExceptMatchEvents(dep.RequestTimeout))
	r.Use(recoverer(dep.Log))
	r.Use(requestLogger(dep.Log))

	r.Route("/api", func(api chi.Router) {
		api.Use(sameOriginUnsafeRequestGuard())
		registerRoutes(api, dep)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, httpx.NotFound("not found"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusMethodNotAllowed, "method not allowed"))
	})

	r.Handle("/metrics", promhttp.Handler())

	return r
}

func timeoutExceptMatchEvents(timeout time.Duration) func(http.Handler) http.Handler {
	timeoutMiddleware := chimw.Timeout(timeout)
	return func(next http.Handler) http.Handler {
		timeoutHandler := timeoutMiddleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSSEPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			timeoutHandler.ServeHTTP(w, r)
		})
	}
}

func isSSEPath(path string) bool {
	return (strings.HasPrefix(path, "/api/matches/") && strings.HasSuffix(path, "/events")) ||
		path == "/api/me/events"
}

func registerRoutes(api chi.Router, dep Dependencies) {
	playerEventsBroker := playerevents.NewBroker()

	adminhandler.New(adminhandler.Dependencies{
		Content:           dep.Content,
		MongoDB:           dep.MongoDB,
		AdminPassword:     dep.AdminPassword,
		AdminCookieSecure: dep.AdminCookieSecure,
		Broker:            playerEventsBroker,
	}).RegisterRoutes(api)
	systemhandler.New(systemhandler.Dependencies{
		MongoClient: dep.MongoClient,
	}).RegisterRoutes(api)
	authhandler.New(authhandler.Dependencies{
		MongoDB: dep.MongoDB,
	}).RegisterRoutes(api)
	cataloghandler.New(cataloghandler.Dependencies{
		Content: dep.Content,
	}).RegisterRoutes(api)
	qrhandler.New(qrhandler.Dependencies{
		MongoDB: dep.MongoDB,
	}).RegisterRoutes(api.With(authctx.RequireStaff(dep.MongoDB)))
	mehandler.New(mehandler.Dependencies{
		Content: dep.Content,
		MongoDB: dep.MongoDB,
		Broker:  playerEventsBroker,
	}).RegisterRoutes(api.With(authctx.RequirePlayer(dep.MongoDB)))
	leaderboardshandler.New(leaderboardshandler.Dependencies{
		Content: dep.Content,
		MongoDB: dep.MongoDB,
	}).RegisterRoutes(api.With(authctx.RequirePlayer(dep.MongoDB)))
	matcheshandler.New(matcheshandler.Dependencies{
		Content:            dep.Content,
		MongoClient:        dep.MongoClient,
		MongoDB:            dep.MongoDB,
		RecoverOpenMatches: dep.RecoverMatchSessions,
	}).RegisterRoutes(api.With(authctx.RequirePlayer(dep.MongoDB)))
	fusionshandler.New(fusionshandler.Dependencies{
		Content:     dep.Content,
		MongoClient: dep.MongoClient,
		MongoDB:     dep.MongoDB,
		Broker:      playerEventsBroker,
	}).RegisterRoutes(api.With(authctx.RequirePlayer(dep.MongoDB)))
	shophandler.New(shophandler.Dependencies{
		Content:     dep.Content,
		MongoClient: dep.MongoClient,
		MongoDB:     dep.MongoDB,
		Broker:      playerEventsBroker,
	}).RegisterRoutes(api.With(authctx.RequirePlayer(dep.MongoDB)))
	communityStandsHandler := communitystandshandler.New(communitystandshandler.Dependencies{
		Content: dep.Content,
		MongoDB: dep.MongoDB,
		Broker:  playerEventsBroker,
	})
	communityStandsHandler.RegisterPublicRoutes(api)
	communityStandsHandler.RegisterRoutes(api.With(authctx.RequirePlayer(dep.MongoDB)))
	staffhandler.New(staffhandler.Dependencies{
		Content: dep.Content,
		MongoDB: dep.MongoDB,
		Broker:  playerEventsBroker,
	}).RegisterRoutes(api.With(authctx.RequireStaff(dep.MongoDB)))

	registerSwaggerRoutes(api)
}
