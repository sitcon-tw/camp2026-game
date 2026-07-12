package fronts

import (
	"context"
	"net/http"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

func (h *Handler) ensureTerritoryBattleOpeningAllowed(ctx context.Context, commandKind string) error {
	if !isTerritoryBattleCommand(commandKind) {
		return nil
	}
	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return httpx.InternalServerError("front command failed", "front_opening_settings_lookup_failed", err)
	}
	if settings.BattleOpeningLocked(time.Now()) {
		return httpx.NewError(http.StatusConflict, "battle opening is locked")
	}
	return nil
}

func isTerritoryBattleCommand(commandKind string) bool {
	return territoryCommandIsSupported(commandKind)
}
