package event

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"quicktix/internal/platform/middleware"
	"quicktix/internal/platform/response"
)

// Handler handles HTTP requests for event management.
type Handler struct {
	service Service
}

// NewHandler creates a new event HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /api/v1/events requests.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	event, err := h.service.CreateEvent(r.Context(), claims.UserID, req)
	if err != nil {
		var vErr *validationErr
		if errors.As(err, &vErr) {
			response.ValidationError(w, vErr.Details)
			return
		}
		response.InternalServerError(w, "failed to create event: "+err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, EventResponse{Event: *event})
}

// Get handles GET /api/v1/events/{id} requests.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "event id is required")
		return
	}

	event, err := h.service.GetEventByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrEventNotFound) {
			response.NotFound(w, "event not found")
			return
		}
		response.InternalServerError(w, "failed to query event")
		return
	}

	response.JSON(w, http.StatusOK, EventResponse{Event: *event})
}

// List handles GET /api/v1/events public search and pagination requests.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	filter := EventListFilter{
		Search: query.Get("search"),
		Status: query.Get("status"),
		Page:   page,
		Limit:  limit,
	}

	resp, err := h.service.ListEvents(r.Context(), filter)
	if err != nil {
		response.InternalServerError(w, "failed to list events")
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// Update handles PUT /api/v1/events/{id} requests.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "event id is required")
		return
	}

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	event, err := h.service.UpdateEvent(r.Context(), claims.UserID, claims.Role, id, req)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedOwner) {
			response.Forbidden(w, "you are not authorized to update this event")
			return
		}
		if errors.Is(err, ErrSaleAlreadyStarted) {
			response.BadRequest(w, err.Error())
			return
		}
		if errors.Is(err, ErrEventNotFound) {
			response.NotFound(w, "event not found")
			return
		}
		var vErr *validationErr
		if errors.As(err, &vErr) {
			response.ValidationError(w, vErr.Details)
			return
		}

		response.InternalServerError(w, "failed to update event")
		return
	}

	response.JSON(w, http.StatusOK, EventResponse{Event: *event})
}
