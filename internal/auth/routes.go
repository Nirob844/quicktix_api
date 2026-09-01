package auth

import (
	"net/http"

	"quicktix/internal/platform/middleware"
	"quicktix/internal/platform/router"
)

// RegisterRoutes registers all auth domain endpoints to the application router.
func (h *Handler) RegisterRoutes(r *router.Router, jwtSecret string) {
	r.Handle("POST /api/v1/auth/register", http.HandlerFunc(h.Register))
	r.Handle("POST /api/v1/auth/login", http.HandlerFunc(h.Login))
	r.Handle("GET /api/v1/auth/me", middleware.Authenticate(jwtSecret)(http.HandlerFunc(h.Me)))
}
