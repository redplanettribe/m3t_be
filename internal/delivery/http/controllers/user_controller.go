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

// SignUpRequest is the request body for POST /auth/signup
type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"` // optional: "admin" or "attendee" (defaults to "attendee")
}

// Validate implements Validator.
func (s SignUpRequest) Validate() []string {
	var errs []string
	email := strings.TrimSpace(strings.ToLower(s.Email))
	if email == "" {
		errs = append(errs, "email is required")
	} else if !emailRegexp.MatchString(email) {
		errs = append(errs, "invalid email format")
	}
	if s.Password == "" {
		errs = append(errs, "password is required")
	} else if len(s.Password) < 8 {
		errs = append(errs, "password must be at least 8 characters")
	}
	role := strings.TrimSpace(strings.ToLower(s.Role))
	if role == "atendee" {
		role = "attendee"
	}
	if role != "" && role != "admin" && role != "attendee" {
		errs = append(errs, "role must be \"admin\" or \"attendee\"")
	}
	return errs
}

// LoginRequest is the request body for POST /auth/login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate implements Validator.
func (l LoginRequest) Validate() []string {
	var errs []string
	if strings.TrimSpace(l.Email) == "" {
		errs = append(errs, "email is required")
	}
	if l.Password == "" {
		errs = append(errs, "password is required")
	}
	return errs
}

// LoginResponse is the response body for POST /auth/login
type LoginResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"token_type"`
	User      *domain.User `json:"user"`
}

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

// SignUpSuccessResponse is the success response envelope for POST /auth/signup (201).
type SignUpSuccessResponse struct {
	Data  *domain.User `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// LoginSuccessResponse is the success response envelope for POST /auth/login (200).
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

// SignUp godoc
// @Summary Sign up a new user
// @Description Create a new user with email, password, and name. Optional role: "admin" or "attendee" (defaults to "attendee"). Password is stored hashed.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body SignUpRequest true "Sign-up data"
// @Success 201 {object} controllers.SignUpSuccessResponse "data contains the created user"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /auth/signup [post]
func (c *UserController) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	role := strings.TrimSpace(strings.ToLower(req.Role))
	if role == "atendee" {
		role = "attendee"
	}
	user, err := c.Service.SignUp(r.Context(), email, req.Password, req.Name, role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "already exists") {
			helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "email already registered")
			return
		}
		if strings.Contains(err.Error(), "invalid email") || strings.Contains(err.Error(), "password must be") {
			helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}

	helpers.WriteJSONSuccess(w, http.StatusCreated, user)
}

// Login godoc
// @Summary Log in
// @Description Authenticate with email and password. Returns a JWT and the user. JWT contains user id, email, and roles.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials"
// @Success 200 {object} controllers.LoginSuccessResponse "data contains token, token_type, and user"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /auth/login [post]
func (c *UserController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	token, user, err := c.Service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "invalid credentials")
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
// @Description Update the authenticated user's profile. Accepts optional name and/or email. Email must be unique. Requires Bearer token.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateUserRequest true "Fields to update (name and/or email, both optional)"
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
	if req.Email != nil {
		user.Email = strings.TrimSpace(strings.ToLower(*req.Email))
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
