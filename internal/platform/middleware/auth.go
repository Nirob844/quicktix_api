package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"quicktix/internal/platform/token"
)

type contextKey string

const userContextKey contextKey = "user_claims"

// Role constants
const (
	RoleBuyer     = "buyer"
	RoleOrganizer = "organizer"
	RoleAdmin     = "admin"
)

// WithUserContext returns a new context with user claims attached.
func WithUserContext(ctx context.Context, claims *token.JWTClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// GetUserFromContext extracts JWTClaims from the request context if present.
func GetUserFromContext(ctx context.Context) (*token.JWTClaims, bool) {
	claims, ok := ctx.Value(userContextKey).(*token.JWTClaims)
	return claims, ok
}

// Authenticate middleware validates the Bearer JWT in the Authorization header.
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing authorization header"})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid authorization header format"})
				return
			}

			tokenString := parts[1]
			claims, err := token.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
				return
			}

			ctx := WithUserContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware verifies that the authenticated user possesses one of the allowed roles.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			roleAllowed := false
			for _, role := range allowedRoles {
				if strings.EqualFold(claims.Role, role) {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "insufficient permissions"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireBuyer convenience middleware ensuring user is a buyer.
func RequireBuyer() func(http.Handler) http.Handler {
	return RequireRole(RoleBuyer)
}

// RequireOrganizer convenience middleware ensuring user is an organizer (or admin).
func RequireOrganizer() func(http.Handler) http.Handler {
	return RequireRole(RoleOrganizer, RoleAdmin)
}

// RequireAdmin convenience middleware ensuring user is an admin.
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(RoleAdmin)
}

// CheckOwnership checks if the authenticated context user is either the resource owner or an admin.
func CheckOwnership(ctx context.Context, resourceOwnerID string) bool {
	claims, ok := GetUserFromContext(ctx)
	if !ok {
		return false
	}
	if strings.EqualFold(claims.Role, RoleAdmin) {
		return true
	}
	return claims.UserID == resourceOwnerID
}
