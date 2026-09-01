package event

import (
	"errors"
	"time"

	"quicktix/internal/platform/validator"
)

var (
	ErrEventNotFound       = errors.New("event not found")
	ErrUnauthorizedOwner   = errors.New("you are not authorized to manage this event")
	ErrSaleAlreadyStarted  = errors.New("event editing is restricted after ticket sales have started")
	ErrInvalidTimeWindow   = errors.New("sale start time must be before sale end time")
	ErrInvalidEventDate    = errors.New("event date must be after sale end time")
	ErrTicketTypesRequired = errors.New("at least one ticket type is required")
)

// Event represents the event domain model in PostgreSQL.
type Event struct {
	ID            string       `db:"id" json:"id"`
	OrganizerID   string       `db:"organizer_id" json:"organizer_id"`
	Name          string       `db:"name" json:"name"`
	Description   string       `db:"description" json:"description"`
	Venue         string       `db:"venue" json:"venue"`
	EventDate     time.Time    `db:"event_date" json:"event_date"`
	SaleStartTime time.Time    `db:"sale_start_time" json:"sale_start_time"`
	SaleEndTime   time.Time    `db:"sale_end_time" json:"sale_end_time"`
	Status        string       `db:"status" json:"status"`
	CreatedAt     time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at" json:"updated_at"`
	TicketTypes   []TicketType `json:"ticket_types,omitempty"`
}

// TicketType represents a ticket category with pricing and inventory.
type TicketType struct {
	ID                string    `db:"id" json:"id"`
	EventID           string    `db:"event_id" json:"event_id"`
	Name              string    `db:"name" json:"name"`
	Price             float64   `db:"price" json:"price"`
	TotalQuantity     int       `db:"total_quantity" json:"total_quantity"`
	AvailableQuantity int       `db:"available_quantity" json:"available_quantity"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

// EventListFilter holds parameters for searching and paginating events.
type EventListFilter struct {
	Search string `json:"search"`
	Status string `json:"status"`
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
}

// PaginatedEventsResponse represents paginated event list response.
type PaginatedEventsResponse struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
}

// CreateTicketTypeRequest holds payload for ticket type creation.
type CreateTicketTypeRequest struct {
	Name     string  `json:"name" validate:"required,min=2"`
	Price    float64 `json:"price" validate:"gte=0"`
	Quantity int     `json:"quantity" validate:"required,gt=0"`
}

// CreateEventRequest holds payload for event creation.
type CreateEventRequest struct {
	Name          string                    `json:"name" validate:"required,min=3"`
	Description   string                    `json:"description"`
	Venue         string                    `json:"venue" validate:"required"`
	EventDate     time.Time                 `json:"event_date" validate:"required"`
	SaleStartTime time.Time                 `json:"sale_start_time" validate:"required"`
	SaleEndTime   time.Time                 `json:"sale_end_time" validate:"required"`
	TicketTypes   []CreateTicketTypeRequest `json:"ticket_types" validate:"required,min=1,dive"`
}

// Validate validates CreateEventRequest.
func (r *CreateEventRequest) Validate() map[string]string {
	errs := validator.Validate(r)
	if errs == nil {
		errs = make(map[string]string)
	}

	if !r.SaleStartTime.Before(r.SaleEndTime) {
		errs["sale_start_time"] = "sale start time must be before sale end time"
	}
	if !r.EventDate.After(r.SaleEndTime) {
		errs["event_date"] = "event date must be after sale end time"
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// UpdateEventRequest holds payload for updating an event.
type UpdateEventRequest struct {
	Name          string                    `json:"name" validate:"required,min=3"`
	Description   string                    `json:"description"`
	Venue         string                    `json:"venue" validate:"required"`
	EventDate     time.Time                 `json:"event_date" validate:"required"`
	SaleStartTime time.Time                 `json:"sale_start_time" validate:"required"`
	SaleEndTime   time.Time                 `json:"sale_end_time" validate:"required"`
	TicketTypes   []CreateTicketTypeRequest `json:"ticket_types" validate:"required,min=1,dive"`
}

// Validate validates UpdateEventRequest.
func (r *UpdateEventRequest) Validate() map[string]string {
	errs := validator.Validate(r)
	if errs == nil {
		errs = make(map[string]string)
	}

	if !r.SaleStartTime.Before(r.SaleEndTime) {
		errs["sale_start_time"] = "sale start time must be before sale end time"
	}
	if !r.EventDate.After(r.SaleEndTime) {
		errs["event_date"] = "event date must be after sale end time"
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// EventResponse represents safe public event payload.
type EventResponse struct {
	Event Event `json:"event"`
}
