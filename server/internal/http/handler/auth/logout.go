package auth

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// Logout godoc
// @Summary User logout
// @Description Clears the camp2026_auth cookie for the current browser session.
// @Tags User Authentication
// @Produce json
// @Security AuthCookieAuth
// @Success 204
// @Header 204 {string} Set-Cookie "camp2026_auth=; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=0"
// @Failure 401 {object} httpx.ProblemDetails
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	player, ok := authctx.PlayerFromContext(r.Context())
	if !ok || player.ID == "" || player.AuthToken == "" {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnauthorized, "authentication required"))
		return
	}

	http.SetCookie(w, authCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}
