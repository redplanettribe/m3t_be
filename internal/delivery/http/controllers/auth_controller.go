package controllers

import (
	"log/slog"
	h "multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/domain"
	"net/http"
	"regexp"
	"strings"
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
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
}

type AuthController struct {
	Logger  *slog.Logger
	Service domain.AuthService
}

func NewAuthController(logger *slog.Logger, svc domain.AuthService) *AuthController {
	return &AuthController{
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
// @Success 201 {object} helpers.APIResponse "data contains the created user"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /auth/signup [post]
func (c *AuthController) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if !h.DecodeAndValidate(w, r, &req) {
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
			h.WriteJSONError(w, http.StatusBadRequest, h.ErrCodeBadRequest, "email already registered")
			return
		}
		if strings.Contains(err.Error(), "invalid email") || strings.Contains(err.Error(), "password must be") {
			h.WriteJSONError(w, http.StatusBadRequest, h.ErrCodeBadRequest, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}

	h.WriteJSONSuccess(w, http.StatusCreated, user)
}

// Login godoc
// @Summary Log in
// @Description Authenticate with email and password. Returns a JWT containing user id, email, and roles.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials"
// @Success 200 {object} helpers.APIResponse "data contains token and token_type"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !h.DecodeAndValidate(w, r, &req) {
		return
	}
	token, err := c.Service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "invalid credentials")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}

	h.WriteJSONSuccess(w, http.StatusOK, LoginResponse{Token: token, TokenType: "Bearer"})
}
