package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const requestIDContextKey contextKey = "request_id"
const RequestIDHeader = "X-Request-ID"

// WithRequestID returns a context containing the request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// GetRequestID retrieves the Request ID from the context if available.
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(requestIDContextKey).(string); ok {
		return reqID
	}
	return ""
}

// RequestID middleware ensures every incoming request has an X-Request-ID header and populates r.Context().
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		w.Header().Set(RequestIDHeader, reqID)
		ctx := WithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
