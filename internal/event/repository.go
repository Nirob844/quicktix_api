package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Repository defines database access interface for events and ticket types.
type Repository interface {
	CreateEventWithTickets(ctx context.Context, event *Event, tickets []TicketType) error
	GetEventByID(ctx context.Context, id string) (*Event, []TicketType, error)
	UpdateEventWithTickets(ctx context.Context, event *Event, tickets []TicketType) error
	ListEvents(ctx context.Context, filter EventListFilter) ([]Event, int, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL event repository.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateEventWithTickets(ctx context.Context, event *Event, tickets []TicketType) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Insert Event
	eventQuery := `
		INSERT INTO events (id, organizer_id, name, description, venue, event_date, sale_start_time, sale_end_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(
		ctx,
		eventQuery,
		event.ID,
		event.OrganizerID,
		event.Name,
		event.Description,
		event.Venue,
		event.EventDate,
		event.SaleStartTime,
		event.SaleEndTime,
		event.Status,
	).Scan(&event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	// 2. Insert Ticket Types
	ticketQuery := `
		INSERT INTO ticket_types (id, event_id, name, price, total_quantity, available_quantity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	for i := range tickets {
		tickets[i].EventID = event.ID
		err = tx.QueryRowContext(
			ctx,
			ticketQuery,
			tickets[i].ID,
			tickets[i].EventID,
			tickets[i].Name,
			tickets[i].Price,
			tickets[i].TotalQuantity,
			tickets[i].AvailableQuantity,
		).Scan(&tickets[i].CreatedAt, &tickets[i].UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert ticket type %s: %w", tickets[i].Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetEventByID(ctx context.Context, id string) (*Event, []TicketType, error) {
	var event Event
	eventQuery := `SELECT id, organizer_id, name, description, venue, event_date, sale_start_time, sale_end_time, status, created_at, updated_at FROM events WHERE id = $1`

	err := r.db.GetContext(ctx, &event, eventQuery, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrEventNotFound
		}
		return nil, nil, fmt.Errorf("failed to query event: %w", err)
	}

	var tickets []TicketType
	ticketQuery := `SELECT id, event_id, name, price, total_quantity, available_quantity, created_at, updated_at FROM ticket_types WHERE event_id = $1 ORDER BY price ASC`

	err = r.db.SelectContext(ctx, &tickets, ticketQuery, id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query ticket types: %w", err)
	}

	return &event, tickets, nil
}

func (r *postgresRepository) UpdateEventWithTickets(ctx context.Context, event *Event, tickets []TicketType) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Update Event Metadata
	eventQuery := `
		UPDATE events
		SET name = $1, description = $2, venue = $3, event_date = $4, sale_start_time = $5, sale_end_time = $6, updated_at = NOW()
		WHERE id = $7
	`
	res, err := tx.ExecContext(ctx, eventQuery, event.Name, event.Description, event.Venue, event.EventDate, event.SaleStartTime, event.SaleEndTime, event.ID)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrEventNotFound
	}

	// 2. Refresh Ticket Types (Delete existing & Re-insert updated categories)
	_, err = tx.ExecContext(ctx, `DELETE FROM ticket_types WHERE event_id = $1`, event.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old ticket types: %w", err)
	}

	ticketQuery := `
		INSERT INTO ticket_types (id, event_id, name, price, total_quantity, available_quantity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	for _, t := range tickets {
		_, err = tx.ExecContext(ctx, ticketQuery, t.ID, event.ID, t.Name, t.Price, t.TotalQuantity, t.AvailableQuantity)
		if err != nil {
			return fmt.Errorf("failed to insert updated ticket type %s: %w", t.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *postgresRepository) ListEvents(ctx context.Context, filter EventListFilter) ([]Event, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	if filter.Limit > 50 {
		filter.Limit = 50
	}
	offset := (filter.Page - 1) * filter.Limit

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	} else {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, "published")
		argIdx++
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(venue) LIKE $%d)", argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	whereStmt := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM events WHERE %s`, whereStmt)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, organizer_id, name, description, venue, event_date, sale_start_time, sale_end_time, status, created_at, updated_at
		FROM events
		WHERE %s
		ORDER BY event_date ASC
		LIMIT $%d OFFSET $%d
	`, whereStmt, argIdx, argIdx+1)

	listArgs := append(args, filter.Limit, offset)

	var events []Event
	if err := r.db.SelectContext(ctx, &events, listQuery, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	// Populate ticket types for each returned event
	for i := range events {
		var tickets []TicketType
		ticketQuery := `SELECT id, event_id, name, price, total_quantity, available_quantity, created_at, updated_at FROM ticket_types WHERE event_id = $1 ORDER BY price ASC`
		if err := r.db.SelectContext(ctx, &tickets, ticketQuery, events[i].ID); err == nil {
			events[i].TicketTypes = tickets
		}
	}

	return events, total, nil
}
