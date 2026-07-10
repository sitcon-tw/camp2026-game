package matches

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ScanPairing godoc
// @Summary Join on-site match pairing
// @Description Joins a waiting quiz match by scanning a short-lived QR pairing token.
// @Tags matches
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body ScanMatchPairingRequest true "Scan pairing request"
// @Success 200 {object} ScanMatchPairingResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/pairings/scan [post]
func (h *Handler) ScanPairing(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	var body ScanMatchPairingRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.Token = strings.ToUpper(strings.TrimSpace(body.Token))
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := h.ensureBattleOpeningAllowed(r.Context(), "match pairing failed"); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	pairing, err := h.findActivePairingToken(r.Context(), body.Token, time.Now())
	if err != nil {
		if isMissingPairingToken(err) {
			httpx.WriteProblem(w, r, httpx.NotFound("pairing token not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match pairing lookup failed", "match_pairing_lookup_failed", err))
		return
	}

	match, err := h.findMatchByID(r.Context(), pairing.MatchID)
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	if pairing.HostPlayerID == player.ID || match.HostPlayerID == player.ID {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "cannot join own pairing"))
		return
	}
	if match.Status != mongomodel.MatchStatusWaiting || !matchAcceptsHumanJoin(match) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "match is not joinable"))
		return
	}
	wasParticipant := isParticipant(match, player.ID)
	if !wasParticipant &&
		(len(match.Players) >= matchParticipantCapacity(match) ||
			len(humanParticipantIDs(match)) >= matchHumanCapacity(match)) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "match is full"))
		return
	}
	if !wasParticipant {
		if err := h.ensureBattleOpeningAllowed(r.Context(), "match pairing failed"); err != nil {
			httpx.WriteProblem(w, r, err)
			return
		}
		if err := h.ensureNoOpenParticipantMatch(r.Context(), player.ID); err != nil {
			if errors.Is(err, errOpenParticipantMatchExists) {
				h.writeExistingOpenParticipantMatch(w, r, player.ID)
				return
			}
			httpx.WriteProblem(w, r, httpx.InternalServerError("match pairing failed", "match_open_lookup_failed", err))
			return
		}
		if err := h.ensureSameTeamBattleAllowed(r.Context(), match, player); err != nil {
			httpx.WriteProblem(w, r, err)
			return
		}
		if err := h.ensureOpponentBattleLimitAllowed(r.Context(), match, player); err != nil {
			httpx.WriteProblem(w, r, err)
			return
		}
		if matchPairingTokenSingleUse(match) {
			consumed, err := h.consumePairingToken(r.Context(), pairing, player.ID)
			if err != nil {
				httpx.WriteProblem(w, r, httpx.InternalServerError("match pairing failed", "match_pairing_consume_failed", err))
				return
			}
			if !consumed {
				httpx.WriteProblem(w, r, httpx.NotFound("pairing token not found"))
				return
			}
		}
	}

	session, err := h.sessions.GetOrLoad(r.Context(), match.ID)
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	state, err := session.Join(r.Context(), player)
	if err != nil {
		if errors.Is(err, errOpenParticipantMatchExists) {
			h.writeExistingOpenParticipantMatch(w, r, player.ID)
			return
		}
		if errors.Is(err, errMatchSaveConflict) {
			writeMatchProblem(w, r, err)
			return
		}
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}
