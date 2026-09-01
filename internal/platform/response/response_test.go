package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseHelpers(t *testing.T) {
	// 1. Test JSON Helper
	recJSON := httptest.NewRecorder()
	JSON(recJSON, http.StatusOK, map[string]string{"key": "value"})

	if recJSON.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recJSON.Code)
	}
	if contentType := recJSON.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// 2. Test Error Helper
	recErr := httptest.NewRecorder()
	BadRequest(recErr, "invalid request")

	if recErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", recErr.Code)
	}

	var errBody map[string]string
	_ = json.NewDecoder(recErr.Body).Decode(&errBody)
	if errBody["error"] != "invalid request" {
		t.Errorf("expected error message 'invalid request', got %s", errBody["error"])
	}

	// 3. Test ValidationError Helper
	recVal := httptest.NewRecorder()
	ValidationError(recVal, map[string]string{"email": "email is required"})

	if recVal.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for validation error, got %d", recVal.Code)
	}

	var valBody map[string]any
	_ = json.NewDecoder(recVal.Body).Decode(&valBody)
	if valBody["error"] != "validation failed" {
		t.Errorf("expected 'validation failed', got %v", valBody["error"])
	}
}
