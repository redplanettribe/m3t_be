package http

import (
	"encoding/json"
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

// LoginRequest is the request body for POST /auth/login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the response body for POST /auth/login
type LoginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
}

type AuthController struct {
	Service domain.AuthService
}

func NewAuthController(svc domain.AuthService) *AuthController {
	return &AuthController{Service: svc}
}

// SignUp godoc
// @Summary Sign up a new user
// @Description Create a new user with email, password, and name. Optional role: "admin" or "attendee" (defaults to "attendee"). Password is stored hashed.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body SignUpRequest true "Sign-up data"
// @Success 201 {object} domain.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/signup [post]
func (c *AuthController) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	if !emailRegexp.MatchString(email) {
		http.Error(w, "invalid email format", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	role := strings.TrimSpace(strings.ToLower(req.Role))
	if role == "atendee" {
		role = "attendee"
	}
	if role != "" && role != "admin" && role != "attendee" {
		http.Error(w, "role must be \"admin\" or \"attendee\"", http.StatusBadRequest)
		return
	}

	user, err := c.Service.SignUp(r.Context(), email, req.Password, req.Name, role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "already exists") {
			http.Error(w, "email already registered", http.StatusBadRequest)
			return
		}
		if strings.Contains(err.Error(), "invalid email") || strings.Contains(err.Error(), "password must be") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Login godoc
// @Summary Log in
// @Description Authenticate with email and password. Returns a JWT containing user id, email, and roles.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}

	token, err := c.Service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{Token: token, TokenType: "Bearer"})
}
