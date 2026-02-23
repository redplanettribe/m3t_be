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
	getEventByIDErr        error
	deleteEventErr         error
	toggleRoomErr          error
	toggleRoomResult       *domain.Room
	addTeamMemberErr         error
	addTeamMemberByEmailErr  error
	addTeamMemberByEmailResult *domain.EventTeamMember
	listTeamMembersErr       error
	listTeamMembersResult    []*domain.EventTeamMember
	removeTeamMemberErr      error
	lastCreateEvent          *domain.Event
	lastImportEventID        string
	lastImportSessionizeID   string
	lastDeleteEventID        string
	lastDeleteOwnerID        string
	lastAddTeamMemberEventID string
	lastAddTeamMemberEmail   string
	lastAddTeamMemberOwnerID string
	lastListTeamMembersEventID   string
	lastListTeamMembersCallerID  string
	lastRemoveTeamMemberEventID  string
	lastRemoveTeamMemberUserID   string
	lastRemoveTeamMemberOwnerID  string
	eventsByOwner          map[string][]*domain.Event // ownerID -> events to return
	eventByID              map[string]struct {       // eventID -> event, rooms, sessions to return
		event    *domain.Event
		rooms    []*domain.Room
		sessions []*domain.Session
	}
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

func (f *fakeManageScheduleService) GetEventByID(ctx context.Context, eventID string) (*domain.Event, []*domain.Room, []*domain.Session, error) {
	if f.getEventByIDErr != nil {
		return nil, nil, nil, f.getEventByIDErr
	}
	if f.eventByID != nil {
		if data, ok := f.eventByID[eventID]; ok {
			return data.event, data.rooms, data.sessions, nil
		}
	}
	return nil, nil, nil, domain.ErrNotFound
}

func (f *fakeManageScheduleService) DeleteEvent(ctx context.Context, eventID string, ownerID string) error {
	f.lastDeleteEventID = eventID
	f.lastDeleteOwnerID = ownerID
	return f.deleteEventErr
}

func (f *fakeManageScheduleService) ToggleRoomNotBookable(ctx context.Context, eventID, roomID, ownerID string) (*domain.Room, error) {
	if f.toggleRoomErr != nil {
		return nil, f.toggleRoomErr
	}
	return f.toggleRoomResult, nil
}

func (f *fakeManageScheduleService) AddEventTeamMember(ctx context.Context, eventID, userIDToAdd, ownerID string) error {
	f.lastAddTeamMemberEventID = eventID
	f.lastAddTeamMemberOwnerID = ownerID
	return f.addTeamMemberErr
}

func (f *fakeManageScheduleService) AddEventTeamMemberByEmail(ctx context.Context, eventID, email, ownerID string) (*domain.EventTeamMember, error) {
	f.lastAddTeamMemberEventID = eventID
	f.lastAddTeamMemberEmail = email
	f.lastAddTeamMemberOwnerID = ownerID
	if f.addTeamMemberByEmailErr != nil {
		return nil, f.addTeamMemberByEmailErr
	}
	if f.addTeamMemberByEmailResult != nil {
		return f.addTeamMemberByEmailResult, nil
	}
	return &domain.EventTeamMember{EventID: eventID, UserID: "resolved-user-id"}, nil
}

func (f *fakeManageScheduleService) ListEventTeamMembers(ctx context.Context, eventID, callerID string) ([]*domain.EventTeamMember, error) {
	f.lastListTeamMembersEventID = eventID
	f.lastListTeamMembersCallerID = callerID
	if f.listTeamMembersErr != nil {
		return nil, f.listTeamMembersErr
	}
	if f.listTeamMembersResult != nil {
		return f.listTeamMembersResult, nil
	}
	return []*domain.EventTeamMember{}, nil
}

