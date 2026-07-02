package matches

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	computerPlayerID      = "computer"
	computerNickname      = "電腦對手"
	computerDefaultSitone = "stone_engineering_base"
)

// ComputerSettings godoc
// @Summary Get computer battle availability
// @Description Returns whether computer battles are currently enabled for players.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} ComputerBattleSettingsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/computer/settings [get]
func (h *Handler) ComputerSettings(w http.ResponseWriter, r *http.Request) {
	_, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	settings, err := gamecontrol.ReadSettings(r.Context(), h.db)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("computer battle settings unavailable", "computer_settings_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ComputerBattleSettingsResponse{
		Enabled: settings.ComputerBattlesEnabled,
	})
}

// CreateComputer godoc
// @Summary Create computer match room
// @Description Creates a two-player quiz match room against the system-controlled computer opponent.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 201 {object} CreateMatchResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/computer [post]
func (h *Handler) CreateComputer(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	settings, err := gamecontrol.ReadSettings(r.Context(), h.db)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("computer battle settings unavailable", "computer_settings_lookup_failed", err))
		return
	}
	if !settings.ComputerBattlesEnabled {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "computer battles are disabled"))
		return
	}
	if err := h.ensureNoOpenHostedMatch(r.Context(), player.ID); err != nil {
		writeCreateMatchProblem(w, r, err)
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
		Mode:         mongomodel.MatchModeComputer,
		Status:       mongomodel.MatchStatusWaiting,
		HostPlayerID: player.ID,
		OpenHostLock: player.ID,
		Players: []mongomodel.MatchPlayer{
			{
				PlayerID:  player.ID,
				Nickname:  player.Nickname,
				Kind:      mongomodel.MatchPlayerKindHuman,
				Ready:     false,
				Score:     0,
				SitoneIDs: sitoneIDs,
			},
			{
				PlayerID:  computerPlayerID,
				Nickname:  computerNickname,
				Kind:      mongomodel.MatchPlayerKindComputer,
				Ready:     true,
				Score:     0,
				SitoneIDs: []string{computerDefaultSitone},
			},
		},
		CreatedAt: now,
	}

	if _, err := h.db.Collection(mongomodel.MatchesCollection).InsertOne(r.Context(), match); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			writeOpenHostedMatchConflict(w, r)
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_insert_failed", err))
		return
	}

	state, err := h.buildMatchState(r.Context(), match, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, state)
}
