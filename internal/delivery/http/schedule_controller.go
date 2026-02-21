package http

import (
	"encoding/json"
	"multitrackticketing/internal/domain"
	"net/http"
)

// CreateEventRequest is the request body for POST /events. Only name and slug are accepted.
type CreateEventRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
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
// @Success 201 {object} domain.Event
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /events [post]
func (c *ScheduleController) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	event := &domain.Event{Name: req.Name, Slug: req.Slug}
	if err := c.Service.CreateEvent(r.Context(), event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// ImportSessionize godoc
// @Summary Import schedule from Sessionize
// @Description Import rooms and sessions from Sessionize for a specific event
// @Tags events
// @Param eventID path string true "Event ID"
// @Param sessionizeID path string true "Sessionize ID"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /events/{eventID}/import/sessionize/{sessionizeID} [post]
func (c *ScheduleController) ImportSessionize(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	sessionizeID := r.PathValue("sessionizeID")

	if eventID == "" || sessionizeID == "" {
		http.Error(w, "missing eventID or sessionizeID", http.StatusBadRequest)
		return
	}

	if err := c.Service.ImportSessionizeData(r.Context(), eventID, sessionizeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "imported successfully"})
}
