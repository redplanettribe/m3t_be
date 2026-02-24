package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"
	"multitrackticketing/internal/domain"
)

type mockAttendeeService struct {
	registrations []*domain.EventRegistrationWithEvent
	err           error
}

func (m *mockAttendeeService) RegisterForEvent(ctx context.Context, eventID, userID string) (*domain.EventRegistration, error) {
	return nil, nil
}

func (m *mockAttendeeService) ListMyRegisteredEvents(ctx context.Context, userID string) ([]*domain.EventRegistrationWithEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.registrations, nil
}

func TestAttendeeController_ListMyRegisteredEvents_Unauthorized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := &mockAttendeeService{}
	ctrl := NewAttendeeController(logger, svc)

	req := httptest.NewRequest(http.MethodGet, "/attendee/events", nil)
	w := httptest.NewRecorder()

	ctrl.ListMyRegisteredEvents(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAttendeeController_ListMyRegisteredEvents_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := &domain.EventRegistration{ID: "r1", EventID: "e1", UserID: "u1"}
	ev := &domain.Event{ID: "e1", Name: "Event 1"}
	svc := &mockAttendeeService{
		registrations: []*domain.EventRegistrationWithEvent{
			{Registration: reg, Event: ev},
		},
	}
	ctrl := NewAttendeeController(logger, svc)

	req := httptest.NewRequest(http.MethodGet, "/attendee/events", nil)
	ctx := middleware.SetUserID(req.Context(), "u1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	ctrl.ListMyRegisteredEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp helpers.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}
}

func TestAttendeeController_ListMyRegisteredEvents_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := &mockAttendeeService{
		err: errors.New("service error"),
	}
	ctrl := NewAttendeeController(logger, svc)

	req := httptest.NewRequest(http.MethodGet, "/attendee/events", nil)
	ctx := middleware.SetUserID(req.Context(), "u1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	ctrl.ListMyRegisteredEvents(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

