package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response payload with the specified HTTP status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes a standardized JSON error message response.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// ValidationError writes a 400 Bad Request response containing field-level validation errors.
func ValidationError(w http.ResponseWriter, errs map[string]string) {
	JSON(w, http.StatusBadRequest, map[string]any{
		"error":   "validation failed",
		"details": errs,
	})
}

// BadRequest writes a 400 Bad Request JSON error response.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Unauthorized writes a 401 Unauthorized JSON error response.
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden writes a 403 Forbidden JSON error response.
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

// NotFound writes a 404 Not Found JSON error response.
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// Conflict writes a 409 Conflict JSON error response.
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, message)
}

// InternalServerError writes a 500 Internal Server Error JSON response.
func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}
