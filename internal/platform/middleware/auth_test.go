package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quicktix/internal/platform/token"
)

func TestAuthAndRBACMiddleware(t *testing.T) {
	jwtSecret := "platform-secret-key"

	validToken, err := token.GenerateJWT("usr_buyer_1", "buyer@example.com", RoleBuyer, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate buyer JWT: %v", err)
	}

	organizerToken, err := token.GenerateJWT("usr_org_1", "org@example.com", RoleOrganizer, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate organizer JWT: %v", err)
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Errorf("expected claims in context")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"user_id": claims.UserID,
			"role":    claims.Role,
		})
	})

	authMw := Authenticate(jwtSecret)

	// 1. Test Valid Bearer Token
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rec := httptest.NewRecorder()

	authMw(dummyHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for valid token, got %d", rec.Code)
	}

	// 2. Test Missing Header -> 401
	reqNoHeader := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	recNoHeader := httptest.NewRecorder()

	authMw(dummyHandler).ServeHTTP(recNoHeader, reqNoHeader)
	if recNoHeader.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing header, got %d", recNoHeader.Code)
	}

	// 3. Test RequireOrganizer RBAC
	orgOnlyMw := Authenticate(jwtSecret)(RequireOrganizer()(dummyHandler))

	// Buyer Access -> 403
	recBuyer := httptest.NewRecorder()
	orgOnlyMw.ServeHTTP(recBuyer, req)
	if recBuyer.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for buyer on organizer route, got %d", recBuyer.Code)
	}

	// Organizer Access -> 200
	reqOrg := httptest.NewRequest("GET", "/api/v1/organizer/events", nil)
	reqOrg.Header.Set("Authorization", "Bearer "+organizerToken)
	recOrg := httptest.NewRecorder()

	orgOnlyMw.ServeHTTP(recOrg, reqOrg)
	if recOrg.Code != http.StatusOK {
		t.Errorf("expected 200 OK for organizer on organizer route, got %d", recOrg.Code)
	}

	// 4. Test CheckOwnership
	ctxOrg := WithUserContext(context.Background(), &token.JWTClaims{UserID: "usr_org_1", Role: RoleOrganizer})
	if !CheckOwnership(ctxOrg, "usr_org_1") {
		t.Errorf("expected CheckOwnership to return true for matching owner")
	}

	if CheckOwnership(ctxOrg, "usr_other") {
		t.Errorf("expected CheckOwnership to return false for non-matching owner")
	}
}
