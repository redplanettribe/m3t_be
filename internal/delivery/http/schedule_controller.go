package http

import (
	"multitrackticketing/internal/domain"
	"net/http"
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
	Service domain.ManageScheduleService
}

func NewScheduleController(svc domain.ManageScheduleService) *ScheduleController {
	return &ScheduleController{
		Service: svc,
	}
}

// CreateEvent godoc
// @Summary Create a new event
// @Description Create a new conference event. Only name and slug are accepted in the body; id and timestamps are server-generated.
// @Tags events
// @Accept json
// @Produce json
// @Param event body CreateEventRequest true "Event data (name and slug only)"
// @Success 201 {object} APIResponse "data contains the created event"
// @Failure 400 {object} APIResponse "error.code: bad_request"
// @Failure 500 {object} APIResponse "error.code: internal_error"
// @Router /events [post]
func (c *ScheduleController) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}
	event := &domain.Event{Name: req.Name, Slug: req.Slug}
	if err := c.Service.CreateEvent(r.Context(), event); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	WriteJSONSuccess(w, http.StatusCreated, event)
}

// ImportSessionize godoc
// @Summary Import schedule from Sessionize
// @Description Import rooms and sessions from Sessionize for a specific event
// @Tags events
// @Param eventID path string true "Event ID"
// @Param sessionizeID path string true "Sessionize ID"
// @Success 200 {object} APIResponse "data contains status message"
// @Failure 400 {object} APIResponse "error.code: bad_request"
// @Failure 500 {object} APIResponse "error.code: internal_error"
// @Router /events/{eventID}/import/sessionize/{sessionizeID} [post]
func (c *ScheduleController) ImportSessionize(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionizeID := r.PathValue("sessionizeID")

	if eventID == "" || sessionizeID == "" {
		WriteJSONError(w, http.StatusBadRequest, ErrCodeBadRequest, "missing eventID or sessionizeID")
		return
	}

	if err := c.Service.ImportSessionizeData(r.Context(), eventID, sessionizeID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	WriteJSONSuccess(w, http.StatusOK, map[string]string{"status": "imported successfully"})
}
