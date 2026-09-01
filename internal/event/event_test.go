package event

import (
	"context"
	"strings"
	"testing"
	"time"
)

type mockRepository struct {
	events  map[string]*Event
	tickets map[string][]TicketType
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		events:  make(map[string]*Event),
		tickets: make(map[string][]TicketType),
	}
}

func (m *mockRepository) CreateEventWithTickets(ctx context.Context, event *Event, tickets []TicketType) error {
	m.events[event.ID] = event
	m.tickets[event.ID] = tickets
	return nil
}

func (m *mockRepository) GetEventByID(ctx context.Context, id string) (*Event, []TicketType, error) {
	event, exists := m.events[id]
	if !exists {
		return nil, nil, ErrEventNotFound
	}
	tickets := m.tickets[id]
	return event, tickets, nil
}

func (m *mockRepository) UpdateEventWithTickets(ctx context.Context, event *Event, tickets []TicketType) error {
	if _, exists := m.events[event.ID]; !exists {
		return ErrEventNotFound
	}
	m.events[event.ID] = event
	m.tickets[event.ID] = tickets
	return nil
}

func (m *mockRepository) ListEvents(ctx context.Context, filter EventListFilter) ([]Event, int, error) {
	var list []Event
	for _, e := range m.events {
		if filter.Search != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(filter.Search)) && !strings.Contains(strings.ToLower(e.Venue), strings.ToLower(filter.Search)) {
			continue
		}
		eCopy := *e
		eCopy.TicketTypes = m.tickets[e.ID]
		list = append(list, eCopy)
	}
	return list, len(list), nil
}

func TestEventServiceFlows(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	orgA := "usr_org_a"
	orgB := "usr_org_b"

	now := time.Now()
	saleStart := now.Add(1 * time.Hour)
	saleEnd := now.Add(24 * time.Hour)
	eventDate := now.Add(48 * time.Hour)

	// 1. Create Event Success
	createReq := CreateEventRequest{
		Name:          "Tech Conference 2026",
		Description:   "Annual Developer Summit",
		Venue:         "Dhaka Convention Center",
		EventDate:     eventDate,
		SaleStartTime: saleStart,
		SaleEndTime:   saleEnd,
		TicketTypes: []CreateTicketTypeRequest{
			{Name: "General Access", Price: 50.0, Quantity: 100},
			{Name: "VIP Pass", Price: 150.0, Quantity: 20},
		},
	}

	createdEvent, err := svc.CreateEvent(ctx, orgA, createReq)
	if err != nil {
		t.Fatalf("expected successful event creation, got: %v", err)
	}

	if createdEvent.ID == "" {
		t.Errorf("expected generated event ID")
	}
	if createdEvent.OrganizerID != orgA {
		t.Errorf("expected organizer ID %s, got %s", orgA, createdEvent.OrganizerID)
	}
	if len(createdEvent.TicketTypes) != 2 {
		t.Errorf("expected 2 ticket types, got %d", len(createdEvent.TicketTypes))
	}

	// 2. Ownership Rejection Test (Organizer B trying to update Organizer A's event)
	updateReq := UpdateEventRequest{
		Name:          "Hacked Conference Title",
		Description:   "Unauthorized update",
		Venue:         "Dhaka Convention Center",
		EventDate:     eventDate,
		SaleStartTime: saleStart,
		SaleEndTime:   saleEnd,
		TicketTypes: []CreateTicketTypeRequest{
			{Name: "General Access", Price: 10.0, Quantity: 500},
		},
	}

	_, err = svc.UpdateEvent(ctx, orgB, "organizer", createdEvent.ID, updateReq)
	if err != ErrUnauthorizedOwner {
		t.Errorf("expected ErrUnauthorizedOwner when non-owner updates event, got: %v", err)
	}

	// 3. Owner Update Success
	updatedEvent, err := svc.UpdateEvent(ctx, orgA, "organizer", createdEvent.ID, updateReq)
	if err != nil {
		t.Fatalf("expected successful update by owner, got: %v", err)
	}
	if updatedEvent.Name != "Hacked Conference Title" {
		t.Errorf("expected updated name, got %s", updatedEvent.Name)
	}

	// 4. Public List Events Test
	listResp, err := svc.ListEvents(ctx, EventListFilter{Search: "Conference"})
	if err != nil {
		t.Fatalf("expected successful event search, got: %v", err)
	}
	if listResp.Total != 1 {
		t.Errorf("expected 1 event matching 'Conference', got %d", listResp.Total)
	}
	if len(listResp.Events[0].TicketTypes) != 1 {
		t.Errorf("expected ticket types embedded in list response")
	}

	// 5. Sale Start Restriction Test (Sales already started)
	repo.events[createdEvent.ID].SaleStartTime = now.Add(-1 * time.Hour) // Past sale start

	_, err = svc.UpdateEvent(ctx, orgA, "organizer", createdEvent.ID, updateReq)
	if err != ErrSaleAlreadyStarted {
		t.Errorf("expected ErrSaleAlreadyStarted when editing active sale event, got: %v", err)
	}
}
