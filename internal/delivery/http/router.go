package http

import (
	"net/http"

	"multitrackticketing/internal/delivery/http/controllers"

	httpSwagger "github.com/swaggo/http-swagger"
)

// AuthWrap is a function that wraps a handler to require authentication.
type AuthWrap func(http.HandlerFunc) http.HandlerFunc

// NewRouter initializes the HTTP router with all application routes.
func NewRouter(
	scheduleController *controllers.ScheduleController,
	authController *controllers.AuthController,
	userController *controllers.UserController,
	requireAuth AuthWrap,
) *http.ServeMux {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("POST /events", scheduleController.CreateEvent)
	mux.HandleFunc("POST /events/{eventID}/import/sessionize/{sessionizeID}", scheduleController.ImportSessionize)

	// Auth
	mux.HandleFunc("POST /auth/signup", authController.SignUp)
	mux.HandleFunc("POST /auth/login", authController.Login)

	// Users (protected)
	mux.HandleFunc("GET /users/me", requireAuth(userController.GetMe))
	mux.HandleFunc("PATCH /users/me", requireAuth(userController.UpdateMe))

	// Swagger
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return mux
}
