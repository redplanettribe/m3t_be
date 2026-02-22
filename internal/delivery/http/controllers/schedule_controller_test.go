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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger is a no-op logger for controller tests so we don't assert on log output.
var testLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

// fakeManageScheduleService implements domain.ManageScheduleService for handler tests.
type fakeManageScheduleService struct {
	createEventErr         error
	importSessionizeErr    error
	listEventsByOwnerErr   error
	lastCreateEvent        *domain.Event
	lastImportEventID      string
	lastImportSessionizeID string
	eventsByOwner          map[string][]*domain.Event // ownerID -> events to return
}

func (f *fakeManageScheduleService) CreateEvent(ctx context.Context, event *domain.Event) error {
	f.lastCreateEvent = event
	if f.createEventErr != nil {
		return f.createEventErr
	}
	event.ID = "ev-created"
	return nil
}

func (f *fakeManageScheduleService) ImportSessionizeData(ctx context.Context, eventID, sessionizeID string) error {
	f.lastImportEventID = eventID
	f.lastImportSessionizeID = sessionizeID
	return f.importSessionizeErr
}

func (f *fakeManageScheduleService) ListEventsByOwner(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	if f.listEventsByOwnerErr != nil {
		return nil, f.listEventsByOwnerErr
	}
	if f.eventsByOwner != nil {
		if events, ok := f.eventsByOwner[ownerID]; ok {
			return events, nil
		}
	}
	return []*domain.Event{}, nil
}

