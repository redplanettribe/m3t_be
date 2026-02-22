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
	userController *controllers.UserController,
	requireAuth AuthWrap,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Event management (protected)
	mux.HandleFunc("GET /events/me", requireAuth(scheduleController.ListMyEvents))
	mux.HandleFunc("GET /events/{eventID}", requireAuth(scheduleController.GetEventByID))
	mux.HandleFunc("POST /events", requireAuth(scheduleController.CreateEvent))
	mux.HandleFunc("POST /events/{eventID}/import/sessionize/{sessionizeID}", requireAuth(scheduleController.ImportSessionize))

	// Auth (handled by user controller)
	mux.HandleFunc("POST /auth/signup", userController.SignUp)
	mux.HandleFunc("POST /auth/login", userController.Login)

	// Users (protected)
	mux.HandleFunc("GET /users/me", requireAuth(userController.GetMe))
	mux.HandleFunc("PATCH /users/me", requireAuth(userController.UpdateMe))

	// Swagger
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return mux
}
