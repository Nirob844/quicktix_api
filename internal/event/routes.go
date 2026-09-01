package event

import (
	"net/http"

	"quicktix/internal/platform/middleware"
	"quicktix/internal/platform/router"
)

// RegisterRoutes registers all event domain endpoints to the application router.
func (h *Handler) RegisterRoutes(r *router.Router, jwtSecret string) {
	// Public event endpoints
	r.Handle("GET /api/v1/events", http.HandlerFunc(h.List))
	r.Handle("GET /api/v1/events/{id}", http.HandlerFunc(h.Get))

	// Protected Organizer endpoints
	r.Handle("POST /api/v1/events", middleware.Authenticate(jwtSecret)(middleware.RequireOrganizer()(http.HandlerFunc(h.Create))))
	r.Handle("PUT /api/v1/events/{id}", middleware.Authenticate(jwtSecret)(middleware.RequireOrganizer()(http.HandlerFunc(h.Update))))
}
