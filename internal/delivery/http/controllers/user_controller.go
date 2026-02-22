package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	h "multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"
	"multitrackticketing/internal/domain"
)

// UpdateUserRequest is the request body for PATCH /users/me. Both fields are optional.
type UpdateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// Validate implements Validator.
func (u UpdateUserRequest) Validate() []string {
	var errs []string
	if u.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*u.Email))
		if email == "" {
			errs = append(errs, "email cannot be empty")
		} else if !emailRegexp.MatchString(email) {
			errs = append(errs, "invalid email format")
		}
	}
	return errs
}

// UserController handles user profile endpoints.
type UserController struct {
	Logger  *slog.Logger
	Service domain.UserService
}

// NewUserController creates a UserController with the given logger and service.
func NewUserController(logger *slog.Logger, svc domain.UserService) *UserController {
	return &UserController{
		Logger:  logger,
		Service: svc,
	}
}

// GetMe godoc
// @Summary Get current user
// @Description Returns the authenticated user's profile (id, email, name, created_at, updated_at). Requires Bearer token.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} helpers.APIResponse "data contains the user"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /users/me [get]
func (c *UserController) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "unauthorized")
		return
	}
	user, err := c.Service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			h.WriteJSONError(w, http.StatusNotFound, h.ErrCodeNotFound, "user not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}
	h.WriteJSONSuccess(w, http.StatusOK, user)
}

// UpdateMe godoc
// @Summary Update current user
// @Description Update the authenticated user's profile. Accepts optional name and/or email. Email must be unique. Requires Bearer token.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateUserRequest true "Fields to update (name and/or email, both optional)"
// @Success 200 {object} helpers.APIResponse "data contains the updated user"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 409 {object} helpers.APIResponse "error.code: conflict"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /users/me [patch]
func (c *UserController) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "unauthorized")
		return
	}
	var req UpdateUserRequest
	if !h.DecodeAndValidate(w, r, &req) {
		return
	}
	user, err := c.Service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			h.WriteJSONError(w, http.StatusNotFound, h.ErrCodeNotFound, "user not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}
	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}
	if req.Email != nil {
		user.Email = strings.TrimSpace(strings.ToLower(*req.Email))
	}
	if err := c.Service.Update(r.Context(), user); err != nil {
		if errors.Is(err, domain.ErrDuplicateEmail) {
			h.WriteJSONError(w, http.StatusConflict, h.ErrCodeConflict, "email already in use")
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			h.WriteJSONError(w, http.StatusNotFound, h.ErrCodeNotFound, "user not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}
	h.WriteJSONSuccess(w, http.StatusOK, user)
}
