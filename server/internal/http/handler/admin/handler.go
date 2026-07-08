package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
)

const (
	CookieName        = "camp2026_admin"
	sessionMessage    = "camp2026-admin-session-v1"
	adminCookieMaxAge = 365 * 24 * 60 * 60
)

type Dependencies struct {
	Content           *content.Store
	MongoDB           *mongo.Database
	AdminPassword     string
	AdminCookieSecure bool
	Broker            *playerevents.Broker
}

type Handler struct {
	content           *content.Store
	db                *mongo.Database
	adminPassword     string
	adminCookieSecure bool
	broker            *playerevents.Broker
}

func New(dep Dependencies) *Handler {
	return &Handler{
		content:           dep.Content,
		db:                dep.MongoDB,
		adminPassword:     strings.TrimSpace(dep.AdminPassword),
		adminCookieSecure: dep.AdminCookieSecure,
		broker:            dep.Broker,
	}
}

func (h *Handler) RegisterRoutes(api chi.Router) {
	api.Post("/admin/login", h.Login)
	api.Post("/admin/logout", h.Logout)
	api.Get("/admin/dashboard", h.Dashboard)
	api.Get("/admin/gift-history", h.GiftHistory)
	api.Get("/admin/history", h.History)
	api.Get("/admin/student-changes", h.ListStudentChanges)
	api.Post("/admin/inventory-trims", h.CreateInventoryTrim)
	api.Put("/admin/players/{playerID}/balance", h.UpdatePlayerBalance)
	api.Get("/admin/settings", h.GetSettings)
	api.Put("/admin/settings", h.UpdateSettings)
	api.Get("/admin/community-stands", h.ListCommunityStands)
	api.Get("/admin/community-stand-claims", h.ListCommunityStandClaims)
	api.Post("/admin/community-stands", h.CreateCommunityStand)
	api.Put("/admin/community-stands/{standID}", h.UpdateCommunityStand)
	api.Delete("/admin/community-stands/{standID}", h.DeleteCommunityStand)
	api.Put("/admin/teams/{teamID}", h.UpdateTeam)
}

type LoginRequest struct {
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Authenticated bool `json:"authenticated" example:"true"`
}

type SettingsRequest struct {
	ComputerBattlesEnabled     *bool  `json:"computerBattlesEnabled" validate:"required"`
	SameTeamBattlesEnabled     *bool  `json:"sameTeamBattlesEnabled" validate:"required"`
	ComputerEasyAccuracy       *int   `json:"computerEasyAccuracy" validate:"required,min=0,max=100"`
	ComputerNormalAccuracy     *int   `json:"computerNormalAccuracy" validate:"required,min=0,max=100"`
	ComputerHardAccuracy       *int   `json:"computerHardAccuracy" validate:"required,min=0,max=100"`
	ClassTimeBattleLockEnabled *bool  `json:"classTimeBattleLockEnabled" validate:"required"`
	ClassTimeBattleLockStart   string `json:"classTimeBattleLockStart" validate:"required"`
	ClassTimeBattleLockEnd     string `json:"classTimeBattleLockEnd" validate:"required"`
	BattleOpeningOverride      string `json:"battleOpeningOverride" validate:"required,oneof=schedule force_open force_closed"`
	MaintenanceMode            string `json:"maintenanceMode"`
	MaintenanceMessage         string `json:"maintenanceMessage"`
}

type SettingsResponse struct {
	ComputerBattlesEnabled     bool   `json:"computerBattlesEnabled"`
	SameTeamBattlesEnabled     bool   `json:"sameTeamBattlesEnabled"`
	ComputerEasyAccuracy       int    `json:"computerEasyAccuracy"`
	ComputerNormalAccuracy     int    `json:"computerNormalAccuracy"`
	ComputerHardAccuracy       int    `json:"computerHardAccuracy"`
	ClassTimeBattleLockEnabled bool   `json:"classTimeBattleLockEnabled"`
	ClassTimeBattleLockStart   string `json:"classTimeBattleLockStart"`
	ClassTimeBattleLockEnd     string `json:"classTimeBattleLockEnd"`
	BattleOpeningOverride      string `json:"battleOpeningOverride"`
	BattleOpeningLocked        bool   `json:"battleOpeningLocked"`
	MaintenanceMode            string `json:"maintenanceMode"`
	MaintenanceMessage         string `json:"maintenanceMessage"`
	MaintenanceActive          bool   `json:"maintenanceActive"`
}

// Login godoc
// @Summary Login as admin
// @Description Uses the ADMIN_PASSWORD environment variable to create an admin session cookie.
// @Tags admin
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Admin login request"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.requireEnabled(w, r) {
		return
	}

	var body LoginRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.Password = strings.TrimSpace(body.Password)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(h.adminPassword)) != 1 {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "invalid admin password"))
		return
	}

	http.SetCookie(w, h.sessionCookie(adminSessionValue(h.adminPassword), adminCookieMaxAge))
	httpx.WriteJSON(w, http.StatusOK, LoginResponse{Authenticated: true})
}

// Logout godoc
// @Summary Logout admin session
// @Tags admin
// @Produce json
// @Success 204
// @Router /admin/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, h.sessionCookie("", -1))
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

// GetSettings godoc
// @Summary Get admin game settings
// @Tags admin
// @Produce json
// @Success 200 {object} SettingsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	settings, err := gamecontrol.ReadSettings(r.Context(), h.db)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("settings unavailable", "admin_settings_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, settingsResponse(settings))
}

