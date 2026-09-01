package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"quicktix/internal/platform/token"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// Service defines authentication business logic interface.
type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) (string, error)
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
}

type service struct {
	repo       Repository
	rdb        *redis.Client
	jwtSecret  string
	jwtTTL     time.Duration
	memResetMu sync.RWMutex
	memResets  map[string]string // fallback memory store for tests if redis is nil
}

// NewService creates a new auth service instance.
func NewService(repo Repository, rdb *redis.Client, jwtSecret string, jwtTTL time.Duration) Service {
	if jwtTTL == 0 {
		jwtTTL = 24 * time.Hour
	}
	return &service{
		repo:      repo,
		rdb:       rdb,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
		memResets: make(map[string]string),
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		Role:         req.Role,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	t, err := token.GenerateJWT(user.ID, user.Email, user.Role, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &AuthResponse{
		Token: t,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	t, err := token.GenerateJWT(user.ID, user.Email, user.Role, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &AuthResponse{
		Token: t,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (s *service) ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return ErrInvalidPassword
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, userID, string(newHashedPassword))
}

func (s *service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == ErrUserNotFound {
			// Return silent success or error depending on security policy; here we check user exists
			return "", ErrUserNotFound
		}
		return "", err
	}

	resetToken := uuid.New().String()
	ttl := 15 * time.Minute

	if s.rdb != nil {
		redisKey := fmt.Sprintf("password_reset:%s", user.Email)
		if err := s.rdb.Set(ctx, redisKey, resetToken, ttl).Err(); err != nil {
			return "", fmt.Errorf("failed to store reset token in redis: %w", err)
		}
	} else {
		s.memResetMu.Lock()
		s.memResets[user.Email] = resetToken
		s.memResetMu.Unlock()
	}

	slog.Info("password reset token generated", "email", user.Email, "reset_token", resetToken)
	return resetToken, nil
}

func (s *service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	var storedToken string
	if s.rdb != nil {
		redisKey := fmt.Sprintf("password_reset:%s", req.Email)
		val, err := s.rdb.Get(ctx, redisKey).Result()
		if err != nil {
			return ErrInvalidResetToken
		}
		storedToken = val
	} else {
		s.memResetMu.RLock()
		val, ok := s.memResets[req.Email]
		s.memResetMu.RUnlock()
		if !ok {
			return ErrInvalidResetToken
		}
		storedToken = val
	}

	if storedToken != req.ResetToken {
		return ErrInvalidResetToken
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.repo.UpdatePasswordByEmail(ctx, req.Email, string(newHashedPassword)); err != nil {
		return err
	}

	// Delete reset token to prevent reuse
	if s.rdb != nil {
		redisKey := fmt.Sprintf("password_reset:%s", req.Email)
		_ = s.rdb.Del(ctx, redisKey).Err()
	} else {
		s.memResetMu.Lock()
		delete(s.memResets, req.Email)
		s.memResetMu.Unlock()
	}

	slog.Info("password reset successfully executed", "email", req.Email)
	return nil
}
