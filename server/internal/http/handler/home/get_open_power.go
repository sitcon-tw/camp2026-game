package home

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetOpenPower godoc
// @Summary Get open power balance
// @Description Returns the current player's spendable open power balance.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.OpenPowerBalanceResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/open-power [get]
func (h *Handler) GetOpenPower(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.OpenPowerBalanceResponse{
		Balance: 1280,
	})
}
