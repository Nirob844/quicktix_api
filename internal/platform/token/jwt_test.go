package token

import (
	"testing"
	"time"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := "test-secret-key"
	userID := "usr_123"
	email := "test@example.com"
	role := "organizer"

	// 1. Generate Token
	tokenStr, err := GenerateJWT(userID, email, role, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error on token generation, got: %v", err)
	}

	// 2. Validate Token
	claims, err := ValidateJWT(tokenStr, secret)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("expected Role %s, got %s", role, claims.Role)
	}

	// 3. Test Invalid Secret
	_, err = ValidateJWT(tokenStr, "wrong-secret-key")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for wrong secret, got: %v", err)
	}

	// 4. Test Expired Token
	expiredToken, err := GenerateJWT(userID, email, role, secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}
	_, err = ValidateJWT(expiredToken, secret)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for expired token, got: %v", err)
	}
}
