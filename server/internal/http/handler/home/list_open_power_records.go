package home

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListOpenPowerRecords godoc
// @Summary List open power records
// @Description Lists the current player's open power ledger records.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.OpenPowerRecordListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/open-power/records [get]
func (h *Handler) ListOpenPowerRecords(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.OpenPowerRecordListResponse{
		Records: []apimodel.OpenPowerRecordSummary{
			{
				RecordID:  "N6T3ZA9K",
				Amount:    120,
				Reason:    "match_win",
				Source:    "match",
				CreatedAt: "2026-07-24T10:35:00+08:00",
			},
		},
	})
}
