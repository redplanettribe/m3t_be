package http

import (
	"encoding/json"
	"multitrackticketing/internal/domain"
	"net/http"
)

type ScheduleController struct {
	UseCase domain.ManageScheduleUseCase
}

func NewScheduleController(uc domain.ManageScheduleUseCase) *ScheduleController {
	return &ScheduleController{
		UseCase: uc,
	}
}

// CreateSession godoc
// @Summary Create a new session
// @Description Create a new conference session
// @Tags sessions
// @Accept json
// @Produce json
// @Param session body domain.Session true "Session Data"
// @Success 201 {object} domain.Session
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /sessions [post]
func (c *ScheduleController) CreateSession(w http.ResponseWriter, r *http.Request) {
	var session domain.Session
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.UseCase.CreateSession(r.Context(), &session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}
