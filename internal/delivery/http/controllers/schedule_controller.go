package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"multitrackticketing/internal/delivery/http/helpers"
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

// CreateEventSuccessResponse is the success response envelope for POST /events (201).
type CreateEventSuccessResponse struct {
	Data  *domain.Event  `json:"data"`
	Error *helpers.APIError `json:"error"`
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
// @Success 201 {object} controllers.CreateEventSuccessResponse "data contains the created event"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events [post]
func (c *ScheduleController) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	now := time.Now()
	event := domain.NewEvent(req.Name, req.Slug, userID, now, now)
	if err := c.Service.CreateEvent(r.Context(), event); err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusCreated, event)
}

// GetEventByIDResponse is the response body for GET /events/{eventID}. Contains the event, its rooms, and sessions.
type GetEventByIDResponse struct {
	Event    *domain.Event     `json:"event"`
	Rooms    []*domain.Room    `json:"rooms"`
	Sessions []*domain.Session `json:"sessions"`
}

// GetEventByIDSuccessResponse is the success response envelope for GET /events/{eventID} (200).
type GetEventByIDSuccessResponse struct {
	Data  GetEventByIDResponse `json:"data"`
	Error *helpers.APIError   `json:"error"`
}

// GetEventByID godoc
// @Summary Get an event by ID
// @Description Returns the event, its rooms, and all sessions for that event. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} controllers.GetEventByIDSuccessResponse "data contains event, rooms, and sessions"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID} [get]
func (c *ScheduleController) GetEventByID(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	_, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	event, rooms, sessions, err := c.Service.GetEventByID(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event not found")
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, GetEventByIDResponse{Event: event, Rooms: rooms, Sessions: sessions})
}

// ImportSessionizeResponse is the data payload for POST /events/{eventID}/import/sessionize/{sessionizeID} (200).
type ImportSessionizeResponse struct {
	Status string `json:"status"`
}

// ImportSessionizeSuccessResponse is the success response envelope for POST /events/{eventID}/import/sessionize/{sessionizeID} (200).
type ImportSessionizeSuccessResponse struct {
	Data  ImportSessionizeResponse `json:"data"`
	Error *helpers.APIError        `json:"error"`
}

// ImportSessionize godoc
// @Summary Import schedule from Sessionize
// @Description Import rooms and sessions from Sessionize for a specific event
// @Tags events
// @Security BearerAuth
// @Param eventID path string true "Event ID"
// @Param sessionizeID path string true "Sessionize ID"
// @Success 200 {object} controllers.ImportSessionizeSuccessResponse "data contains status message"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/import/sessionize/{sessionizeID} [post]
func (c *ScheduleController) ImportSessionize(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionizeID := r.PathValue("sessionizeID")

	if eventID == "" || sessionizeID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or sessionizeID")
		return
	}

	if err := c.Service.ImportSessionizeData(r.Context(), eventID, sessionizeID); err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}

	helpers.WriteJSONSuccess(w, http.StatusOK, ImportSessionizeResponse{Status: "imported successfully"})
}

// ListMyEventsSuccessResponse is the success response envelope for GET /events/me (200).
type ListMyEventsSuccessResponse struct {
	Data  []*domain.Event `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// DeleteEventResponse is the data payload for DELETE /events/{eventID} (200).
type DeleteEventResponse struct {
	Status string `json:"status"`
}

// DeleteEventSuccessResponse is the success response envelope for DELETE /events/{eventID} (200).
type DeleteEventSuccessResponse struct {
	Data  DeleteEventResponse `json:"data"`
	Error *helpers.APIError   `json:"error"`
}

// DeleteEvent godoc
// @Summary Delete an event
// @Description Delete an event and all its associated data (rooms, sessions). Only the event owner can delete. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} controllers.DeleteEventSuccessResponse "data contains status"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID} [delete]
func (c *ScheduleController) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	if err := c.Service.DeleteEvent(r.Context(), eventID, userID); err != nil {
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
	helpers.WriteJSONSuccess(w, http.StatusOK, DeleteEventResponse{Status: "deleted"})
}

// ListMyEvents godoc
// @Summary List events owned by the current user
// @Description Returns events where the authenticated user is the owner. Requires Bearer token.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Success 200 {object} controllers.ListMyEventsSuccessResponse "data is an array of events"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/me [get]
func (c *ScheduleController) ListMyEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	events, err := c.Service.ListEventsByOwner(r.Context(), userID)
	if err != nil {
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	if events == nil {
		events = []*domain.Event{}
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, events)
}
