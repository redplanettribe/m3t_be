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

// uuidRegexAttendee matches a canonical UUID string (8-4-4-4-12 hex).
var uuidRegexAttendee = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type AttendeeController struct {
	Logger  *slog.Logger
	Service domain.AttendeeService
}

func NewAttendeeController(logger *slog.Logger, svc domain.AttendeeService) *AttendeeController {
	return &AttendeeController{
		Logger:  logger,
		Service: svc,
	}
}

// RegisterForEventSuccessResponse is the success response envelope for POST /attendee/events/{eventID}/registrations and POST /attendee/registrations (200 or 201).
type RegisterForEventSuccessResponse struct {
	Data  *domain.EventRegistration `json:"data"`
	Error *helpers.APIError          `json:"error"`
}

// RegisterForEvent godoc
// @Summary Register the current attendee for an event
// @Description Registers the authenticated user as an attendee for the specified event. Idempotent: returns 201 when a new registration is created, 200 when already registered.
// @Tags attendee
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} controllers.RegisterForEventSuccessResponse "Already registered"
// @Success 201 {object} controllers.RegisterForEventSuccessResponse "New registration created"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /attendee/events/{eventID}/registrations [post]
func (c *AttendeeController) RegisterForEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	if !uuidRegexAttendee.MatchString(eventID) {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "invalid eventID")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}

	reg, created, err := c.Service.RegisterForEvent(r.Context(), eventID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	if created {
		helpers.WriteJSONSuccess(w, http.StatusCreated, reg)
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, reg)
}

// RegisterForEventByCodeRequest is the request body for POST /attendee/registrations.
type RegisterForEventByCodeRequest struct {
	EventCode string `json:"event_code"`
}

// Validate implements helpers.Validator.
func (r *RegisterForEventByCodeRequest) Validate() []string {
	code := strings.ToLower(strings.TrimSpace(r.EventCode))
	if code == "" {
		return []string{"event_code is required"}
	}
	if len(code) != 4 {
		return []string{"event_code must be exactly 4 characters"}
	}
	for _, c := range code {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			continue
		}
		return []string{"event_code must contain only lowercase letters and digits"}
	}
	r.EventCode = code
	return nil
}

// RegisterForEventByCode godoc
// @Summary Register for an event by event code
// @Description Registers the authenticated user as an attendee for the event with the given event_code. Idempotent: returns 201 when a new registration is created, 200 when already registered.
// @Tags attendee
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body controllers.RegisterForEventByCodeRequest true "Event code (4 characters)"
// @Success 200 {object} controllers.RegisterForEventSuccessResponse "Already registered"
// @Success 201 {object} controllers.RegisterForEventSuccessResponse "New registration created"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /attendee/registrations [post]
func (c *AttendeeController) RegisterForEventByCode(w http.ResponseWriter, r *http.Request) {
	var req RegisterForEventByCodeRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}

	reg, created, err := c.Service.RegisterForEventByCode(r.Context(), req.EventCode, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	if created {
		helpers.WriteJSONSuccess(w, http.StatusCreated, reg)
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, reg)
}

// ListMyRegisteredEventsItem is an item in the response for GET /attendee/events.
type ListMyRegisteredEventsItem struct {
	Event        *domain.Event               `json:"event"`
	Registration *domain.EventRegistration   `json:"registration"`
}

// ListMyRegisteredEventsSuccessResponse is the success response envelope for GET /attendee/events (200).
type ListMyRegisteredEventsSuccessResponse struct {
	Data  []ListMyRegisteredEventsItem `json:"data"`
	Error *helpers.APIError            `json:"error"`
}

// ListMyRegisteredEvents godoc
// @Summary Get events the current user is registered for
// @Description Returns the list of events the authenticated user is registered for, including registration metadata.
// @Tags attendee
// @Produce json
// @Security BearerAuth
// @Success 200 {object} controllers.ListMyRegisteredEventsSuccessResponse "data is an array of event + registration objects"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /attendee/events [get]
func (c *AttendeeController) ListMyRegisteredEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}

	items, err := c.Service.ListMyRegisteredEvents(r.Context(), userID)
	if err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}

	if items == nil {
		items = []*domain.EventRegistrationWithEvent{}
	}

	responseItems := make([]ListMyRegisteredEventsItem, 0, len(items))
	for _, it := range items {
		responseItems = append(responseItems, ListMyRegisteredEventsItem{
			Event:        it.Event,
			Registration: it.Registration,
		})
	}

	helpers.WriteJSONSuccess(w, http.StatusOK, responseItems)
}

// GetEventScheduleSuccessResponse is the success response envelope for GET /attendee/events/{eventID}/schedule (200).
type GetEventScheduleSuccessResponse struct {
	Data  *domain.EventSchedule `json:"data"`
	Error *helpers.APIError     `json:"error"`
}

// GetEventSchedule godoc
// @Summary Get event schedule for a registered attendee
// @Description Returns the event schedule (event plus bookable rooms with nested sessions) for the specified event. Only registered attendees or the event owner may access this. Only rooms with not_bookable=false are included.
// @Tags attendee
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} controllers.GetEventScheduleSuccessResponse "data contains event and rooms (bookable only) with nested sessions"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not registered or owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /attendee/events/{eventID}/schedule [get]
func (c *AttendeeController) GetEventSchedule(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	if !uuidRegexAttendee.MatchString(eventID) {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "invalid eventID")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}

	schedule, err := c.Service.GetEventSchedule(r.Context(), eventID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			helpers.WriteJSONError(w, http.StatusForbidden, helpers.ErrCodeForbidden, "forbidden")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}

	helpers.WriteJSONSuccess(w, http.StatusOK, schedule)
}

