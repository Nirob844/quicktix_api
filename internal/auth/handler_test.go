package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthHTTPHandlers(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, "secret-key-123", 1*time.Hour)
	handler := NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", handler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)

	server := httptest.NewServer(mux)
	defer server.Close()

	// 1. Test Registration Endpoint
	regPayload := map[string]string{
		"email":     "organizer@example.com",
		"password":  "mypassword123",
		"full_name": "Event Organizer",
		"role":      "organizer",
	}
	body, _ := json.Marshal(regPayload)

	resp, err := http.Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send register request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201 Created, got %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	if authResp.Token == "" {
		t.Errorf("expected non-empty JWT token")
	}
	if authResp.User.Email != "organizer@example.com" {
		t.Errorf("expected email organizer@example.com, got %s", authResp.User.Email)
	}
	if authResp.User.Role != "organizer" {
		t.Errorf("expected role organizer, got %s", authResp.User.Role)
	}

	// 2. Test Login Endpoint
	loginPayload := map[string]string{
		"email":    "organizer@example.com",
		"password": "mypassword123",
	}
	loginBody, _ := json.Marshal(loginPayload)

	loginResp, err := http.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(loginBody))
	if err != nil {
		t.Fatalf("failed to send login request: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", loginResp.StatusCode)
	}

	var loginAuthResp AuthResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginAuthResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loginAuthResp.Token == "" {
		t.Errorf("expected non-empty JWT token on login")
	}

	// 3. Test Invalid Login (Wrong Password)
	badLoginPayload := map[string]string{
		"email":    "organizer@example.com",
		"password": "wrongpassword",
	}
	badBody, _ := json.Marshal(badLoginPayload)

	badResp, err := http.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(badBody))
	if err != nil {
		t.Fatalf("failed to send bad login request: %v", err)
	}
	defer badResp.Body.Close()

	if badResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", badResp.StatusCode)
	}
}
