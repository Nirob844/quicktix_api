package auth

import (
	"errors"
	"strings"
	"time"

	"quicktix/internal/platform/validator"
)

// Standard RBAC Role Constants
const (
	RoleBuyer     = "buyer"
	RoleOrganizer = "organizer"
	RoleAdmin     = "admin"
)

var (
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid email or password")
	ErrInvalidRole       = errors.New("role must be 'buyer', 'organizer', or 'admin'")
	ErrInvalidResetToken = errors.New("invalid or expired password reset token")
	ErrSamePassword      = errors.New("new password cannot be identical to old password")
)

// User represents the user domain model in PostgreSQL.
type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	FullName     string    `db:"full_name" json:"full_name"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// RegisterRequest holds DTO payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	FullName string `json:"full_name" validate:"required"`
	Role     string `json:"role" validate:"omitempty,oneof=buyer organizer admin"`
}

// Validate validates registration inputs using the platform validator.
func (r *RegisterRequest) Validate() map[string]string {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Role == "" {
		r.Role = RoleBuyer
	}
	r.Role = strings.ToLower(r.Role)
	return validator.Validate(r)
}

// LoginRequest holds DTO payload for user login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Validate validates login inputs.
func (r *LoginRequest) Validate() map[string]string {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	return validator.Validate(r)
}

// ChangePasswordRequest holds DTO payload for changing password when authenticated.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,nefield=OldPassword"`
}

// Validate validates change password inputs.
func (r *ChangePasswordRequest) Validate() map[string]string {
	return validator.Validate(r)
}

// ForgotPasswordRequest holds DTO payload for requesting a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// Validate validates forgot password inputs.
func (r *ForgotPasswordRequest) Validate() map[string]string {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	return validator.Validate(r)
}

// ResetPasswordRequest holds DTO payload for executing a password reset with token.
type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	ResetToken  string `json:"reset_token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// Validate validates reset password inputs.
func (r *ResetPasswordRequest) Validate() map[string]string {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.ResetToken = strings.TrimSpace(r.ResetToken)
	return validator.Validate(r)
}

// UserResponse represents safe public user payload.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse represents the JSON response for successful authentication.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
