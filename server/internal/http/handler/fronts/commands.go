package fronts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var frontCommandCosts = map[string]int{
	"expand":           10,
	"attack":           15,
	"reinforce":        8,
	"repair":           12,
	"scout":            4,
	"rescue":           10,
	"support":          0,
	"answer_challenge": 0,
}

var (
	errFrontRevisionConflict      = errors.New("front revision conflict")
	errDuplicateFrontCommand      = errors.New("duplicate front command")
	errFrontChangedCommandInvalid = errors.New("front changed and command is no longer valid")
)

// CreateCommand godoc
// @Summary Create front command
// @Description Accepts a front command for the authenticated player, applies it to the front snapshot, and records the command.
// @Tags fronts
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param frontID path string true "Front ID"
// @Param request body CreateCommandRequest true "Front command"
// @Success 202 {object} CreateCommandResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fronts/{frontID}/commands [post]
func (h *Handler) CreateCommand(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	front, err := h.frontByID(r.Context(), chi.URLParam(r, "frontID"))
	if errors.Is(err, errFrontNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("front not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front unavailable", "front_lookup_failed", err))
		return
	}
	front = withCurrentPlayerTeam(front, player)
	if front.MapMode == contentFrontMapModeTerritoryGrid && !isParticipatingTerritoryTeam(player.TeamID) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "this team can only observe the territory front"))
		return
	}

	var body CreateCommandRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body = normalizeCommandRequest(body)
	if err := validateCommandRequest(body, front.MapMode); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	if body.ClientCommandID != "" {
		existing, found, err := h.findExistingCommand(r.Context(), front.ID, player.ID, body.ClientCommandID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_command_lookup_failed", err))
			return
		}
		if found {
			h.writeCommandResponse(w, r, http.StatusAccepted, front, player, existing)
			return
		}
	}
	if body.ExpectedRevision != nil && *body.ExpectedRevision != front.Revision {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "front revision is stale"))
		return
	}

	command := mongomodel.FrontCommand{
		ID:               newID("front_command"),
		ClientCommandID:  body.ClientCommandID,
		FrontID:          front.ID,
		PlayerID:         player.ID,
		TeamID:           player.TeamID,
		Kind:             body.Kind,
		Type:             body.Kind,
		FromCellID:       body.FromCellID,
		ToCellID:         body.ToCellID,
		TargetX:          body.TargetX,
		TargetY:          body.TargetY,
		ExpectedRevision: body.ExpectedRevision,
		SitoneID:         body.SitoneID,
		Payload:          clonePayload(body.Payload),
		CreatedAt:        time.Now().UTC(),
	}
	previousRevision := front.Revision
	front, command, err = applyCommandToFront(front, command)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(err.Error()))
		return
	}
	front, command, err = h.persistCommandWithRecompute(
		r.Context(), previousRevision, front, command, body.ExpectedRevision != nil,
	)
	if errors.Is(err, errFrontRevisionConflict) || errors.Is(err, errFrontChangedCommandInvalid) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "front changed while applying the command"))
		return
	} else if errors.Is(err, errDuplicateFrontCommand) {
		existing, found, lookupErr := h.findExistingCommand(r.Context(), front.ID, player.ID, command.ClientCommandID)
		if lookupErr != nil || !found {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "front command was already submitted"))
			return
		}
		latest, lookupErr := h.frontByID(r.Context(), front.ID)
		if lookupErr != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("front unavailable", "front_lookup_failed", lookupErr))
			return
		}
		h.writeCommandResponse(w, r, http.StatusAccepted, latest, player, existing)
		return
	} else if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_command_failed", err))
		return
	}
	h.writeCommandResponse(w, r, http.StatusAccepted, front, player, command)
}

