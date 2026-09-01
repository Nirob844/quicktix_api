package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository interface defines database operations for user authentication.
type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdatePassword(ctx context.Context, userID string, newPasswordHash string) error
	UpdatePasswordByEmail(ctx context.Context, email string, newPasswordHash string) error
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL user repository.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // unique_violation for email
				return ErrUserAlreadyExists
			}
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	query := `SELECT id, email, password_hash, full_name, role, created_at, updated_at FROM users WHERE email = $1`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	return &user, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	query := `SELECT id, email, password_hash, full_name, role, created_at, updated_at FROM users WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	return &user, nil
}

func (r *postgresRepository) UpdatePassword(ctx context.Context, userID string, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *postgresRepository) UpdatePasswordByEmail(ctx context.Context, email string, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE email = $2`
	res, err := r.db.ExecContext(ctx, query, newPasswordHash, email)
	if err != nil {
		return fmt.Errorf("failed to update user password by email: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
