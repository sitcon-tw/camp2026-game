package me

import (
	"context"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// ListSitones godoc
// @Summary List current player sitones
// @Description Returns sitones owned by the authenticated player with catalog definitions.
// @Tags me
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} SitoneListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/sitones [get]
func (h *Handler) ListSitones(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}
	if h.content == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("content store is unavailable"))
		return
	}

	records, err := h.findPlayerSitones(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("sitones unavailable", "me_sitones_lookup_failed", err))
		return
	}

	sitones, err := mapPlayerSitones(h.content, records)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("sitone inventory is inconsistent", "me_sitones_response_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, SitoneListResponse{
		Sitones: sitones,
	})
}

func (h *Handler) findPlayerSitones(ctx context.Context, playerID string) ([]mongomodel.PlayerSitone, error) {
	cursor, err := h.db.Collection(mongomodel.PlayerSitonesCollection).Find(
		ctx,
		bson.M{"player_id": playerID, "quantity": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "sitone_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.PlayerSitone
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func mapPlayerSitones(store *content.Store, records []mongomodel.PlayerSitone) ([]PlayerSitoneResponse, error) {
	out := make([]PlayerSitoneResponse, 0, len(records))
	for _, record := range records {
		sitone, ok := store.GetSitone(record.SitoneID)
		if !ok {
			continue
		}
		out = append(out, PlayerSitoneResponse{
			ID:       record.ID,
			SitoneID: record.SitoneID,
			Quantity: record.Quantity,
			Sitone:   sitoneResponse(sitone),
		})
	}
	return out, nil
}

func sitoneResponse(sitone content.Sitone) SitoneResponse {
	return SitoneResponse{
		ID:                 sitone.ID,
		Name:               sitone.Name,
		Type:               sitone.Type,
		Rarity:             sitone.Rarity,
		Style:              sitone.Style,
		Description:        sitone.Description,
		IconPath:           sitone.IconPath,
		AbilityName:        sitone.AbilityName,
		AbilityKind:        sitone.AbilityKind,
		AbilityValue:       sitone.AbilityValue,
		AbilityCount:       sitone.AbilityCount,
		AbilityDescription: sitone.AbilityDescription,
	}
}
