package controllers

import (
	"bytes"
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
	registrations        []*domain.EventRegistrationWithEvent
	err                  error
	registerByCodeReg   *domain.EventRegistration
	registerByCodeErr   error
	registerByCodeCreated bool
}

func (m *mockAttendeeService) RegisterForEvent(ctx context.Context, eventID, userID string) (*domain.EventRegistration, bool, error) {
	return nil, false, nil
}

func (m *mockAttendeeService) RegisterForEventByCode(ctx context.Context, eventCode, userID string) (*domain.EventRegistration, bool, error) {
	if m.registerByCodeErr != nil {
		return nil, false, m.registerByCodeErr
	}
	return m.registerByCodeReg, m.registerByCodeCreated, nil
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

func TestAttendeeController_RegisterForEventByCode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		body           string
		setUserID      bool
		svc            *mockAttendeeService
		wantStatus     int
		wantErrCode    string
		wantDataHasID  bool
	}{
		{
			name:       "success",
			body:       `{"event_code":"abc1"}`,
			setUserID:  true,
			svc:        &mockAttendeeService{registerByCodeReg: &domain.EventRegistration{ID: "r1", EventID: "e1", UserID: "u1"}, registerByCodeCreated: true},
			wantStatus: http.StatusCreated,
			wantDataHasID: true,
		},
		{
			name:       "already registered returns 200",
			body:       `{"event_code":"abc1"}`,
			setUserID:  true,
			svc:        &mockAttendeeService{registerByCodeReg: &domain.EventRegistration{ID: "r1", EventID: "e1", UserID: "u1"}, registerByCodeCreated: false},
			wantStatus: http.StatusOK,
			wantDataHasID: true,
		},
		{
			name:       "unauthorized",
			body:       `{"event_code":"abc1"}`,
			setUserID:  false,
			svc:        &mockAttendeeService{},
			wantStatus: http.StatusUnauthorized,
			wantErrCode: helpers.ErrCodeUnauthorized,
		},
		{
			name:       "not found",
			body:       `{"event_code":"none"}`,
			setUserID:  true,
			svc:        &mockAttendeeService{registerByCodeErr: domain.ErrNotFound},
			wantStatus: http.StatusNotFound,
			wantErrCode: helpers.ErrCodeNotFound,
		},
		{
			name:       "validation missing event_code",
			body:       `{}`,
			setUserID:  true,
			svc:        &mockAttendeeService{},
			wantStatus: http.StatusBadRequest,
			wantErrCode: helpers.ErrCodeBadRequest,
		},
		{
			name:       "validation invalid event_code length",
			body:       `{"event_code":"ab"}`,
			setUserID:  true,
			svc:        &mockAttendeeService{},
			wantStatus: http.StatusBadRequest,
			wantErrCode: helpers.ErrCodeBadRequest,
		},
		{
			name:       "validation invalid event_code characters",
			body:       `{"event_code":"ab@d"}`,
			setUserID:  true,
			svc:        &mockAttendeeService{},
			wantStatus: http.StatusBadRequest,
			wantErrCode: helpers.ErrCodeBadRequest,
		},
		{
			name:       "service error",
			body:       `{"event_code":"abc1"}`,
			setUserID:  true,
			svc:        &mockAttendeeService{registerByCodeErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
			wantErrCode: helpers.ErrCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewAttendeeController(logger, tt.svc)
			req := httptest.NewRequest(http.MethodPost, "/attendee/registrations", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			if tt.setUserID {
				req = req.WithContext(middleware.SetUserID(req.Context(), "u1"))
			}
			w := httptest.NewRecorder()

			ctrl.RegisterForEventByCode(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: want %d, got %d", tt.wantStatus, w.Code)
			}
			var resp helpers.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if tt.wantErrCode != "" && (resp.Error == nil || resp.Error.Code != tt.wantErrCode) {
				t.Errorf("error code: want %q, got %v", tt.wantErrCode, resp.Error)
			}
			if tt.wantDataHasID && resp.Data != nil {
				dataMap, ok := resp.Data.(map[string]interface{})
				if !ok {
					return
				}
				if _, ok := dataMap["id"]; !ok {
					t.Error("expected data to contain id")
				}
			}
		})
	}
}