func (h *Handler) persistCommandWithRecompute(ctx context.Context, previousRevision int64, front mongomodel.Front, command mongomodel.FrontCommand, allowRecompute bool) (mongomodel.Front, mongomodel.FrontCommand, error) {
	err := h.recordCommand(ctx, previousRevision, front, command)
	for attempt := 0; allowRecompute && errors.Is(err, errFrontRevisionConflict) && attempt < 3; attempt++ {
		latest, lookupErr := h.frontByID(ctx, front.ID)
		if lookupErr != nil {
			return front, command, lookupErr
		}
		previousRevision = latest.Revision
		retryCommand := command
		retryCommand.Accepted = false
		retryCommand.Applied = false
		retryCommand.RejectReason = ""
		retryCommand.AffectedCells = nil
		next, applied, applyErr := applyCommandToFront(latest, retryCommand)
		if applyErr != nil {
			return latest, command, fmt.Errorf("%w: %v", errFrontChangedCommandInvalid, applyErr)
		}
		front, command = next, applied
		err = h.recordCommand(ctx, previousRevision, front, command)
	}
	return front, command, err
}

func (h *Handler) writeCommandResponse(w http.ResponseWriter, r *http.Request, status int, front mongomodel.Front, player mongomodel.Player, command mongomodel.FrontCommand) {
	sitones, err := h.playerFrontSitones(r.Context(), player)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front sitones unavailable", "front_sitones_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, status, CreateCommandResponse{
		Accepted: true,
		Command:  commandResponse(command),
		Front:    detailResponse(front, player.TeamID, sitones),
	})
}

func (h *Handler) findExistingCommand(ctx context.Context, frontID string, playerID string, clientCommandID string) (mongomodel.FrontCommand, bool, error) {
	var command mongomodel.FrontCommand
	err := h.db.Collection(mongomodel.FrontCommandsCollection).FindOne(ctx, bson.M{
		"front_id": frontID, "player_id": playerID, "client_command_id": clientCommandID,
	}).Decode(&command)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return mongomodel.FrontCommand{}, false, nil
	}
	return command, err == nil, err
}

func (h *Handler) recordCommand(ctx context.Context, previousRevision int64, front mongomodel.Front, command mongomodel.FrontCommand) error {
	session, err := h.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		if _, err := h.db.Collection(mongomodel.FrontCommandsCollection).InsertOne(ctx, command); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, errDuplicateFrontCommand
			}
			return nil, err
		}
		if front.MapMode == contentFrontMapModeTerritoryGrid {
			if _, err := h.db.Collection(mongomodel.FrontsCollection).UpdateMany(
				ctx,
				bson.M{"_id": bson.M{"$ne": front.ID}, "current": true},
				bson.M{"$set": bson.M{"current": false, "updated_at": front.UpdatedAt}},
			); err != nil {
				return nil, err
			}
		}
		if err := h.updateFrontAfterCommand(ctx, previousRevision, front, command); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (h *Handler) updateFrontAfterCommand(ctx context.Context, previousRevision int64, front mongomodel.Front, command mongomodel.FrontCommand) error {
	summary := commandSummary(command)
	result, err := h.db.Collection(mongomodel.FrontsCollection).UpdateOne(
		ctx,
		frontRevisionFilter(front.ID, previousRevision),
		bson.M{
			"$set": bson.M{
				"map_id":        front.MapID,
				"map_mode":      front.MapMode,
				"name":          front.Name,
				"status":        front.Status,
				"current":       front.Current,
				"tick":          front.Tick,
				"revision":      front.Revision,
				"cells":         front.Cells,
				"teams":         front.Teams,
				"zones":         front.Zones,
				"active_events": front.ActiveEvents,
				"leaderboard":   front.Leaderboard,
				"last_command":  summary,
				"territory":     front.Territory,
				"updated_at":    front.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"_id":        front.ID,
				"created_at": front.CreatedAt,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errFrontRevisionConflict
		}
		return err
	}
	if result.MatchedCount == 0 && result.UpsertedCount == 0 {
		return errFrontRevisionConflict
	}
	return nil
}

func frontRevisionFilter(frontID string, previousRevision int64) bson.M {
	filter := bson.M{"_id": frontID, "revision": previousRevision}
	if previousRevision == 1 {
		filter = bson.M{
			"_id": frontID,
			"$or": bson.A{
				bson.M{"revision": previousRevision},
				bson.M{"revision": bson.M{"$exists": false}},
			},
		}
	}
	return filter
}

func normalizeCommandRequest(body CreateCommandRequest) CreateCommandRequest {
	body.ClientCommandID = strings.TrimSpace(body.ClientCommandID)
	body.Kind = strings.TrimSpace(body.Kind)
	body.Type = strings.TrimSpace(body.Type)
	if body.Kind == "" {
		body.Kind = body.Type
	}
	body.FromCellID = strings.TrimSpace(body.FromCellID)
	body.ToCellID = strings.TrimSpace(body.ToCellID)
	body.SitoneID = strings.TrimSpace(body.SitoneID)
	if body.Payload != nil {
		if body.FromCellID == "" {
			body.FromCellID = payloadString(body.Payload, "fromCellId", "from_cell_id", "from")
		}
		if body.ToCellID == "" {
			body.ToCellID = payloadString(body.Payload, "toCellId", "to_cell_id", "targetCellId", "cellId", "to")
		}
		if body.SitoneID == "" {
			body.SitoneID = payloadString(body.Payload, "sitoneId", "sitone_id")
		}
		if body.TargetX == nil {
			body.TargetX = payloadInt(body.Payload, "targetX", "target_x", "x")
		}
		if body.TargetY == nil {
			body.TargetY = payloadInt(body.Payload, "targetY", "target_y", "y")
		}
		if body.ExpectedRevision == nil {
			body.ExpectedRevision = payloadInt64(body.Payload, "expectedRevision", "expected_revision", "revision")
		}
	}
	return body
}

func payloadInt(payload map[string]any, keys ...string) *int {
	value := payloadInt64(payload, keys...)
	if value == nil {
		return nil
	}
	converted := int(*value)
	if int64(converted) != *value {
		return nil
	}
	return &converted
}

func payloadInt64(payload map[string]any, keys ...string) *int64 {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		var converted int64
		switch number := value.(type) {
		case int:
			converted = int64(number)
		case int64:
			converted = number
		case float64:
			converted = int64(number)
			if float64(converted) != number {
				continue
			}
		default:
			continue
		}
		return &converted
	}
	return nil
}

func payloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func validateCommandRequest(body CreateCommandRequest, mapMode string) error {
	var details []httpx.ErrorDetail
	if body.Kind == "" {
		details = append(details, httpx.ErrorDetail{Location: "body.kind", Message: "kind is required"})
	} else if _, ok := frontCommandCosts[body.Kind]; !ok {
		details = append(details, httpx.ErrorDetail{Location: "body.kind", Message: "kind is unsupported"})
	}
	if mapMode == contentFrontMapModeTerritoryGrid {
		if !territoryCommandIsSupported(body.Kind) {
			details = append(details, httpx.ErrorDetail{Location: "body.kind", Message: "kind is unsupported on a territory map"})
		}
		if body.ClientCommandID == "" {
			details = append(details, httpx.ErrorDetail{Location: "body.clientCommandId", Message: "clientCommandId is required"})
		}
		if body.TargetX == nil || *body.TargetX < 0 || *body.TargetX >= 64 {
			details = append(details, httpx.ErrorDetail{Location: "body.targetX", Message: "targetX must be between 0 and 63"})
		}
		if body.TargetY == nil || *body.TargetY < 0 || *body.TargetY >= 57 {
			details = append(details, httpx.ErrorDetail{Location: "body.targetY", Message: "targetY must be between 0 and 56"})
		}
		if body.ExpectedRevision != nil && *body.ExpectedRevision < 0 {
			details = append(details, httpx.ErrorDetail{Location: "body.expectedRevision", Message: "expectedRevision must be non-negative"})
		}
	} else {
		if body.FromCellID == "" {
			details = append(details, httpx.ErrorDetail{Location: "body.fromCellId", Message: "fromCellId is required"})
		}
		if body.ToCellID == "" {
			details = append(details, httpx.ErrorDetail{Location: "body.toCellId", Message: "toCellId is required"})
		}
		if body.SitoneID == "" {
			details = append(details, httpx.ErrorDetail{Location: "body.sitoneId", Message: "sitoneId is required"})
		}
	}
	if len(details) > 0 {
		return httpx.UnprocessableEntity("invalid front command", details...)
	}
	return nil
}