// UpdateSettings godoc
// @Summary Update admin game settings
// @Tags admin
// @Accept json
// @Produce json
// @Param request body SettingsRequest true "Admin settings request"
// @Success 200 {object} SettingsResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/settings [put]
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	var body SettingsRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := validateSettingsRequest(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	maintenanceMode := body.MaintenanceMode
	if maintenanceMode == "" {
		maintenanceMode = gamecontrol.MaintenanceModeOff
	}

	settings, err := gamecontrol.SaveSettings(r.Context(), h.db, gamecontrol.Settings{
		ComputerBattlesEnabled:     *body.ComputerBattlesEnabled,
		SameTeamBattlesDisabled:    !*body.SameTeamBattlesEnabled,
		ComputerEasyAccuracy:       *body.ComputerEasyAccuracy,
		ComputerNormalAccuracy:     *body.ComputerNormalAccuracy,
		ComputerHardAccuracy:       *body.ComputerHardAccuracy,
		ClassTimeBattleLockEnabled: *body.ClassTimeBattleLockEnabled,
		ClassTimeBattleLockStart:   body.ClassTimeBattleLockStart,
		ClassTimeBattleLockEnd:     body.ClassTimeBattleLockEnd,
		BattleOpeningOverride:      body.BattleOpeningOverride,
		MaintenanceMode:            maintenanceMode,
		MaintenanceMessage:         strings.TrimSpace(body.MaintenanceMessage),
		MaintenanceStartedAt:       maintenanceStartedAt(maintenanceMode),
	})
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("settings update failed", "admin_settings_update_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, settingsResponse(settings))
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !h.requireEnabled(w, r) {
		return false
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "admin authentication required"))
		return false
	}

	expected := adminSessionValue(h.adminPassword)
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expected)) != 1 {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "admin authentication required"))
		return false
	}
	return true
}

func (h *Handler) requireEnabled(w http.ResponseWriter, r *http.Request) bool {
	if h.adminPassword != "" {
		return true
	}
	httpx.WriteProblem(w, r, httpx.ServiceUnavailable("admin is disabled"))
	return false
}

func (h *Handler) requireDatabase(w http.ResponseWriter, r *http.Request) bool {
	if h.db != nil {
		return true
	}
	httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
	return false
}

func validateSettingsRequest(body SettingsRequest) error {
	var details []httpx.ErrorDetail
	if !gamecontrol.ValidClockTime(body.ClassTimeBattleLockStart) {
		details = append(details, httpx.ErrorDetail{
			Location: "body.classTimeBattleLockStart",
			Message:  "classTimeBattleLockStart must use HH:MM format",
		})
	}
	if !gamecontrol.ValidClockTime(body.ClassTimeBattleLockEnd) {
		details = append(details, httpx.ErrorDetail{
			Location: "body.classTimeBattleLockEnd",
			Message:  "classTimeBattleLockEnd must use HH:MM format",
		})
	}
	if !gamecontrol.ValidBattleOpeningOverride(body.BattleOpeningOverride) {
		details = append(details, httpx.ErrorDetail{
			Location: "body.battleOpeningOverride",
			Message:  "battleOpeningOverride must be one of: schedule force_open force_closed",
		})
	}
	if body.MaintenanceMode != "" && !gamecontrol.ValidMaintenanceMode(body.MaintenanceMode) {
		details = append(details, httpx.ErrorDetail{
			Location: "body.maintenanceMode",
			Message:  "maintenanceMode must be one of: off draining active",
		})
	}
	if len(details) > 0 {
		return httpx.UnprocessableEntity("invalid request body", details...)
	}
	return nil
}

func (h *Handler) requireContent(w http.ResponseWriter, r *http.Request) bool {
	if h.content != nil {
		return true
	}
	httpx.WriteProblem(w, r, httpx.ServiceUnavailable("content store is unavailable"))
	return false
}

func (h *Handler) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.adminCookieSecure,
	}
}

func adminSessionValue(password string) string {
	mac := hmac.New(sha256.New, []byte(password))
	_, _ = mac.Write([]byte(sessionMessage))
	return hex.EncodeToString(mac.Sum(nil))
}

func settingsResponse(settings gamecontrol.Settings) SettingsResponse {
	settings.Normalize()
	return SettingsResponse{
		ComputerBattlesEnabled:     settings.ComputerBattlesEnabled,
		SameTeamBattlesEnabled:     settings.SameTeamBattlesEnabled(),
		ComputerEasyAccuracy:       settings.ComputerEasyAccuracy,
		ComputerNormalAccuracy:     settings.ComputerNormalAccuracy,
		ComputerHardAccuracy:       settings.ComputerHardAccuracy,
		ClassTimeBattleLockEnabled: settings.ClassTimeBattleLockEnabled,
		ClassTimeBattleLockStart:   settings.ClassTimeBattleLockStart,
		ClassTimeBattleLockEnd:     settings.ClassTimeBattleLockEnd,
		BattleOpeningOverride:      settings.BattleOpeningOverride,
		BattleOpeningLocked:        settings.BattleOpeningLocked(time.Now()),
		MaintenanceMode:            settings.MaintenanceMode,
		MaintenanceMessage:         settings.MaintenanceMessage,
		MaintenanceActive:          settings.MaintenanceActive(),
	}
}

func maintenanceStartedAt(mode string) time.Time {
	if mode == gamecontrol.MaintenanceModeOff {
		return time.Time{}
	}
	return time.Now().UTC()
}
