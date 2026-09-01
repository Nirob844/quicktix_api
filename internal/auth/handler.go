package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"quicktix/internal/platform/middleware"
	"quicktix/internal/platform/response"
)

// Handler handles HTTP requests for user authentication.
type Handler struct {
	service Service
}

// NewHandler creates a new auth HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register handles user registration requests.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			response.Conflict(w, err.Error())
			return
		}
		if errors.Is(err, ErrInvalidRole) {
			response.BadRequest(w, err.Error())
			return
		}

		response.InternalServerError(w, "failed to register user")
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

// Login handles user login requests.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			response.Unauthorized(w, err.Error())
			return
		}

		response.InternalServerError(w, "failed to authenticate user")
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// Me handles GET /api/v1/auth/me requests to return current authenticated user claims.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}

// ChangePassword handles POST /api/v1/auth/change-password requests.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	if err := h.service.ChangePassword(r.Context(), claims.UserID, req); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			response.Unauthorized(w, "old password is incorrect")
			return
		}
		if errors.Is(err, ErrSamePassword) {
			response.BadRequest(w, err.Error())
			return
		}

		response.InternalServerError(w, "failed to change password")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password requests.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	token, err := h.service.ForgotPassword(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(w, "user with this email does not exist")
			return
		}
		response.InternalServerError(w, "failed to process forgot password request")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message":     "password reset token generated and sent to email",
		"reset_token": token,
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password requests.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON payload")
		return
	}

	if valErrs := req.Validate(); valErrs != nil {
		response.ValidationError(w, valErrs)
		return
	}

	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		if errors.Is(err, ErrInvalidResetToken) {
			response.BadRequest(w, err.Error())
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalServerError(w, "failed to reset password")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
}
