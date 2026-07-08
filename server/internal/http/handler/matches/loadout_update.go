package matches

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// UpdateLoadout godoc
// @Summary Update match sitone loadout
// @Description Updates the authenticated player's sitone loadout for a waiting match and saves it as the player's default loadout.
// @Tags matches
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body UpdateLoadoutRequest true "Update loadout request"
// @Success 200 {object} UpdateLoadoutResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/loadout [put]
func (h *Handler) UpdateLoadout(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	var body UpdateLoadoutRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := h.ensureMaintenanceAllowsWrites(r.Context(), "loadout update failed"); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	sitoneIDs, err := h.validateOwnedSitoneLoadout(r.Context(), player.ID, body.SitoneIDs)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	session, err := h.sessions.GetOrLoad(r.Context(), chi.URLParam(r, "matchID"))
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	state, err := session.UpdateLoadout(r.Context(), player.ID, sitoneIDs)
	if err != nil {
		if errors.Is(err, errMatchSaveConflict) || errors.Is(err, errOpenParticipantMatchExists) {
			writeMatchProblem(w, r, err)
			return
		}
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := h.saveDefaultSitoneLoadout(r.Context(), player.ID, sitoneIDs); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("loadout update failed", "match_loadout_save_default_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, state)
}
