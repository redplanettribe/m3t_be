package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"
	"multitrackticketing/internal/domain"
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// RequestLoginCodeRequest is the request body for POST /auth/login/request
type RequestLoginCodeRequest struct {
	Email string `json:"email"`
}

// Validate implements Validator.
func (r RequestLoginCodeRequest) Validate() []string {
	var errs []string
	email := strings.TrimSpace(strings.ToLower(r.Email))
	if email == "" {
		errs = append(errs, "email is required")
	} else if !emailRegexp.MatchString(email) {
		errs = append(errs, "invalid email format")
	}
	return errs
}

// VerifyLoginCodeRequest is the request body for POST /auth/login/verify
type VerifyLoginCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// Validate implements Validator.
func (v VerifyLoginCodeRequest) Validate() []string {
	var errs []string
	email := strings.TrimSpace(strings.ToLower(v.Email))
	if email == "" {
		errs = append(errs, "email is required")
	} else if !emailRegexp.MatchString(email) {
		errs = append(errs, "invalid email format")
	}
	code := strings.TrimSpace(v.Code)
	if code == "" {
		errs = append(errs, "code is required")
	} else if len(code) != 6 {
		errs = append(errs, "code must be 6 digits")
	} else {
		for _, c := range code {
			if c < '0' || c > '9' {
				errs = append(errs, "code must be 6 digits")
				break
			}
		}
	}
	return errs
}

// LoginResponse is the response body for POST /auth/login/verify
type LoginResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"token_type"`
	User      *domain.User `json:"user"`
}

// UpdateUserRequest is the request body for PATCH /users/me. All fields are optional. Email cannot be updated.
type UpdateUserRequest struct {
	Name     *string `json:"name"`
	LastName *string `json:"last_name"`
}

// Validate implements Validator.
func (u UpdateUserRequest) Validate() []string {
	return nil
}

// LoginSuccessResponse is the success response envelope for POST /auth/login/verify (200).
type LoginSuccessResponse struct {
	Data  LoginResponse `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// GetMeSuccessResponse is the success response envelope for GET /users/me (200).
type GetMeSuccessResponse struct {
	Data  *domain.User `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// UpdateUserSuccessResponse is the success response envelope for PATCH /users/me (200).
type UpdateUserSuccessResponse struct {
	Data  *domain.User `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// UserController handles user profile and auth endpoints.
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

// RequestLoginCode godoc
// @Summary Request a login code
// @Description Send a one-time login code to the given email. The code expires after a short period.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RequestLoginCodeRequest true "Email to receive the code"
// @Success 200 {object} helpers.APIResponse "success"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /auth/login/request [post]
func (c *UserController) RequestLoginCode(w http.ResponseWriter, r *http.Request) {
	var req RequestLoginCodeRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	err := c.Service.RequestLoginCode(r.Context(), email)
	if err != nil {
		if strings.Contains(err.Error(), "invalid email") {
			helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, nil)
}

// VerifyLoginCode godoc
// @Summary Verify login code and get token
// @Description Exchange the one-time code for a JWT and user. Creates the user on first successful login. Returns token, token_type, and user.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body VerifyLoginCodeRequest true "Email and code"
// @Success 200 {object} controllers.LoginSuccessResponse "data contains token, token_type, and user"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /auth/login/verify [post]
func (c *UserController) VerifyLoginCode(w http.ResponseWriter, r *http.Request) {
	var req VerifyLoginCodeRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	code := strings.TrimSpace(req.Code)
	token, user, err := c.Service.VerifyLoginCode(r.Context(), email, code)
	if err != nil {
		if strings.Contains(err.Error(), "invalid or expired code") {
			helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "invalid or expired code")
			return
		}
		if strings.Contains(err.Error(), "invalid email") {
			helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, LoginResponse{Token: token, TokenType: "Bearer", User: user})
}

// GetMe godoc
// @Summary Get current user
// @Description Returns the authenticated user's profile (id, email, name, created_at, updated_at). Requires Bearer token.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} controllers.GetMeSuccessResponse "data contains the user"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /users/me [get]
func (c *UserController) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	user, err := c.Service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "user not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, user)
}

// UpdateMe godoc
// @Summary Update current user
// @Description Update the authenticated user's profile. Accepts optional name and/or last_name only; email cannot be updated. Requires Bearer token.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateUserRequest true "Fields to update (name and/or last_name, both optional)"
// @Success 200 {object} controllers.UpdateUserSuccessResponse "data contains the updated user"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 409 {object} helpers.APIResponse "error.code: conflict"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /users/me [patch]
func (c *UserController) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	var req UpdateUserRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	user, err := c.Service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "user not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}
	if req.LastName != nil {
		user.LastName = strings.TrimSpace(*req.LastName)
	}
	if err := c.Service.Update(r.Context(), user); err != nil {
		if errors.Is(err, domain.ErrDuplicateEmail) {
			helpers.WriteJSONError(w, http.StatusConflict, helpers.ErrCodeConflict, "email already in use")
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "user not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, user)
}
