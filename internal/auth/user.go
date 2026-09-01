package auth

import (
	"errors"
	"strings"
	"time"
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
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

// Validate validates registration inputs.
func (r *RegisterRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	if len(r.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if strings.TrimSpace(r.FullName) == "" {
		return errors.New("full name is required")
	}
	if r.Role == "" {
		r.Role = RoleBuyer
	}
	r.Role = strings.ToLower(r.Role)
	if r.Role != RoleBuyer && r.Role != RoleOrganizer && r.Role != RoleAdmin {
		return ErrInvalidRole
	}
	return nil
}

// LoginRequest holds DTO payload for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates login inputs.
func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// ChangePasswordRequest holds DTO payload for changing password when authenticated.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Validate validates change password inputs.
func (r *ChangePasswordRequest) Validate() error {
	if r.OldPassword == "" {
		return errors.New("old password is required")
	}
	if len(r.NewPassword) < 6 {
		return errors.New("new password must be at least 6 characters")
	}
	if r.OldPassword == r.NewPassword {
		return ErrSamePassword
	}
	return nil
}

// ForgotPasswordRequest holds DTO payload for requesting a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// Validate validates forgot password inputs.
func (r *ForgotPasswordRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// ResetPasswordRequest holds DTO payload for executing a password reset with token.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
}

// Validate validates reset password inputs.
func (r *ResetPasswordRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	r.ResetToken = strings.TrimSpace(r.ResetToken)
	if r.ResetToken == "" {
		return errors.New("reset token is required")
	}
	if len(r.NewPassword) < 6 {
		return errors.New("new password must be at least 6 characters")
	}
	return nil
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