func applyCommandToFront(front mongomodel.Front, command mongomodel.FrontCommand) (mongomodel.Front, mongomodel.FrontCommand, error) {
	if front.MapMode == contentFrontMapModeTerritoryGrid {
		return applyTerritoryCommand(front, command)
	}
	if !isPlayableFrontStatus(front.Status) {
		return front, command, errors.New("front is not accepting commands")
	}
	teamIndex := frontTeamIndex(front.Teams, command.TeamID)
	if teamIndex < 0 {
		return front, command, errors.New("team is not part of this front")
	}
	fromIndex := frontCellIndex(front.Cells, command.FromCellID)
	if fromIndex < 0 {
		return front, command, errors.New("from cell does not exist")
	}
	toIndex := frontCellIndex(front.Cells, command.ToCellID)
	if toIndex < 0 {
		return front, command, errors.New("target cell does not exist")
	}
	fromCell := front.Cells[fromIndex]
	if fromCell.OwnerTeamID != command.TeamID {
		return front, command, errors.New("from cell is not controlled by your team")
	}
	if !frontCellsAreNeighbors(fromCell, command.ToCellID) {
		return front, command, errors.New("target cell is not adjacent to from cell")
	}

	cost := frontCommandCosts[command.Kind]
	if command.Kind != "support" && front.Teams[teamIndex].FrontOpenPower < cost {
		return front, command, errors.New("not enough front open power")
	}
	if err := validateCommandTarget(front, command, front.Cells[toIndex]); err != nil {
		return front, command, err
	}

	if command.Kind != "support" {
		front.Teams[teamIndex].FrontOpenPower -= cost
	}
	front.Teams[teamIndex].LastCommandAt = command.CreatedAt
	scoreDelta, resourceDelta := applyCommandEffect(&front, teamIndex, toIndex, command)
	front.Teams[teamIndex].Score += scoreDelta
	front.Teams[teamIndex].FrontOpenPower += resourceDelta
	front.Revision++
	front.Tick++
	front.UpdatedAt = command.CreatedAt
	command.Accepted = true
	command.Applied = true
	summary := commandSummary(command)
	front.LastCommand = &summary
	syncTeamRanks(front.Teams, front.Cells)
	front.Leaderboard = deriveLeaderboard(front)
	return front, command, nil
}

func validateCommandTarget(front mongomodel.Front, command mongomodel.FrontCommand, target mongomodel.FrontCell) error {
	switch command.Kind {
	case "expand":
		if target.OwnerTeamID != "" {
			return errors.New("expand target must be neutral")
		}
	case "attack":
		if target.OwnerTeamID == "" || target.OwnerTeamID == command.TeamID {
			return errors.New("attack target must be controlled by another team")
		}
	case "reinforce":
		if target.OwnerTeamID != command.TeamID {
			return errors.New("reinforce target must be controlled by your team")
		}
		if target.Control >= 100 && target.Defense >= territoryMaxDefense {
			return errors.New("reinforce target is already at maximum defense")
		}
	case "repair":
		if activeEventKindAtFrontCell(front, target.ID) != "repair" {
			return errors.New("target has no active repair event")
		}
	case "rescue":
		if activeEventKindAtFrontCell(front, target.ID) != "rescue" {
			return errors.New("target has no active rescue event")
		}
	}
	return nil
}

