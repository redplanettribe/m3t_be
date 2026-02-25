package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"
	"multitrackticketing/internal/domain"
)

// uuidRegex matches a canonical UUID string (8-4-4-4-12 hex).
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// emailRegex matches a simple email format (local@domain with at least one dot in domain).
var emailRegex = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)

// CreateEventRequest is the request body for POST /events. Only name is accepted.
type CreateEventRequest struct {
	Name string `json:"name"`
}

// Validate implements Validator. Returns error messages for required and format rules.
func (c CreateEventRequest) Validate() []string {
	var errs []string
	if c.Name == "" {
		errs = append(errs, "name is required")
	}
	return errs
}

// CreateEventSuccessResponse is the success response envelope for POST /events (201).
type CreateEventSuccessResponse struct {
	Data  *domain.Event     `json:"data"`
	Error *helpers.APIError `json:"error"`
}

type ScheduleController struct {
	Logger  *slog.Logger
	Service domain.EventService
}

func NewScheduleController(logger *slog.Logger, svc domain.EventService) *ScheduleController {
	return &ScheduleController{
		Logger:  logger,
		Service: svc,
	}
}

// CreateEvent godoc
// @Summary Create a new event
// @Description Create a new conference event. Only name is accepted in the body; id, event_code and timestamps are server-generated. The authenticated user becomes the event owner.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event body CreateEventRequest true "Event data (name only)"
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
	event := domain.NewEvent(req.Name, "", userID, now, now)
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
	Error *helpers.APIError    `json:"error"`
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

// UpdateEventRequest is the request body for PATCH /events/{eventID}. All fields optional; omitted fields are unchanged.
type UpdateEventRequest struct {
	Date         *time.Time `json:"date"`
	Description  *string    `json:"description"`
	LocationLat  *float64   `json:"location_lat"`
	LocationLng  *float64   `json:"location_lng"`
}

// Validate implements Validator. Optional bounds for lat (-90..90) and lng (-180..180).
func (u UpdateEventRequest) Validate() []string {
	var errs []string
	if u.LocationLat != nil && (*u.LocationLat < -90 || *u.LocationLat > 90) {
		errs = append(errs, "location_lat must be between -90 and 90")
	}
	if u.LocationLng != nil && (*u.LocationLng < -180 || *u.LocationLng > 180) {
		errs = append(errs, "location_lng must be between -180 and 180")
	}
	return errs
}

// UpdateEventSuccessResponse is the success response envelope for PATCH /events/{eventID} (200).
type UpdateEventSuccessResponse struct {
	Data  *domain.Event     `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// UpdateEvent godoc
// @Summary Update event details
// @Description Updates event date, description, and location (lat/lng). Only the event owner can update. Optional fields omitted from body are unchanged. Requires authentication.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param body body UpdateEventRequest true "Fields to update (all optional)"
// @Success 200 {object} controllers.UpdateEventSuccessResponse "data contains the updated event"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID} [patch]
func (c *ScheduleController) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	var req UpdateEventRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	event, err := c.Service.UpdateEvent(r.Context(), eventID, ownerID, req.Date, req.Description, req.LocationLat, req.LocationLng)
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
	helpers.WriteJSONSuccess(w, http.StatusOK, event)
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
	Data  []*domain.Event   `json:"data"`
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

// ToggleRoomNotBookableSuccessResponse is the success response envelope for PATCH /events/{eventID}/rooms/{roomID}/not-bookable (200).
type ToggleRoomNotBookableSuccessResponse struct {
	Data  *domain.Room      `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// ToggleRoomNotBookable godoc
// @Summary Toggle room not_bookable flag
// @Description Toggles the not_bookable flag for a room. Only the event owner can toggle. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param roomID path string true "Room ID (UUID)"
// @Success 200 {object} controllers.ToggleRoomNotBookableSuccessResponse "data contains the updated room"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/rooms/{roomID}/not-bookable [patch]
func (c *ScheduleController) ToggleRoomNotBookable(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	roomID := r.PathValue("roomID")
	if eventID == "" || roomID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or roomID")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	room, err := c.Service.ToggleRoomNotBookable(r.Context(), eventID, roomID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or room not found")
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
	helpers.WriteJSONSuccess(w, http.StatusOK, room)
}

