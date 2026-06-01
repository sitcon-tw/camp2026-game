package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetSitone godoc
// @Summary Get owned sitone
// @Description Returns one sitone owned by the current player.
// @Tags Me
// @Produce json
// @Security AuthCookieAuth
// @Param sitoneID path string true "Sitone ID"
// @Success 200 {object} apimodel.SitoneSummary
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/sitones/{sitoneID} [get]
func (h *Handler) GetSitone(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, exampleSitone())
}

func exampleSitone() apimodel.SitoneSummary {
	return apimodel.SitoneSummary{
		ID:           "S9K2QA",
		DefinitionID: "sitone-engineering",
		Name:         "Engineering Sitone",
		Type:         "engineering",
		Rarity:       "rare",
		Style:        "default",
	}
}
