package matches

import (
	"context"
	"net/http"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

func (h *Handler) ensureBattleOpeningAllowed(ctx context.Context, message string) error {
	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return httpx.InternalServerError(message, "match_opening_settings_lookup_failed", err)
	}
	if settings.MaintenanceBlocksNewMatches() {
		return httpx.NewError(http.StatusServiceUnavailable, "maintenance is active")
	}
	if settings.BattleOpeningLocked(time.Now()) {
		return httpx.NewError(http.StatusConflict, "battle opening is locked")
	}
	return nil
}

func (h *Handler) ensureMaintenanceAllowsNewMatch(ctx context.Context, message string) error {
	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return httpx.InternalServerError(message, "maintenance_settings_lookup_failed", err)
	}
	if settings.MaintenanceBlocksNewMatches() {
		return httpx.NewError(http.StatusServiceUnavailable, "maintenance is active")
	}
	return nil
}

func (h *Handler) ensureMaintenanceAllowsWrites(ctx context.Context, message string) error {
	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return httpx.InternalServerError(message, "maintenance_settings_lookup_failed", err)
	}
	if settings.MaintenanceBlocksWrites() {
		return httpx.NewError(http.StatusServiceUnavailable, "maintenance is active")
	}
	return nil
}