func TestScheduleController_CreateEvent(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		decodeEvent    bool
		checkEvent     func(t *testing.T, event domain.Event)
		noUserContext  bool // if true, do not set user ID in context (expect 401)
	}{
		{
			name:           "success",
			body:           `{"name":"Conf 2025","slug":"conf-2025"}`,
			wantStatus:     http.StatusCreated,
			wantBodySubstr: "",
			decodeEvent:    true,
			checkEvent: func(t *testing.T, event domain.Event) {
				assert.Equal(t, "ev-created", event.ID)
				assert.Equal(t, "Conf 2025", event.Name)
				assert.Equal(t, "conf-2025", event.Slug)
				assert.Equal(t, "user-123", event.OwnerID)
			},
		},
		{
			name:           "no user in context",
			body:           `{"name":"Conf 2025","slug":"conf-2025"}`,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
			decodeEvent:    false,
			checkEvent:     nil,
			noUserContext:  true,
		},
		{
			name:           "bad request invalid json",
			body:           `{invalid`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "invalid",
			decodeEvent:    false,
			checkEvent:     nil,
			noUserContext:  true, // decode fails before we check context
		},
		{
			name:           "missing name",
			body:           `{"slug":"conf-2025"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "name is required",
			decodeEvent:    false,
			checkEvent:     nil,
		},
		{
			name:           "missing slug",
			body:           `{"name":"Conf 2025"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "slug is required",
			decodeEvent:    false,
			checkEvent:     nil,
		},
		{
			name:           "unknown field rejected",
			body:           `{"name":"Conf","slug":"conf","id":"custom-id"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "unknown field",
			decodeEvent:    false,
			checkEvent:     nil,
		},
		{
			name:           "service error",
			body:           `{"name":"Conf","slug":"conf"}`,
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
			decodeEvent:    false,
			checkEvent:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{createEventErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()

			ctrl.CreateEvent(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusCreated && tt.decodeEvent {
				require.Nil(t, envelope.Error, "success response must have error nil")
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var event domain.Event
				require.NoError(t, json.Unmarshal(dataBytes, &event))
				tt.checkEvent(t, event)
			}
			if tt.wantStatus != http.StatusCreated && tt.wantBodySubstr != "" {
				require.NotNil(t, envelope.Error, "error response must have error set")
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr, "error message")
			}
		})
	}
}

func TestScheduleController_ImportSessionize(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		wantStatusJSON string
	}{
		{
			name:           "success",
			path:           "/events/ev-1/import/sessionize/abc123",
			wantStatus:     http.StatusOK,
			wantBodySubstr: "imported successfully",
			wantStatusJSON: "imported successfully",
		},
		{
			name:           "missing eventID",
			path:           "/events//import/sessionize/abc",
			fakeErr:        nil,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionizeID",
			wantStatusJSON: "",
		},
		{
			name:           "missing sessionizeID",
			path:           "/events/ev-1/import/sessionize/",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionizeID",
			wantStatusJSON: "",
		},
		{
			name:           "service error",
			path:           "/events/ev-1/import/sessionize/xyz",
			fakeErr:        errors.New("import failed"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "import failed",
			wantStatusJSON: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{importSessionizeErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodPost, "http://test"+tt.path, nil)
			req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			// Set path params for direct handler call (router would set these in production).
			switch tt.name {
			case "success":
				req.SetPathValue("eventID", "ev-1")
				req.SetPathValue("sessionizeID", "abc123")
			case "missing eventID":
				req.SetPathValue("eventID", "")
				req.SetPathValue("sessionizeID", "abc")
			case "missing sessionizeID":
				req.SetPathValue("eventID", "ev-1")
				req.SetPathValue("sessionizeID", "")
			case "service error":
				req.SetPathValue("eventID", "ev-1")
				req.SetPathValue("sessionizeID", "xyz")
			}
			rr := httptest.NewRecorder()
			ctrl.ImportSessionize(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error, "success response must have error nil")
				if tt.wantStatusJSON != "" {
					dataMap, ok := envelope.Data.(map[string]interface{})
					require.True(t, ok, "data must be object")
					assert.Equal(t, tt.wantStatusJSON, dataMap["status"], "data.status")
				}
			} else {
				require.NotNil(t, envelope.Error, "error response must have error set")
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr, "error message")
			}
		})
	}
}

func TestScheduleController_ListMyEvents(t *testing.T) {
	tests := []struct {
		name           string
		noUserContext  bool
		fakeErr        error
		eventsByOwner  map[string][]*domain.Event
		wantStatus     int
		wantBodySubstr string
		checkEvents    func(t *testing.T, events []domain.Event)
	}{
		{
			name: "success with events",
			eventsByOwner: map[string][]*domain.Event{
				"user-123": {
					{ID: "ev-1", Name: "Conf A", Slug: "conf-a", OwnerID: "user-123"},
					{ID: "ev-2", Name: "Conf B", Slug: "conf-b", OwnerID: "user-123"},
				},
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: "",
			checkEvents: func(t *testing.T, events []domain.Event) {
				require.Len(t, events, 2)
				assert.Equal(t, "ev-1", events[0].ID)
				assert.Equal(t, "Conf A", events[0].Name)
				assert.Equal(t, "user-123", events[0].OwnerID)
			},
		},
		{
			name:           "success empty",
			eventsByOwner:  map[string][]*domain.Event{"user-123": {}},
			wantStatus:     http.StatusOK,
			wantBodySubstr: "",
			checkEvents: func(t *testing.T, events []domain.Event) {
				require.Len(t, events, 0)
			},
		},
		{
			name:          "no user in context",
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
			checkEvents:   nil,
		},
		{
			name:          "service error",
			fakeErr:       errors.New("db error"),
			wantStatus:    http.StatusInternalServerError,
			wantBodySubstr: "db error",
			checkEvents:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{
				listEventsByOwnerErr: tt.fakeErr,
				eventsByOwner:        tt.eventsByOwner,
			}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodGet, "/events/me", nil)
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ListMyEvents(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusOK && tt.checkEvents != nil {
				require.Nil(t, envelope.Error, "success response must have error nil")
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var events []domain.Event
				require.NoError(t, json.Unmarshal(dataBytes, &events))
				tt.checkEvents(t, events)
			}
			if tt.wantStatus != http.StatusOK && tt.wantBodySubstr != "" {
				require.NotNil(t, envelope.Error, "error response must have error set")
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr, "error message")
			}
		})
	}
}
