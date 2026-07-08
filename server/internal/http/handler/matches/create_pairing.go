package matches

import (
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// CreatePairing godoc
// @Summary Create on-site match pairing
// @Description Creates a two-player quiz match and returns a short-lived QR pairing token for the authenticated host.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 201 {object} MatchPairingResponse
// @Success 200 {object} MatchPairingResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/pairings [post]
func (h *Handler) CreatePairing(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}
	if err := h.ensureBattleOpeningAllowed(r.Context(), "match creation failed"); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	openMatch, err := h.findOpenParticipantMatch(r.Context(), player.ID)
	if err == nil {
		h.writeExistingPairingOrConflict(w, r, openMatch, player.ID)
		return
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_open_lookup_failed", err))
		return
	}

	matchID, err := newID("match")
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_id_create_failed", err))
		return
	}
	sitoneIDs, err := h.defaultSitoneLoadout(r.Context(), player)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_default_loadout_failed", err))
		return
	}

	now := time.Now()
	match := mongomodel.Match{
		ID:           matchID,
		Mode:         mongomodel.MatchModePVP,
		Status:       mongomodel.MatchStatusWaiting,
		HostPlayerID: player.ID,
		OpenHostLock: player.ID,
		OpenPlayerLocks: []string{
			player.ID,
		},
		Players: []mongomodel.MatchPlayer{
			{
				PlayerID:  player.ID,
				Nickname:  player.Nickname,
				Kind:      mongomodel.MatchPlayerKindHuman,
				Ready:     false,
				Score:     0,
				SitoneIDs: sitoneIDs,
			},
		},
		CreatedAt: now,
	}

	if _, err := h.db.Collection(mongomodel.MatchesCollection).InsertOne(r.Context(), match); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			h.writePairingConflictForOpenMatch(w, r, player.ID)
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_insert_failed", err))
		return
	}

	session := h.sessions.Start(match)
	state, err := session.State(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	pairing, err := h.createPairingToken(r.Context(), match, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match pairing failed", "match_pairing_token_create_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, MatchPairingResponse{
		Match:     state,
		Token:     pairing.Token,
		ExpiresAt: pairing.ExpiresAt,
	})
}

func (h *Handler) writeExistingPairingOrConflict(
	w http.ResponseWriter,
	r *http.Request,
	match mongomodel.Match,
	playerID string,
) {
	if !matchCanIssuePairingToken(match, playerID) {
		writeOpenParticipantMatchConflict(w, r)
		return
	}

	response, err := h.pairingResponse(r.Context(), match, playerID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) writePairingConflictForOpenMatch(w http.ResponseWriter, r *http.Request, playerID string) {
	match, err := h.findOpenParticipantMatch(r.Context(), playerID)
	if err == nil {
		h.writeExistingPairingOrConflict(w, r, match, playerID)
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeOpenParticipantMatchConflict(w, r)
		return
	}
	httpx.WriteProblem(w, r, httpx.InternalServerError("match lookup failed", "match_open_lookup_failed", err))
}
