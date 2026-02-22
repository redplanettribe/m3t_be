package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	h "multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"
	"multitrackticketing/internal/domain"
)

// CreateEventRequest is the request body for POST /events. Only name and slug are accepted.
type CreateEventRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Validate implements Validator. Returns error messages for required and format rules.
func (c CreateEventRequest) Validate() []string {
	var errs []string
	if c.Name == "" {
		errs = append(errs, "name is required")
	}
	if c.Slug == "" {
		errs = append(errs, "slug is required")
	}
	return errs
}

type ScheduleController struct {
	Logger  *slog.Logger
	Service domain.ManageScheduleService
}

func NewScheduleController(logger *slog.Logger, svc domain.ManageScheduleService) *ScheduleController {
	return &ScheduleController{
		Logger:  logger,
		Service: svc,
	}
}

// CreateEvent godoc
// @Summary Create a new event
// @Description Create a new conference event. Only name and slug are accepted in the body; id and timestamps are server-generated. The authenticated user becomes the event owner.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event body CreateEventRequest true "Event data (name and slug only)"
// @Success 201 {object} helpers.APIResponse "data contains the created event"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events [post]
func (c *ScheduleController) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if !h.DecodeAndValidate(w, r, &req) {
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "unauthorized")
		return
	}
	now := time.Now()
	event := domain.NewEvent(req.Name, req.Slug, userID, now, now)
	if err := c.Service.CreateEvent(r.Context(), event); err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}
	h.WriteJSONSuccess(w, http.StatusCreated, event)
}

// GetEventByIDResponse is the response body for GET /events/{eventID}. Contains the event, its rooms, and sessions.
type GetEventByIDResponse struct {
	Event    *domain.Event    `json:"event"`
	Rooms    []*domain.Room   `json:"rooms"`
	Sessions []*domain.Session `json:"sessions"`
}

// GetEventByID godoc
// @Summary Get an event by ID
// @Description Returns the event, its rooms, and all sessions for that event. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} helpers.APIResponse "data contains event, rooms, and sessions"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID} [get]
func (c *ScheduleController) GetEventByID(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		h.WriteJSONError(w, http.StatusBadRequest, h.ErrCodeBadRequest, "missing eventID")
		return
	}
	_, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "unauthorized")
		return
	}
	event, rooms, sessions, err := c.Service.GetEventByID(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			h.WriteJSONError(w, http.StatusNotFound, h.ErrCodeNotFound, "event not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}
	h.WriteJSONSuccess(w, http.StatusOK, GetEventByIDResponse{Event: event, Rooms: rooms, Sessions: sessions})
}

// ImportSessionize godoc
// @Summary Import schedule from Sessionize
// @Description Import rooms and sessions from Sessionize for a specific event
// @Tags events
// @Security BearerAuth
// @Param eventID path string true "Event ID"
// @Param sessionizeID path string true "Sessionize ID"
// @Success 200 {object} helpers.APIResponse "data contains status message"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/import/sessionize/{sessionizeID} [post]
func (c *ScheduleController) ImportSessionize(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionizeID := r.PathValue("sessionizeID")

	if eventID == "" || sessionizeID == "" {
		h.WriteJSONError(w, http.StatusBadRequest, h.ErrCodeBadRequest, "missing eventID or sessionizeID")
		return
	}

	if err := c.Service.ImportSessionizeData(r.Context(), eventID, sessionizeID); err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}

	h.WriteJSONSuccess(w, http.StatusOK, map[string]string{"status": "imported successfully"})
}

// ListMyEvents godoc
// @Summary List events owned by the current user
// @Description Returns events where the authenticated user is the owner. Requires Bearer token.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Success 200 {object} helpers.APIResponse "data is an array of events"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/me [get]
func (c *ScheduleController) ListMyEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "unauthorized")
		return
	}
	events, err := c.Service.ListEventsByOwner(r.Context(), userID)
	if err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		h.WriteJSONError(w, http.StatusInternalServerError, h.ErrCodeInternalError, err.Error())
		return
	}
	if events == nil {
		events = []*domain.Event{}
	}
	h.WriteJSONSuccess(w, http.StatusOK, events)
}