// UpdateRoomRequest is the request body for PATCH /events/{eventID}/rooms/{roomID}.
type UpdateRoomRequest struct {
	Capacity      int     `json:"capacity"`
	Description   string  `json:"description"`
	HowToGetThere string  `json:"how_to_get_there"`
	NotBookable   *bool   `json:"not_bookable"`
}

// Validate implements Validator.
func (u UpdateRoomRequest) Validate() []string {
	var errs []string
	if u.Capacity < 0 {
		errs = append(errs, "capacity must be non-negative")
	}
	return errs
}

// GetRoomSuccessResponse is the success response envelope for GET /events/{eventID}/rooms/{roomID} (200).
type GetRoomSuccessResponse struct {
	Data  *domain.Room      `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// ListRoomsSuccessResponse is the success response envelope for GET /events/{eventID}/rooms (200).
type ListRoomsSuccessResponse struct {
	Data  []*domain.Room   `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// UpdateRoomSuccessResponse is the success response envelope for PATCH /events/{eventID}/rooms/{roomID} (200).
type UpdateRoomSuccessResponse struct {
	Data  *domain.Room      `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// DeleteRoomSuccessResponse is the success response envelope for DELETE /events/{eventID}/rooms/{roomID} (200).
type DeleteRoomSuccessResponse struct {
	Data  DeleteEventResponse `json:"data"`
	Error *helpers.APIError   `json:"error"`
}

// ListEventRooms godoc
// @Summary List rooms for an event
// @Description Returns the list of rooms for the event. Only the event owner can list. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} controllers.ListRoomsSuccessResponse "data is an array of rooms"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/rooms [get]
func (c *ScheduleController) ListEventRooms(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	rooms, err := c.Service.ListEventRooms(r.Context(), eventID, ownerID)
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
	if rooms == nil {
		rooms = []*domain.Room{}
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, rooms)
}

// GetEventRoom godoc
// @Summary Get a room by ID
// @Description Returns a single room for the event. Only the event owner can access. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param roomID path string true "Room ID (UUID)"
// @Success 200 {object} controllers.GetRoomSuccessResponse "data contains the room"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/rooms/{roomID} [get]
func (c *ScheduleController) GetEventRoom(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	roomID := r.PathValue("roomID")
	if eventID == "" || roomID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or roomID")
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	room, err := c.Service.GetEventRoom(r.Context(), eventID, roomID, ownerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or room not found")
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
	helpers.WriteJSONSuccess(w, http.StatusOK, room)
}

// UpdateEventRoom godoc
// @Summary Update a room
// @Description Updates room details (capacity, description, how_to_get_there, not_bookable). Only the event owner can update. Optional fields omitted from body are unchanged (not_bookable keeps current value when omitted). Requires authentication.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param roomID path string true "Room ID (UUID)"
// @Param body body UpdateRoomRequest true "Room fields to update"
// @Success 200 {object} controllers.UpdateRoomSuccessResponse "data contains the updated room"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/rooms/{roomID} [patch]
func (c *ScheduleController) UpdateEventRoom(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	roomID := r.PathValue("roomID")
	if eventID == "" || roomID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or roomID")
		return
	}
	var req UpdateRoomRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	room, err := c.Service.UpdateEventRoom(r.Context(), eventID, roomID, ownerID, req.Capacity, req.Description, req.HowToGetThere, req.NotBookable)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or room not found")
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
	helpers.WriteJSONSuccess(w, http.StatusOK, room)
}

// DeleteEventRoom godoc
// @Summary Delete a room
// @Description Deletes a room and its sessions. Only the event owner can delete. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param roomID path string true "Room ID (UUID)"
// @Success 200 {object} controllers.DeleteRoomSuccessResponse "data contains status"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/rooms/{roomID} [delete]
func (c *ScheduleController) DeleteEventRoom(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	roomID := r.PathValue("roomID")
	if eventID == "" || roomID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or roomID")
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	if err := c.Service.DeleteEventRoom(r.Context(), eventID, roomID, ownerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or room not found")
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

// AddEventTeamMemberRequest is the request body for POST /events/{eventID}/team-members.
type AddEventTeamMemberRequest struct {
	Email string `json:"email"`
}

// Validate implements Validator.
func (a AddEventTeamMemberRequest) Validate() []string {
	var errs []string
	if a.Email == "" {
		errs = append(errs, "email is required")
	} else if !emailRegex.MatchString(strings.TrimSpace(a.Email)) {
		errs = append(errs, "email must be a valid email address")
	}
	return errs
}

// AddEventTeamMemberSuccessResponse is the success response envelope for POST /events/{eventID}/team-members (201).
type AddEventTeamMemberSuccessResponse struct {
	Data  *domain.EventTeamMember `json:"data"`
	Error *helpers.APIError       `json:"error"`
}

// AddEventTeamMember godoc
// @Summary Add a team member to an event
// @Description Add a user as a team member of the event by email. Only the event owner can add. Returns 404 with a message if no user exists with that email. Requires authentication.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param body body AddEventTeamMemberRequest true "Email of the user to add"
// @Success 201 {object} controllers.AddEventTeamMemberSuccessResponse "data contains the added team member"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found (no user with that email)"
// @Failure 409 {object} helpers.APIResponse "error.code: conflict (already member or invalid)"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/team-members [post]
func (c *ScheduleController) AddEventTeamMember(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	var req AddEventTeamMemberRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	member, err := c.Service.AddEventTeamMemberByEmail(r.Context(), eventID, req.Email, ownerID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "no user with that email")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			helpers.WriteJSONError(w, http.StatusForbidden, helpers.ErrCodeForbidden, "forbidden")
			return
		}
		if errors.Is(err, domain.ErrAlreadyMember) || errors.Is(err, domain.ErrInvalidInput) {
			helpers.WriteJSONError(w, http.StatusConflict, helpers.ErrCodeConflict, err.Error())
			return
		}
		c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)
		helpers.WriteJSONError(w, http.StatusInternalServerError, helpers.ErrCodeInternalError, err.Error())
		return
	}
	helpers.WriteJSONSuccess(w, http.StatusCreated, member)
}

// ListEventTeamMembersSuccessResponse is the success response envelope for GET /events/{eventID}/team-members (200).
type ListEventTeamMembersSuccessResponse struct {
	Data  []*domain.EventTeamMember `json:"data"`
	Error *helpers.APIError         `json:"error"`
}

// ListEventTeamMembers godoc
// @Summary List team members of an event
// @Description Returns the list of team members for the event. Only the event owner can list. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Success 200 {object} controllers.ListEventTeamMembersSuccessResponse "data is an array of team members"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/team-members [get]
func (c *ScheduleController) ListEventTeamMembers(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	callerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	members, err := c.Service.ListEventTeamMembers(r.Context(), eventID, callerID)
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
	if members == nil {
		members = []*domain.EventTeamMember{}
	}
	helpers.WriteJSONSuccess(w, http.StatusOK, members)
}

// RemoveEventTeamMemberResponse is the data payload for DELETE /events/{eventID}/team-members/{userID} (200).
type RemoveEventTeamMemberResponse struct {
	Status string `json:"status"`
}

// RemoveEventTeamMemberSuccessResponse is the success response envelope for DELETE /events/{eventID}/team-members/{userID} (200).
type RemoveEventTeamMemberSuccessResponse struct {
	Data  RemoveEventTeamMemberResponse `json:"data"`
	Error *helpers.APIError             `json:"error"`
}

// RemoveEventTeamMember godoc
// @Summary Remove a team member from an event
// @Description Remove a user from the event's team members. Only the event owner can remove. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param userID path string true "User ID (UUID) of the team member to remove"
// @Success 200 {object} controllers.RemoveEventTeamMemberSuccessResponse "data contains status"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/team-members/{userID} [delete]
func (c *ScheduleController) RemoveEventTeamMember(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	userID := r.PathValue("userID")
	if eventID == "" || userID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or userID")
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	err := c.Service.RemoveEventTeamMember(r.Context(), eventID, userID, ownerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or team member not found")
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
	helpers.WriteJSONSuccess(w, http.StatusOK, RemoveEventTeamMemberResponse{Status: "removed"})
}

// ListEventInvitationsResponse is the data payload for GET /events/{eventID}/invitations (200).
type ListEventInvitationsResponse struct {
	Items      []*domain.EventInvitation `json:"items"`
	Pagination helpers.PaginationMeta    `json:"pagination"`
}

// ListEventInvitationsSuccessResponse is the success response envelope for GET /events/{eventID}/invitations (200).
type ListEventInvitationsSuccessResponse struct {
	Data  ListEventInvitationsResponse `json:"data"`
	Error *helpers.APIError            `json:"error"`
}

// ListEventInvitations godoc
// @Summary List invited emails for an event
// @Description Returns a paginated list of emails invited to the event (with id and sent_at). Only the event owner can list. Use page and page_size query params. Optional search filters by email substring (case-insensitive). Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param search query string false "Filter emails containing this string (case-insensitive)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size (default 20, max 100)"
// @Success 200 {object} controllers.ListEventInvitationsSuccessResponse "data contains items and pagination"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/invitations [get]
func (c *ScheduleController) ListEventInvitations(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	callerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	params := helpers.ParsePagination(r)
	list, total, err := c.Service.ListEventInvitations(r.Context(), eventID, callerID, search, params)
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
	if list == nil {
		list = []*domain.EventInvitation{}
	}
	meta := helpers.NewPaginationMeta(params.Page, params.PageSize, total)
	helpers.WriteJSONSuccess(w, http.StatusOK, ListEventInvitationsResponse{Items: list, Pagination: meta})
}

// SendEventInvitationsRequest is the request body for POST /events/{eventID}/invitations.
// Emails is a long string of emails separated by commas or spaces.
type SendEventInvitationsRequest struct {
	Emails string `json:"emails"`
}

// Validate implements Validator.
func (s SendEventInvitationsRequest) Validate() []string {
	if strings.TrimSpace(s.Emails) == "" {
		return []string{"emails is required"}
	}
	return nil
}

// parseEmailsFromString splits the input by commas and spaces, trims, lowercases, deduplicates,
// and returns only strings that match emailRegex. May return an empty slice.
func parseEmailsFromString(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", " ")
	parts := strings.Fields(raw)
	seen := make(map[string]struct{})
	var out []string
	for _, p := range parts {
		email := strings.TrimSpace(strings.ToLower(p))
		if email == "" {
			continue
		}
		if !emailRegex.MatchString(email) {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	return out
}

// SendEventInvitationsResponse is the data payload for POST /events/{eventID}/invitations (200).
type SendEventInvitationsResponse struct {
	Sent  int      `json:"sent"`
	Failed []string `json:"failed"`
}

// SendEventInvitationsSuccessResponse is the success response envelope for POST /events/{eventID}/invitations (200).
type SendEventInvitationsSuccessResponse struct {
	Data  SendEventInvitationsResponse `json:"data"`
	Error *helpers.APIError            `json:"error"`
}

// UpdateSessionScheduleRequest is the request body for PATCH /events/{eventID}/sessions/{sessionID}.
// All fields are optional; omitted fields are unchanged.
type UpdateSessionScheduleRequest struct {
	RoomID    *string    `json:"room_id"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

// Validate implements Validator.
func (u UpdateSessionScheduleRequest) Validate() []string {
	var errs []string
	if u.RoomID != nil && strings.TrimSpace(*u.RoomID) == "" {
		errs = append(errs, "room_id cannot be empty")
	}
	return errs
}

// UpdateSessionScheduleSuccessResponse is the success response envelope for PATCH /events/{eventID}/sessions/{sessionID} (200).
type UpdateSessionScheduleSuccessResponse struct {
	Data  *domain.Session   `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// UpdateSessionSchedule godoc
// @Summary Update session schedule
// @Description Moves a session to a different room and/or time slot by updating room_id, start_time, and end_time. Only the event owner can update. Optional fields omitted from body are unchanged. Requires authentication.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param sessionID path string true "Session ID (UUID)"
// @Param body body UpdateSessionScheduleRequest true "Fields to update (all optional)"
// @Success 200 {object} controllers.UpdateSessionScheduleSuccessResponse "data contains the updated session"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/sessions/{sessionID} [patch]
func (c *ScheduleController) UpdateSessionSchedule(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionID := r.PathValue("sessionID")
	if eventID == "" || sessionID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or sessionID")
		return
	}

	var req UpdateSessionScheduleRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}

	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}

	session, err := c.Service.UpdateSessionSchedule(r.Context(), eventID, sessionID, ownerID, req.RoomID, req.StartTime, req.EndTime)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event, session, or room not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			helpers.WriteJSONError(w, http.StatusForbidden, helpers.ErrCodeForbidden, "forbidden")
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

	helpers.WriteJSONSuccess(w, http.StatusOK, session)
}

// UpdateSessionContentRequest is the request body for PATCH /events/{eventID}/sessions/{sessionID}/content.
// All fields are optional; omitted fields are unchanged.
type UpdateSessionContentRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

// Validate implements Validator.
func (u UpdateSessionContentRequest) Validate() []string {
	var errs []string
	if u.Title != nil && strings.TrimSpace(*u.Title) == "" {
		errs = append(errs, "title cannot be empty")
	}
	return errs
}

// UpdateSessionContentSuccessResponse is the success response envelope for PATCH /events/{eventID}/sessions/{sessionID}/content (200).
type UpdateSessionContentSuccessResponse struct {
	Data  *domain.Session   `json:"data"`
	Error *helpers.APIError `json:"error"`
}

// UpdateSessionContent godoc
// @Summary Update session content
// @Description Updates a session's title and/or description. Only the event owner can update. Optional fields omitted from body are unchanged. Requires authentication.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param sessionID path string true "Session ID (UUID)"
// @Param body body UpdateSessionContentRequest true "Fields to update (all optional)"
// @Success 200 {object} controllers.UpdateSessionContentSuccessResponse "data contains the updated session"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/sessions/{sessionID}/content [patch]
func (c *ScheduleController) UpdateSessionContent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionID := r.PathValue("sessionID")
	if eventID == "" || sessionID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or sessionID")
		return
	}

	var req UpdateSessionContentRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}

	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}

	session, err := c.Service.UpdateSessionContent(r.Context(), eventID, sessionID, ownerID, req.Title, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or session not found")
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

	helpers.WriteJSONSuccess(w, http.StatusOK, session)
}

// DeleteEventSession godoc
// @Summary Delete a session
// @Description Deletes a session. Only the event owner can delete. Requires authentication.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param sessionID path string true "Session ID (UUID)"
// @Success 200 {object} controllers.DeleteRoomSuccessResponse "data contains status"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/sessions/{sessionID} [delete]
func (c *ScheduleController) DeleteEventSession(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionID := r.PathValue("sessionID")
	if eventID == "" || sessionID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID or sessionID")
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	if err := c.Service.DeleteEventSession(r.Context(), eventID, sessionID, ownerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			helpers.WriteJSONError(w, http.StatusNotFound, helpers.ErrCodeNotFound, "event or session not found")
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

// SendEventInvitations godoc
// @Summary Send event invitation emails
// @Description Send invitation emails to register for the event. Body contains a string of emails separated by commas or spaces. Only the event owner can invite. Each invitation is persisted and emailed; duplicates for the same event are skipped. Returns count of sent and list of failed addresses.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventID path string true "Event ID (UUID)"
// @Param body body SendEventInvitationsRequest true "Emails string (comma or space separated)"
// @Success 200 {object} controllers.SendEventInvitationsSuccessResponse "data contains sent count and failed list"
// @Failure 400 {object} helpers.APIResponse "error.code: bad_request (empty or no valid emails)"
// @Failure 401 {object} helpers.APIResponse "error.code: unauthorized"
// @Failure 403 {object} helpers.APIResponse "error.code: forbidden (not owner)"
// @Failure 404 {object} helpers.APIResponse "error.code: not_found"
// @Failure 500 {object} helpers.APIResponse "error.code: internal_error"
// @Router /events/{eventID}/invitations [post]
func (c *ScheduleController) SendEventInvitations(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "missing eventID")
		return
	}
	var req SendEventInvitationsRequest
	if !helpers.DecodeAndValidate(w, r, &req) {
		return
	}
	emails := parseEmailsFromString(req.Emails)
	if len(emails) == 0 {
		helpers.WriteJSONError(w, http.StatusBadRequest, helpers.ErrCodeBadRequest, "no valid emails found")
		return
	}
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSONError(w, http.StatusUnauthorized, helpers.ErrCodeUnauthorized, "unauthorized")
		return
	}
	sent, failed, err := c.Service.SendEventInvitations(r.Context(), eventID, ownerID, emails)
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
	helpers.WriteJSONSuccess(w, http.StatusOK, SendEventInvitationsResponse{Sent: sent, Failed: failed})
}
