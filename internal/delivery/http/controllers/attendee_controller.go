package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"

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

// RegisterForEvent godoc
// @Summary Register the current attendee for an event
// @Description Registers the authenticated user as an attendee for the specified event. Idempotent if already registered.
// @Tags attendee
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 201 {object} helpers.APIResponse "data contains the event registration"
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

	reg, err := c.Service.RegisterForEvent(r.Context(), eventID, userID)
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

	helpers.WriteJSONSuccess(w, http.StatusCreated, reg)
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

