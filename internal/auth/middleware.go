package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user_claims"

// WithUserContext returns a new context with user claims attached.
func WithUserContext(ctx context.Context, claims *JWTClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// GetUserFromContext extracts JWTClaims from the request context if present.
func GetUserFromContext(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value(userContextKey).(*JWTClaims)
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
			claims, err := ValidateJWT(tokenString, jwtSecret)
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
