package http

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter initializes the HTTP router with all application routes
func NewRouter(scheduleController *ScheduleController, authController *AuthController) *http.ServeMux {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("POST /events", scheduleController.CreateEvent)
	mux.HandleFunc("POST /events/{eventID}/import/sessionize/{sessionizeID}", scheduleController.ImportSessionize)

	// Auth
	mux.HandleFunc("POST /auth/signup", authController.SignUp)
	mux.HandleFunc("POST /auth/login", authController.Login)

	// Swagger
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return mux
}
