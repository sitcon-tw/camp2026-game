package qrcode

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// GetMyQRCode godoc
// @Summary Get my QRCode
// @Description Returns the current player's QRCode token metadata for identity and match pairing.
// @Tags QRCode
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.QRCodeResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/qrcode [get]
func (h *Handler) GetMyQRCode(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.QRCodeResponse{
		Token:    "player_qr_token",
		ImageURL: "https://example.test/qrcode/player_qr_token.png",
		Purposes: []string{"staff_verification", "match_pairing"},
	})
}
