package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthMiddleware(t *testing.T) {
	jwtSecret := "test-jwt-secret-key"

	// Sample valid token
	validToken, err := GenerateJWT("usr_123", "buyer@example.com", "buyer", jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate test JWT: %v", err)
	}

	// Sample organizer token
	organizerToken, err := GenerateJWT("usr_456", "organizer@example.com", "organizer", jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate organizer JWT: %v", err)
	}

	// Sample expired token
	expiredToken, err := GenerateJWT("usr_789", "expired@example.com", "buyer", jwtSecret, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate expired JWT: %v", err)
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Errorf("expected claims in context")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		})
	})

	authMiddleware := Authenticate(jwtSecret)

	// 1. Test Valid Token
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rec := httptest.NewRecorder()

	authMiddleware(dummyHandler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for valid token, got %d", rec.Code)
	}

	var claimsResp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&claimsResp)
	if claimsResp["user_id"] != "usr_123" || claimsResp["role"] != "buyer" {
		t.Errorf("unexpected claims returned: %v", claimsResp)
	}

	// 2. Test Missing Header
	reqNoHeader := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	recNoHeader := httptest.NewRecorder()

	authMiddleware(dummyHandler).ServeHTTP(recNoHeader, reqNoHeader)

	if recNoHeader.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing header, got %d", recNoHeader.Code)
	}

	// 3. Test Invalid Format Header
	reqBadFormat := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqBadFormat.Header.Set("Authorization", "Basic invalidtokenformat")
	recBadFormat := httptest.NewRecorder()

	authMiddleware(dummyHandler).ServeHTTP(recBadFormat, reqBadFormat)

	if recBadFormat.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for bad header format, got %d", recBadFormat.Code)
	}

	// 4. Test Expired Token
	reqExpired := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqExpired.Header.Set("Authorization", "Bearer "+expiredToken)
	recExpired := httptest.NewRecorder()

	authMiddleware(dummyHandler).ServeHTTP(recExpired, reqExpired)

	if recExpired.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for expired token, got %d", recExpired.Code)
	}

	// 5. Test Role Authorization Middleware (RequireRole)
	organizerOnlyHandler := Authenticate(jwtSecret)(RequireRole("organizer", "admin")(dummyHandler))

	// Buyer accessing organizer endpoint -> 403 Forbidden
	reqBuyerAccess := httptest.NewRequest("GET", "/api/v1/events/create", nil)
	reqBuyerAccess.Header.Set("Authorization", "Bearer "+validToken)
	recBuyerAccess := httptest.NewRecorder()

	organizerOnlyHandler.ServeHTTP(recBuyerAccess, reqBuyerAccess)

	if recBuyerAccess.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for buyer accessing organizer endpoint, got %d", recBuyerAccess.Code)
	}

	// Organizer accessing organizer endpoint -> 200 OK
	reqOrgAccess := httptest.NewRequest("GET", "/api/v1/events/create", nil)
	reqOrgAccess.Header.Set("Authorization", "Bearer "+organizerToken)
	recOrgAccess := httptest.NewRecorder()

	organizerOnlyHandler.ServeHTTP(recOrgAccess, reqOrgAccess)

	if recOrgAccess.Code != http.StatusOK {
		t.Errorf("expected 200 OK for organizer accessing organizer endpoint, got %d", recOrgAccess.Code)
	}
}

