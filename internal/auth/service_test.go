package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type mockRepo struct {
	users map[string]*User
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]*User)}
}

func (m *mockRepo) CreateUser(ctx context.Context, user *User) error {
	if _, exists := m.users[user.Email]; exists {
		return ErrUserAlreadyExists
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return nil
}

func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user, exists := m.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepo) GetUserByID(ctx context.Context, id string) (*User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func TestRegisterAndLogin(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, "test-secret-key", 1*time.Hour)
	ctx := context.Background()

	// 1. Register User
	regReq := RegisterRequest{
		Email:    "buyer@example.com",
		Password: "password123",
		FullName: "Test Buyer",
		Role:     "buyer",
	}

	regResp, err := svc.Register(ctx, regReq)
	if err != nil {
		t.Fatalf("expected no error on registration, got: %v", err)
	}

	if regResp.Token == "" {
		t.Errorf("expected non-empty token")
	}
	if regResp.User.Email != "buyer@example.com" {
		t.Errorf("expected email buyer@example.com, got %s", regResp.User.Email)
	}

	// Verify stored password hash
	storedUser := repo.users["buyer@example.com"]
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte("password123")); err != nil {
		t.Errorf("stored password hash invalid: %v", err)
	}

	// 2. Duplicate Registration Test
	_, err = svc.Register(ctx, regReq)
	if err != ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists for duplicate email, got: %v", err)
	}

	// 3. Successful Login
	loginReq := LoginRequest{
		Email:    "buyer@example.com",
		Password: "password123",
	}
	loginResp, err := svc.Login(ctx, loginReq)
	if err != nil {
		t.Fatalf("expected successful login, got: %v", err)
	}
	if loginResp.Token == "" {
		t.Errorf("expected non-empty token on login")
	}

	// 4. Failed Login (Wrong Password)
	wrongLoginReq := LoginRequest{
		Email:    "buyer@example.com",
		Password: "wrongpassword",
	}
	_, err = svc.Login(ctx, wrongLoginReq)
	if err != ErrInvalidPassword {
		t.Errorf("expected ErrInvalidPassword for wrong password, got: %v", err)
	}
}