func applyCommandEffect(front *mongomodel.Front, teamIndex int, cellIndex int, command mongomodel.FrontCommand) (int, int) {
	cell := &front.Cells[cellIndex]
	switch command.Kind {
	case "expand":
		addFrontPressure(cell, command.TeamID, 55)
		if frontPressureFor(cell, command.TeamID) >= 50 {
			captureFrontCell(cell, command.TeamID, 55, 8)
			return 20, collectFrontCellResource(cell)
		}
		return 4, 0
	case "attack":
		addFrontPressure(cell, command.TeamID, 45)
		cell.Control = clampFrontInt(cell.Control-35, 0, 100)
		cell.Defense = maxFrontInt(0, cell.Defense-10)
		if cell.Control == 0 || frontPressureFor(cell, command.TeamID) >= 60 {
			captureFrontCell(cell, command.TeamID, 45, 6)
			return 25, collectFrontCellResource(cell)
		}
		return 8, 0
	case "reinforce":
		cell.Control = clampFrontInt(cell.Control+25, 0, 100)
		cell.Defense = clampFrontInt(cell.Defense+12, 0, territoryMaxDefense)
		return 5, 0
	case "repair":
		removeActiveFrontEventAtCell(front, cell.ID)
		cell.EventID = ""
		cell.Control = clampFrontInt(cell.Control+20, 0, 100)
		cell.Defense = clampFrontInt(cell.Defense+10, 0, territoryMaxDefense)
		front.Teams[teamIndex].RepairedEvents++
		return 30, 0
	case "rescue":
		removeActiveFrontEventAtCell(front, cell.ID)
		cell.EventID = ""
		front.Teams[teamIndex].RescuedSitones++
		return 30, 0
	case "scout":
		addFrontPressure(cell, command.TeamID, 8)
		front.Teams[teamIndex].CollaborationScore++
		return 3, 0
	case "support", "answer_challenge":
		addFrontPressure(cell, command.TeamID, 15)
		front.Teams[teamIndex].CollaborationScore += 3
		return 3, 0
	default:
		return 0, 0
	}
}

func isPlayableFrontStatus(status string) bool {
	return status == "" || status == mongomodel.FrontStatusOpenPlay || status == "surge" || status == "booth_window"
}

func frontCellIndex(cells []mongomodel.FrontCell, cellID string) int {
	for i, cell := range cells {
		if cell.ID == cellID {
			return i
		}
	}
	return -1
}

func frontCellsAreNeighbors(cell mongomodel.FrontCell, targetCellID string) bool {
	for _, neighborID := range cell.NeighborIDs {
		if neighborID == targetCellID {
			return true
		}
	}
	return false
}

func activeEventKindAtFrontCell(front mongomodel.Front, cellID string) string {
	for _, event := range front.ActiveEvents {
		if event.CellID == cellID {
			return event.Kind
		}
	}
	return ""
}

func removeActiveFrontEventAtCell(front *mongomodel.Front, cellID string) {
	active := front.ActiveEvents[:0]
	for _, event := range front.ActiveEvents {
		if event.CellID == cellID {
			continue
		}
		active = append(active, event)
	}
	front.ActiveEvents = active
}

func addFrontPressure(cell *mongomodel.FrontCell, teamID string, amount int) {
	if cell.PressureByTeam == nil {
		cell.PressureByTeam = make(map[string]int)
	}
	cell.PressureByTeam[teamID] += amount
}

func frontPressureFor(cell *mongomodel.FrontCell, teamID string) int {
	if cell.PressureByTeam == nil {
		return 0
	}
	return cell.PressureByTeam[teamID]
}

func captureFrontCell(cell *mongomodel.FrontCell, teamID string, control int, defense int) {
	cell.OwnerTeamID = teamID
	cell.Control = clampFrontInt(control, 1, 100)
	cell.Defense = defense
	cell.PressureByTeam = map[string]int{}
}

func collectFrontCellResource(cell *mongomodel.FrontCell) int {
	resource := cell.Resource
	cell.Resource = 0
	return resource
}

func clampFrontInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxFrontInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func commandSummary(command mongomodel.FrontCommand) mongomodel.FrontCommandSummary {
	return mongomodel.FrontCommandSummary{
		ID:               command.ID,
		ClientCommandID:  command.ClientCommandID,
		Kind:             command.Kind,
		Type:             command.Type,
		PlayerID:         command.PlayerID,
		TeamID:           command.TeamID,
		FromCellID:       command.FromCellID,
		ToCellID:         command.ToCellID,
		TargetX:          command.TargetX,
		TargetY:          command.TargetY,
		ExpectedRevision: command.ExpectedRevision,
		AffectedCells:    append([]mongomodel.FrontCoordinate(nil), command.AffectedCells...),
		SitoneID:         command.SitoneID,
		Payload:          clonePayload(command.Payload),
		Accepted:         command.Accepted,
		Applied:          command.Applied,
		RejectReason:     command.RejectReason,
		CreatedAt:        command.CreatedAt,
	}
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
