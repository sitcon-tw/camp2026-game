package catalog

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

type SitoneListResponse struct {
	Sitones []SitoneResponse `json:"sitones"`
}

type SitoneResponse struct {
	ID                 string `json:"id" example:"stone_engineering_base"`
	Name               string `json:"name" example:"工程型小石"`
	Type               string `json:"type" example:"engineering"`
	Rarity             string `json:"rarity" example:"base"`
	Style              string `json:"style" example:"default"`
	Description        string `json:"description" example:"修 bug、分享解法、完成技術任務。"`
	IconPath           string `json:"iconPath,omitempty" example:"/game-icons/stones/basic_blue.png"`
	AbilityName        string `json:"abilityName" example:"穩定輸出"`
	AbilityKind        string `json:"abilityKind" example:"answer_score_bonus"`
	AbilityValue       int    `json:"abilityValue" example:"5"`
	AbilityCount       int    `json:"abilityCount" example:"0"`
	AbilityDescription string `json:"abilityDescription" example:"答對時分數提高 5%。"`
	Attack             int    `json:"attack" example:"3"`
	Defense            int    `json:"defense" example:"6"`
	Repeatable         bool   `json:"repeatable" example:"true"`
	Unique             bool   `json:"unique" example:"false"`
	EffectKind         string `json:"effectKind,omitempty" example:"fortify"`
	EffectValue        int    `json:"effectValue,omitempty" example:"8"`
}

// ListSitones godoc
// @Summary List sitone catalog
// @Description Lists all collectible sitone definitions.
// @Tags catalog
// @Produce json
// @Success 200 {object} SitoneListResponse
// @Failure 503 {object} httpx.ProblemDetails
// @Router /catalog/sitones [get]
func (h *Handler) ListSitones(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("content store is unavailable"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, SitoneListResponse{
		Sitones: mapSitones(h.content.ListSitones()),
	})
}

func mapSitones(sitones []content.Sitone) []SitoneResponse {
	if len(sitones) == 0 {
		return nil
	}

	out := make([]SitoneResponse, 0, len(sitones))
	for _, sitone := range sitones {
		out = append(out, SitoneResponse{
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
			Attack:             sitone.Attack,
			Defense:            sitone.Defense,
			Repeatable:         sitone.Repeatable,
			Unique:             sitone.Unique,
			EffectKind:         sitone.EffectKind,
			EffectValue:        sitone.EffectValue,
		})
	}
	return out
}
