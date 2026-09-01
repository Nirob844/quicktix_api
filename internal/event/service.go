package event

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service defines business logic interface for event & ticket management.
type Service interface {
	CreateEvent(ctx context.Context, organizerID string, req CreateEventRequest) (*Event, error)
	GetEventByID(ctx context.Context, eventID string) (*Event, error)
	UpdateEvent(ctx context.Context, userID string, userRole string, eventID string, req UpdateEventRequest) (*Event, error)
}

type service struct {
	repo Repository
}

// NewService creates a new event service instance.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateEvent(ctx context.Context, organizerID string, req CreateEventRequest) (*Event, error) {
	if valErrs := req.Validate(); valErrs != nil {
		return nil, errorsValidation(valErrs)
	}

	eventID := uuid.New().String()
	event := &Event{
		ID:            eventID,
		OrganizerID:   organizerID,
		Name:          req.Name,
		Description:   req.Description,
		Venue:         req.Venue,
		EventDate:     req.EventDate,
		SaleStartTime: req.SaleStartTime,
		SaleEndTime:   req.SaleEndTime,
		Status:        "published",
	}

	var tickets []TicketType
	for _, t := range req.TicketTypes {
		tickets = append(tickets, TicketType{
			ID:                uuid.New().String(),
			EventID:           eventID,
			Name:              t.Name,
			Price:             t.Price,
			TotalQuantity:     t.Quantity,
			AvailableQuantity: t.Quantity,
		})
	}

	if err := s.repo.CreateEventWithTickets(ctx, event, tickets); err != nil {
		return nil, err
	}

	event.TicketTypes = tickets
	return event, nil
}

func (s *service) GetEventByID(ctx context.Context, eventID string) (*Event, error) {
	event, tickets, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	event.TicketTypes = tickets
	return event, nil
}

func (s *service) UpdateEvent(ctx context.Context, userID string, userRole string, eventID string, req UpdateEventRequest) (*Event, error) {
	if valErrs := req.Validate(); valErrs != nil {
		return nil, errorsValidation(valErrs)
	}

	// 1. Fetch existing event
	existingEvent, _, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// 2. Ownership Check (FR2.4)
	if !strings.EqualFold(userRole, "admin") && existingEvent.OrganizerID != userID {
		return nil, ErrUnauthorizedOwner
	}

	// 3. Restriction on editing after ticket sales start (FR2.4)
	if time.Now().After(existingEvent.SaleStartTime) {
		return nil, ErrSaleAlreadyStarted
	}

	// 4. Update Event Object
	existingEvent.Name = req.Name
	existingEvent.Description = req.Description
	existingEvent.Venue = req.Venue
	existingEvent.EventDate = req.EventDate
	existingEvent.SaleStartTime = req.SaleStartTime
	existingEvent.SaleEndTime = req.SaleEndTime

	var tickets []TicketType
	for _, t := range req.TicketTypes {
		tickets = append(tickets, TicketType{
			ID:                uuid.New().String(),
			EventID:           eventID,
			Name:              t.Name,
			Price:             t.Price,
			TotalQuantity:     t.Quantity,
			AvailableQuantity: t.Quantity,
		})
	}

	if err := s.repo.UpdateEventWithTickets(ctx, existingEvent, tickets); err != nil {
		return nil, err
	}

	existingEvent.TicketTypes = tickets
	return existingEvent, nil
}

type validationErr struct {
	Details map[string]string
}

func (v *validationErr) Error() string {
	return "validation failed"
}

func errorsValidation(errs map[string]string) error {
	return &validationErr{Details: errs}
}