func (f *fakeManageScheduleService) RemoveEventTeamMember(ctx context.Context, eventID, userIDToRemove, ownerID string) error {
	f.lastRemoveTeamMemberEventID = eventID
	f.lastRemoveTeamMemberUserID = userIDToRemove
	f.lastRemoveTeamMemberOwnerID = ownerID
	return f.removeTeamMemberErr
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

func TestScheduleController_GetEventByID(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		noUserContext  bool
		fakeErr        error
		eventByID      map[string]struct {
			event    *domain.Event
			rooms    []*domain.Room
			sessions []*domain.Session
		}
		wantStatus     int
		wantBodySubstr string
		checkResponse  func(t *testing.T, data GetEventByIDResponse)
	}{
		{
			name:    "success",
			eventID: "ev-123",
			eventByID: map[string]struct {
				event    *domain.Event
				rooms    []*domain.Room
				sessions []*domain.Session
			}{
				"ev-123": {
					event:    &domain.Event{ID: "ev-123", Name: "Conf 2025", Slug: "conf-2025", OwnerID: "user-1"},
					rooms:    []*domain.Room{{ID: "room-1", EventID: "ev-123", Name: "Room A"}},
					sessions: []*domain.Session{{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", Tags: []string{}}},
				},
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: "",
			checkResponse: func(t *testing.T, data GetEventByIDResponse) {
				require.NotNil(t, data.Event)
				assert.Equal(t, "ev-123", data.Event.ID)
				assert.Equal(t, "Conf 2025", data.Event.Name)
				require.Len(t, data.Rooms, 1)
				assert.Equal(t, "room-1", data.Rooms[0].ID)
				assert.Equal(t, "Room A", data.Rooms[0].Name)
				require.Len(t, data.Sessions, 1)
				assert.Equal(t, "sess-1", data.Sessions[0].ID)
				assert.Equal(t, "Talk 1", data.Sessions[0].Title)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
			checkResponse:  nil,
		},
		{
			name:          "no user in context",
			eventID:       "ev-123",
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
			checkResponse: nil,
		},
		{
			name:    "event not found",
			eventID: "ev-missing",
			eventByID: map[string]struct {
				event    *domain.Event
				rooms    []*domain.Room
				sessions []*domain.Session
			}{},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
			checkResponse:  nil,
		},
		{
			name:     "service error",
			eventID:  "ev-123",
			fakeErr:  errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{
				getEventByIDErr: tt.fakeErr,
				eventByID:       tt.eventByID,
			}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodGet, "http://test/events/"+tt.eventID, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.GetEventByID(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusOK && tt.checkResponse != nil {
				require.Nil(t, envelope.Error, "success response must have error nil")
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var data GetEventByIDResponse
				require.NoError(t, json.Unmarshal(dataBytes, &data))
				tt.checkResponse(t, data)
			}
			if tt.wantStatus != http.StatusOK && tt.wantBodySubstr != "" {
				require.NotNil(t, envelope.Error, "error response must have error set")
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr, "error message")
			}
		})
	}
}

func TestScheduleController_ToggleRoomNotBookable(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		roomID         string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Room
		wantStatus     int
		wantBodySubstr string
		checkResponse  func(t *testing.T, room *domain.Room)
	}{
		{
			name:       "success",
			eventID:    "ev-123",
			roomID:     "room-1",
			fakeResult: &domain.Room{ID: "room-1", EventID: "ev-123", Name: "Room A", NotBookable: true},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				assert.Equal(t, "room-1", room.ID)
				assert.True(t, room.NotBookable)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			roomID:         "room-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or roomID",
		},
		{
			name:           "missing roomID",
			eventID:        "ev-123",
			roomID:         "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or roomID",
		},
		{
			name:          "no user in context",
			eventID:       "ev-123",
			roomID:        "room-1",
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			roomID:         "room-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event or room not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-123",
			roomID:         "room-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-123",
			roomID:         "room-1",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{toggleRoomErr: tt.fakeErr, toggleRoomResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/rooms/" + tt.roomID + "/not-bookable"
			req := httptest.NewRequest(http.MethodPatch, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if tt.roomID != "" {
				req.SetPathValue("roomID", tt.roomID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ToggleRoomNotBookable(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusOK && tt.checkResponse != nil {
				require.Nil(t, envelope.Error, "success response must have error nil")
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var room domain.Room
				require.NoError(t, json.Unmarshal(dataBytes, &room))
				tt.checkResponse(t, &room)
			}
			if tt.wantStatus != http.StatusOK && tt.wantBodySubstr != "" {
				require.NotNil(t, envelope.Error, "error response must have error set")
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr, "error message")
			}
		})
	}
}

func TestScheduleController_DeleteEvent(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeManageScheduleService)
	}{
		{
			name:       "success",
			eventID:    "ev-123",
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeManageScheduleService) {
				assert.Equal(t, "ev-123", fake.lastDeleteEventID)
				assert.Equal(t, "user-123", fake.lastDeleteOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:          "no user in context",
			eventID:       "ev-123",
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden not owner",
			eventID:        "ev-123",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-123",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{deleteEventErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodDelete, "http://test/events/"+tt.eventID, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.DeleteEvent(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error, "success response must have error nil")
				tt.checkCall(t, fake)
				var data DeleteEventResponse
				dataBytes, _ := json.Marshal(envelope.Data)
				require.NoError(t, json.Unmarshal(dataBytes, &data))
				assert.Equal(t, "deleted", data.Status)
			}
			if tt.wantStatus != http.StatusOK && tt.wantBodySubstr != "" {
				require.NotNil(t, envelope.Error, "error response must have error set")
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr, "error message")
			}
		})
	}
}

func TestScheduleController_AddEventTeamMember(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		body           string
		fakeErr        error
		fakeResult     *domain.EventTeamMember
		wantStatus     int
		wantBodySubstr string
		noUserContext  bool
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			body:       `{"email":"teammate@example.com"}`,
			fakeResult: &domain.EventTeamMember{EventID: "ev-1", UserID: "user-456"},
			wantStatus: http.StatusCreated,
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{"email":"teammate@example.com"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "missing email",
			eventID:        "ev-1",
			body:           `{}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "email",
		},
		{
			name:           "invalid email format",
			eventID:        "ev-1",
			body:           `{"email":"not-an-email"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "email",
		},
		{
			name:          "no user in context",
			eventID:       "ev-1",
			body:          `{"email":"teammate@example.com"}`,
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:           "no user with that email",
			eventID:        "ev-1",
			body:           `{"email":"nobody@example.com"}`,
			fakeErr:        domain.ErrUserNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "no user with that email",
		},
		{
			name:           "event not found",
			eventID:        "ev-1",
			body:           `{"email":"teammate@example.com"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			body:           `{"email":"teammate@example.com"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "conflict already member",
			eventID:        "ev-1",
			body:           `{"email":"teammate@example.com"}`,
			fakeErr:        domain.ErrAlreadyMember,
			wantStatus:     http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{addTeamMemberByEmailErr: tt.fakeErr, addTeamMemberByEmailResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodPost, "http://test/events/"+tt.eventID+"/team-members", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.AddEventTeamMember(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusCreated {
				require.Nil(t, envelope.Error)
				assert.Equal(t, tt.eventID, fake.lastAddTeamMemberEventID)
				assert.Equal(t, "user-123", fake.lastAddTeamMemberOwnerID)
				if tt.body != "" {
					assert.Contains(t, fake.lastAddTeamMemberEmail, "teammate@example.com")
				}
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_ListEventTeamMembers(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		fakeErr        error
		fakeResult     []*domain.EventTeamMember
		wantStatus     int
		wantBodySubstr string
		noUserContext  bool
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			fakeResult: []*domain.EventTeamMember{{EventID: "ev-1", UserID: "user-a"}},
			wantStatus: http.StatusOK,
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:          "no user in context",
			eventID:       "ev-1",
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:           "event not found",
			eventID:        "ev-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{listTeamMembersErr: tt.fakeErr, listTeamMembersResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodGet, "http://test/events/"+tt.eventID+"/team-members", nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ListEventTeamMembers(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				assert.Equal(t, tt.eventID, fake.lastListTeamMembersEventID)
				assert.Equal(t, "user-123", fake.lastListTeamMembersCallerID)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_RemoveEventTeamMember(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		userID         string
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		noUserContext  bool
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			userID:     "user-2",
			wantStatus: http.StatusOK,
		},
		{
			name:           "missing eventID",
			eventID:        "",
			userID:         "user-2",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing",
		},
		{
			name:          "no user in context",
			eventID:       "ev-1",
			userID:        "user-2",
			noUserContext: true,
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:           "event not found",
			eventID:        "ev-1",
			userID:         "user-2",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			userID:         "user-2",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeManageScheduleService{removeTeamMemberErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/team-members/" + tt.userID
			req := httptest.NewRequest(http.MethodDelete, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if tt.userID != "" {
				req.SetPathValue("userID", tt.userID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.RemoveEventTeamMember(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				assert.Equal(t, tt.eventID, fake.lastRemoveTeamMemberEventID)
				assert.Equal(t, tt.userID, fake.lastRemoveTeamMemberUserID)
				assert.Equal(t, "user-123", fake.lastRemoveTeamMemberOwnerID)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}
