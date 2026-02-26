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
	attendeeController *controllers.AttendeeController,
	requireAuth AuthWrap,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Event management (protected)
	mux.HandleFunc("GET /events/me", requireAuth(scheduleController.ListMyEvents))
	mux.HandleFunc("GET /events/{eventID}", requireAuth(scheduleController.GetEventByID))
	mux.HandleFunc("PATCH /events/{eventID}", requireAuth(scheduleController.UpdateEvent))
	mux.HandleFunc("POST /events", requireAuth(scheduleController.CreateEvent))
	mux.HandleFunc("DELETE /events/{eventID}", requireAuth(scheduleController.DeleteEvent))
	mux.HandleFunc("PATCH /events/{eventID}/rooms/{roomID}/not-bookable", requireAuth(scheduleController.ToggleRoomNotBookable))
	mux.HandleFunc("GET /events/{eventID}/rooms", requireAuth(scheduleController.ListEventRooms))
	mux.HandleFunc("GET /events/{eventID}/rooms/{roomID}", requireAuth(scheduleController.GetEventRoom))
	mux.HandleFunc("PATCH /events/{eventID}/rooms/{roomID}", requireAuth(scheduleController.UpdateEventRoom))
	mux.HandleFunc("DELETE /events/{eventID}/rooms/{roomID}", requireAuth(scheduleController.DeleteEventRoom))
	mux.HandleFunc("GET /events/{eventID}/speakers", requireAuth(scheduleController.ListEventSpeakers))
	mux.HandleFunc("GET /events/{eventID}/speakers/{speakerID}", requireAuth(scheduleController.GetEventSpeaker))
	mux.HandleFunc("DELETE /events/{eventID}/speakers/{speakerID}", requireAuth(scheduleController.DeleteEventSpeaker))
	mux.HandleFunc("POST /events/{eventID}/speakers", requireAuth(scheduleController.CreateEventSpeaker))
	mux.HandleFunc("PATCH /events/{eventID}/sessions/{sessionID}", requireAuth(scheduleController.UpdateSessionSchedule))
	mux.HandleFunc("PATCH /events/{eventID}/sessions/{sessionID}/content", requireAuth(scheduleController.UpdateSessionContent))
	mux.HandleFunc("DELETE /events/{eventID}/sessions/{sessionID}", requireAuth(scheduleController.DeleteEventSession))
	mux.HandleFunc("POST /events/{eventID}/import/sessionize/{sessionizeID}", requireAuth(scheduleController.ImportSessionize))
	mux.HandleFunc("POST /events/{eventID}/team-members", requireAuth(scheduleController.AddEventTeamMember))
	mux.HandleFunc("GET /events/{eventID}/team-members", requireAuth(scheduleController.ListEventTeamMembers))
	mux.HandleFunc("DELETE /events/{eventID}/team-members/{userID}", requireAuth(scheduleController.RemoveEventTeamMember))
	mux.HandleFunc("GET /events/{eventID}/invitations", requireAuth(scheduleController.ListEventInvitations))
	mux.HandleFunc("POST /events/{eventID}/invitations", requireAuth(scheduleController.SendEventInvitations))

	// Attendee-facing (protected)
	mux.HandleFunc("POST /attendee/registrations", requireAuth(attendeeController.RegisterForEventByCode))
	mux.HandleFunc("POST /attendee/events/{eventID}/registrations", requireAuth(attendeeController.RegisterForEvent))
	mux.HandleFunc("GET /attendee/events", requireAuth(attendeeController.ListMyRegisteredEvents))
	mux.HandleFunc("GET /attendee/events/{eventID}/schedule", requireAuth(attendeeController.GetEventSchedule))

	// Auth (passwordless: request code then verify)
	mux.HandleFunc("POST /auth/login/request", userController.RequestLoginCode)
	mux.HandleFunc("POST /auth/login/verify", userController.VerifyLoginCode)

	// Users (protected)
	mux.HandleFunc("GET /users/me", requireAuth(userController.GetMe))
	mux.HandleFunc("PATCH /users/me", requireAuth(userController.UpdateMe))

	// Swagger
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return mux
}
