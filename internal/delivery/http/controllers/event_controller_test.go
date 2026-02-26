package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"
	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger is a no-op logger for controller tests so we don't assert on log output.
var testLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

// fakeEventService implements domain.EventService for handler tests.
type fakeEventService struct {
	createEventErr              error
	importSessionizeErr         error
	listEventsByOwnerErr        error
	getEventByIDErr             error
	deleteEventErr              error
	toggleRoomErr               error
	toggleRoomResult            *domain.Room
	addTeamMemberErr            error
	addTeamMemberByEmailErr     error
	addTeamMemberByEmailResult  *domain.EventTeamMember
	listTeamMembersErr          error
	listTeamMembersResult       []*domain.EventTeamMember
	removeTeamMemberErr         error
	lastCreateEvent             *domain.Event
	lastImportEventID           string
	lastImportSessionizeID      string
	lastDeleteEventID           string
	lastDeleteOwnerID           string
	lastAddTeamMemberEventID    string
	lastAddTeamMemberEmail      string
	lastAddTeamMemberOwnerID    string
	lastListTeamMembersEventID  string
	lastListTeamMembersCallerID string
	lastRemoveTeamMemberEventID string
	lastRemoveTeamMemberUserID  string
	lastRemoveTeamMemberOwnerID string
	eventsByOwner               map[string][]*domain.Event // ownerID -> events to return
	eventByID                   map[string]struct {        // eventID -> event, rooms, sessions to return
		event    *domain.Event
		rooms    []*domain.Room
		sessions []*domain.Session
	}
	// SendEventInvitations
	sendEventInvitationsErr    error
	sendEventInvitationsSent   int
	sendEventInvitationsFailed []string
	lastSendInvitationsEventID string
	lastSendInvitationsOwnerID string
	lastSendInvitationsEmails  []string
	// ListEventInvitations
	listEventInvitationsErr     error
	listEventInvitationsResult  []*domain.EventInvitation
	listEventInvitationsTotal   int
	lastListInvitationsEventID  string
	lastListInvitationsCallerID string
	lastListInvitationsSearch   string
	lastListInvitationsParams   domain.PaginationParams
	// Room CRUD
	listEventRoomsErr          error
	listEventRoomsResult       []*domain.Room
	getEventRoomErr            error
	getEventRoomResult         *domain.Room
	updateEventRoomErr         error
	updateEventRoomResult      *domain.Room
	deleteEventRoomErr         error
	lastListEventRoomsEventID  string
	lastListEventRoomsOwnerID  string
	lastGetEventRoomEventID    string
	lastGetEventRoomRoomID     string
	lastGetEventRoomOwnerID    string
	lastUpdateEventRoomEventID string
	lastUpdateEventRoomRoomID  string
	lastUpdateEventRoomOwnerID string
	lastUpdateEventRoomName    *string
	lastDeleteEventRoomEventID string
	lastDeleteEventRoomRoomID  string
	lastDeleteEventRoomOwnerID string
	// DeleteEventSession
	deleteEventSessionErr           error
	lastDeleteEventSessionEventID   string
	lastDeleteEventSessionSessionID string
	lastDeleteEventSessionOwnerID   string
	// UpdateSessionContent
	updateSessionContentErr           error
	updateSessionContentResult        *domain.Session
	lastUpdateSessionContentEventID   string
	lastUpdateSessionContentSessionID string
	lastUpdateSessionContentOwnerID   string
	lastUpdateSessionContentTitle     *string
	lastUpdateSessionContentDesc      *string
	// UpdateEvent
	updateEventErr         error
	updateEventResult      *domain.Event
	lastUpdateEventID      string
	lastUpdateEventOwnerID string
	// Speakers
	listEventSpeakersErr            error
	listEventSpeakersResult         []*domain.Speaker
	getEventSpeakerErr              error
	getEventSpeakerResult           *domain.Speaker
	getEventSpeakerSessions         []*domain.Session
	deleteEventSpeakerErr           error
	createEventSpeakerErr           error
	createEventSpeakerResult        *domain.Speaker
	lastListEventSpeakersEventID    string
	lastListEventSpeakersOwnerID    string
	lastGetEventSpeakerEventID      string
	lastGetEventSpeakerSpeakerID    string
	lastGetEventSpeakerOwnerID      string
	lastDeleteEventSpeakerEventID   string
	lastDeleteEventSpeakerSpeakerID string
	lastDeleteEventSpeakerOwnerID   string
	lastCreateEventSpeakerEventID   string
	lastCreateEventSpeakerOwnerID   string
	lastCreateEventSpeakerFirstName string
	lastCreateEventSpeakerLastName  string
	// CreateEventRoom
	createEventRoomErr          error
	createEventRoomResult       *domain.Room
	lastCreateEventRoomEventID  string
	lastCreateEventRoomOwnerID  string
	lastCreateEventRoomName     string
	lastCreateEventRoomCapacity int
	// ListEventTags
	listEventTagsErr          error
	listEventTagsResult       []*domain.Tag
	lastListEventTagsEventID  string
	lastListEventTagsCallerID string
	// AddEventTags
	addEventTagsErr          error
	addEventTagsResult       []*domain.Tag
	lastAddEventTagsEventID  string
	lastAddEventTagsOwnerID  string
	lastAddEventTagsTagNames []string
	// UpdateEventTag
	updateEventTagErr          error
	updateEventTagResult       *domain.Tag
	lastUpdateEventTagEventID  string
	lastUpdateEventTagTagID    string
	lastUpdateEventTagOwnerID  string
	lastUpdateEventTagName     string
	// AddSessionTag
	addSessionTagErr          error
	lastAddSessionTagEventID   string
	lastAddSessionTagSessionID string
	lastAddSessionTagOwnerID   string
	lastAddSessionTagTagID     string
	// RemoveSessionTag
	removeSessionTagErr          error
	lastRemoveSessionTagEventID   string
	lastRemoveSessionTagSessionID string
	lastRemoveSessionTagOwnerID   string
	lastRemoveSessionTagTagID    string
	// RemoveEventTag
	removeEventTagErr         error
	lastRemoveEventTagEventID string
	lastRemoveEventTagOwnerID string
	lastRemoveEventTagTagID   string
	// CreateEventSession
	createEventSessionErr          error
	createEventSessionResult       *domain.Session
	lastCreateEventSessionEventID  string
	lastCreateEventSessionOwnerID  string
	lastCreateEventSessionRoomID   string
	lastCreateEventSessionTitle    string
	lastCreateEventSessionDesc     string
	lastCreateEventSessionStart    time.Time
	lastCreateEventSessionEnd      time.Time
	lastCreateEventSessionTags     []string
	lastCreateEventSessionSpeakers []string
}

func (f *fakeEventService) CreateEvent(ctx context.Context, event *domain.Event) error {
	f.lastCreateEvent = event
	if f.createEventErr != nil {
		return f.createEventErr
	}
	event.ID = "ev-created"
	return nil
}

func (f *fakeEventService) ImportSessionizeData(ctx context.Context, eventID, sessionizeID string) error {
	f.lastImportEventID = eventID
	f.lastImportSessionizeID = sessionizeID
	return f.importSessionizeErr
}

func (f *fakeEventService) ListEventsByOwner(ctx context.Context, ownerID string) ([]*domain.Event, error) {
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

func (f *fakeEventService) GetEventByID(ctx context.Context, eventID string) (*domain.Event, []*domain.Room, []*domain.Session, error) {
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

func (f *fakeEventService) DeleteEvent(ctx context.Context, eventID string, ownerID string) error {
	f.lastDeleteEventID = eventID
	f.lastDeleteOwnerID = ownerID
	return f.deleteEventErr
}

func (f *fakeEventService) UpdateEvent(ctx context.Context, eventID, ownerID string, date *time.Time, description *string, locationLat, locationLng *float64) (*domain.Event, error) {
	f.lastUpdateEventID = eventID
	f.lastUpdateEventOwnerID = ownerID
	if f.updateEventErr != nil {
		return nil, f.updateEventErr
	}
	return f.updateEventResult, nil
}

func (f *fakeEventService) ToggleRoomNotBookable(ctx context.Context, eventID, roomID, ownerID string) (*domain.Room, error) {
	if f.toggleRoomErr != nil {
		return nil, f.toggleRoomErr
	}
	return f.toggleRoomResult, nil
}

func (f *fakeEventService) ListEventRooms(ctx context.Context, eventID, ownerID string) ([]*domain.Room, error) {
	f.lastListEventRoomsEventID = eventID
	f.lastListEventRoomsOwnerID = ownerID
	if f.listEventRoomsErr != nil {
		return nil, f.listEventRoomsErr
	}
	if f.listEventRoomsResult != nil {
		return f.listEventRoomsResult, nil
	}
	return []*domain.Room{}, nil
}

func (f *fakeEventService) GetEventRoom(ctx context.Context, eventID, roomID, ownerID string) (*domain.Room, error) {
	f.lastGetEventRoomEventID = eventID
	f.lastGetEventRoomRoomID = roomID
	f.lastGetEventRoomOwnerID = ownerID
	if f.getEventRoomErr != nil {
		return nil, f.getEventRoomErr
	}
	return f.getEventRoomResult, nil
}

func (f *fakeEventService) UpdateEventRoom(ctx context.Context, eventID, roomID, ownerID string, name *string, capacity int, description, howToGetThere string, notBookable *bool) (*domain.Room, error) {
	f.lastUpdateEventRoomEventID = eventID
	f.lastUpdateEventRoomRoomID = roomID
	f.lastUpdateEventRoomOwnerID = ownerID
	f.lastUpdateEventRoomName = name
	if f.updateEventRoomErr != nil {
		return nil, f.updateEventRoomErr
	}
	return f.updateEventRoomResult, nil
}

func (f *fakeEventService) DeleteEventRoom(ctx context.Context, eventID, roomID, ownerID string) error {
	f.lastDeleteEventRoomEventID = eventID
	f.lastDeleteEventRoomRoomID = roomID
	f.lastDeleteEventRoomOwnerID = ownerID
	return f.deleteEventRoomErr
}

func (f *fakeEventService) DeleteEventSession(ctx context.Context, eventID, sessionID, ownerID string) error {
	f.lastDeleteEventSessionEventID = eventID
	f.lastDeleteEventSessionSessionID = sessionID
	f.lastDeleteEventSessionOwnerID = ownerID
	return f.deleteEventSessionErr
}

func (f *fakeEventService) AddEventTeamMember(ctx context.Context, eventID, userIDToAdd, ownerID string) error {
	f.lastAddTeamMemberEventID = eventID
	f.lastAddTeamMemberOwnerID = ownerID
	return f.addTeamMemberErr
}

func (f *fakeEventService) AddEventTeamMemberByEmail(ctx context.Context, eventID, email, ownerID string) (*domain.EventTeamMember, error) {
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

func (f *fakeEventService) ListEventTeamMembers(ctx context.Context, eventID, callerID string) ([]*domain.EventTeamMember, error) {
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

func (f *fakeEventService) RemoveEventTeamMember(ctx context.Context, eventID, userIDToRemove, ownerID string) error {
	f.lastRemoveTeamMemberEventID = eventID
	f.lastRemoveTeamMemberUserID = userIDToRemove
	f.lastRemoveTeamMemberOwnerID = ownerID
	return f.removeTeamMemberErr
}

func (f *fakeEventService) UpdateSessionSchedule(ctx context.Context, eventID, sessionID, ownerID string, roomID *string, startTime, endTime *time.Time) (*domain.Session, error) {
	return nil, nil
}

func (f *fakeEventService) UpdateSessionContent(ctx context.Context, eventID, sessionID, ownerID string, title *string, description *string) (*domain.Session, error) {
	f.lastUpdateSessionContentEventID = eventID
	f.lastUpdateSessionContentSessionID = sessionID
	f.lastUpdateSessionContentOwnerID = ownerID
	f.lastUpdateSessionContentTitle = title
	f.lastUpdateSessionContentDesc = description
	if f.updateSessionContentErr != nil {
		return nil, f.updateSessionContentErr
	}
	return f.updateSessionContentResult, nil
}

func (f *fakeEventService) SendEventInvitations(ctx context.Context, eventID, ownerID string, emails []string) (sent int, failed []string, err error) {
	f.lastSendInvitationsEventID = eventID
	f.lastSendInvitationsOwnerID = ownerID
	f.lastSendInvitationsEmails = emails
	if f.sendEventInvitationsErr != nil {
		return 0, nil, f.sendEventInvitationsErr
	}
	return f.sendEventInvitationsSent, f.sendEventInvitationsFailed, nil
}

func (f *fakeEventService) ListEventInvitations(ctx context.Context, eventID, callerID string, search string, params domain.PaginationParams) ([]*domain.EventInvitation, int, error) {
	f.lastListInvitationsEventID = eventID
	f.lastListInvitationsCallerID = callerID
	f.lastListInvitationsSearch = search
	f.lastListInvitationsParams = params
	if f.listEventInvitationsErr != nil {
		return nil, 0, f.listEventInvitationsErr
	}
	if f.listEventInvitationsResult != nil {
		return f.listEventInvitationsResult, f.listEventInvitationsTotal, nil
	}
	return []*domain.EventInvitation{}, 0, nil
}

func (f *fakeEventService) ListEventTags(ctx context.Context, eventID, callerID string) ([]*domain.Tag, error) {
	f.lastListEventTagsEventID = eventID
	f.lastListEventTagsCallerID = callerID
	if f.listEventTagsErr != nil {
		return nil, f.listEventTagsErr
	}
	if f.listEventTagsResult != nil {
		return f.listEventTagsResult, nil
	}
	return []*domain.Tag{}, nil
}

func (f *fakeEventService) AddEventTags(ctx context.Context, eventID, ownerID string, tagNames []string) ([]*domain.Tag, error) {
	f.lastAddEventTagsEventID = eventID
	f.lastAddEventTagsOwnerID = ownerID
	f.lastAddEventTagsTagNames = tagNames
	if f.addEventTagsErr != nil {
		return nil, f.addEventTagsErr
	}
	if f.addEventTagsResult != nil {
		return f.addEventTagsResult, nil
	}
	return []*domain.Tag{}, nil
}

func (f *fakeEventService) AddSessionTag(ctx context.Context, eventID, sessionID, ownerID, tagID string) error {
	f.lastAddSessionTagEventID = eventID
	f.lastAddSessionTagSessionID = sessionID
	f.lastAddSessionTagOwnerID = ownerID
	f.lastAddSessionTagTagID = tagID
	return f.addSessionTagErr
}

func (f *fakeEventService) RemoveSessionTag(ctx context.Context, eventID, sessionID, ownerID, tagID string) error {
	f.lastRemoveSessionTagEventID = eventID
	f.lastRemoveSessionTagSessionID = sessionID
	f.lastRemoveSessionTagOwnerID = ownerID
	f.lastRemoveSessionTagTagID = tagID
	return f.removeSessionTagErr
}

func (f *fakeEventService) RemoveEventTag(ctx context.Context, eventID, ownerID, tagID string) error {
	f.lastRemoveEventTagEventID = eventID
	f.lastRemoveEventTagOwnerID = ownerID
	f.lastRemoveEventTagTagID = tagID
	return f.removeEventTagErr
}

func (f *fakeEventService) UpdateEventTag(ctx context.Context, eventID, tagID, ownerID, name string) (*domain.Tag, error) {
	f.lastUpdateEventTagEventID = eventID
	f.lastUpdateEventTagTagID = tagID
	f.lastUpdateEventTagOwnerID = ownerID
	f.lastUpdateEventTagName = name
	if f.updateEventTagErr != nil {
		return nil, f.updateEventTagErr
	}
	if f.updateEventTagResult != nil {
		return f.updateEventTagResult, nil
	}
	return &domain.Tag{ID: tagID, Name: name}, nil
}

func (f *fakeEventService) ListEventSpeakers(ctx context.Context, eventID, ownerID string) ([]*domain.Speaker, error) {
	f.lastListEventSpeakersEventID = eventID
	f.lastListEventSpeakersOwnerID = ownerID
	if f.listEventSpeakersErr != nil {
		return nil, f.listEventSpeakersErr
	}
	if f.listEventSpeakersResult != nil {
		return f.listEventSpeakersResult, nil
	}
	return []*domain.Speaker{}, nil
}

func (f *fakeEventService) GetEventSpeaker(ctx context.Context, eventID, speakerID, ownerID string) (*domain.Speaker, []*domain.Session, error) {
	f.lastGetEventSpeakerEventID = eventID
	f.lastGetEventSpeakerSpeakerID = speakerID
	f.lastGetEventSpeakerOwnerID = ownerID
	if f.getEventSpeakerErr != nil {
		return nil, nil, f.getEventSpeakerErr
	}
	if f.getEventSpeakerResult != nil {
		sessions := f.getEventSpeakerSessions
		if sessions == nil {
			sessions = []*domain.Session{}
		}
		return f.getEventSpeakerResult, sessions, nil
	}
	return nil, nil, domain.ErrNotFound
}

func (f *fakeEventService) DeleteEventSpeaker(ctx context.Context, eventID, speakerID, ownerID string) error {
	f.lastDeleteEventSpeakerEventID = eventID
	f.lastDeleteEventSpeakerSpeakerID = speakerID
	f.lastDeleteEventSpeakerOwnerID = ownerID
	return f.deleteEventSpeakerErr
}

func (f *fakeEventService) CreateEventSpeaker(ctx context.Context, eventID, ownerID string, firstName, lastName, bio, tagLine, profilePicture string, isTopSpeaker bool) (*domain.Speaker, error) {
	f.lastCreateEventSpeakerEventID = eventID
	f.lastCreateEventSpeakerOwnerID = ownerID
	f.lastCreateEventSpeakerFirstName = firstName
	f.lastCreateEventSpeakerLastName = lastName
	if f.createEventSpeakerErr != nil {
		return nil, f.createEventSpeakerErr
	}
	if f.createEventSpeakerResult != nil {
		return f.createEventSpeakerResult, nil
	}
	return &domain.Speaker{ID: "sp-created", EventID: eventID, FirstName: firstName, LastName: lastName}, nil
}

func (f *fakeEventService) CreateEventRoom(ctx context.Context, eventID, ownerID, name string, capacity int, description, howToGetThere string, notBookable bool) (*domain.Room, error) {
	f.lastCreateEventRoomEventID = eventID
	f.lastCreateEventRoomOwnerID = ownerID
	f.lastCreateEventRoomName = name
	f.lastCreateEventRoomCapacity = capacity
	if f.createEventRoomErr != nil {
		return nil, f.createEventRoomErr
	}
	if f.createEventRoomResult != nil {
		return f.createEventRoomResult, nil
	}
	return &domain.Room{
		ID:            "room-created",
		EventID:       eventID,
		Name:          name,
		Capacity:      capacity,
		Description:   description,
		HowToGetThere: howToGetThere,
		NotBookable:   notBookable,
	}, nil
}

func (f *fakeEventService) CreateEventSession(ctx context.Context, eventID, ownerID, roomID, title, description string, startTime, endTime time.Time, tagNames, speakerIDs []string) (*domain.Session, error) {
	f.lastCreateEventSessionEventID = eventID
	f.lastCreateEventSessionOwnerID = ownerID
	f.lastCreateEventSessionRoomID = roomID
	f.lastCreateEventSessionTitle = title
	f.lastCreateEventSessionDesc = description
	f.lastCreateEventSessionStart = startTime
	f.lastCreateEventSessionEnd = endTime
	f.lastCreateEventSessionTags = tagNames
	f.lastCreateEventSessionSpeakers = speakerIDs
	if f.createEventSessionErr != nil {
		return nil, f.createEventSessionErr
	}
	if f.createEventSessionResult != nil {
		return f.createEventSessionResult, nil
	}
	var tags []*domain.Tag
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tags = append(tags, &domain.Tag{Name: name})
	}
	return &domain.Session{
		ID:          "sess-created",
		RoomID:      roomID,
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Tags:        tags,
		SpeakerIDs:  append([]string(nil), speakerIDs...),
	}, nil
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
			body:           `{"name":"Conf 2025"}`,
			wantStatus:     http.StatusCreated,
			wantBodySubstr: "",
			decodeEvent:    true,
			checkEvent: func(t *testing.T, event domain.Event) {
				assert.Equal(t, "ev-created", event.ID)
				assert.Equal(t, "Conf 2025", event.Name)
				assert.Equal(t, "user-123", event.OwnerID)
			},
		},
		{
			name:           "no user in context",
			body:           `{"name":"Conf 2025"}`,
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
			body:           `{}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "name is required",
			decodeEvent:    false,
			checkEvent:     nil,
		},
		{
			name:           "unknown field rejected",
			body:           `{"name":"Conf","id":"custom-id"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "unknown field",
			decodeEvent:    false,
			checkEvent:     nil,
		},
		{
			name:           "service error",
			body:           `{"name":"Conf"}`,
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
			decodeEvent:    false,
			checkEvent:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{createEventErr: tt.fakeErr}
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
			fake := &fakeEventService{importSessionizeErr: tt.fakeErr}
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
					{ID: "ev-1", Name: "Conf A", OwnerID: "user-123"},
					{ID: "ev-2", Name: "Conf B", OwnerID: "user-123"},
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
			name:           "no user in context",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
			checkEvents:    nil,
		},
		{
			name:           "service error",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
			checkEvents:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
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
		name          string
		eventID       string
		noUserContext bool
		fakeErr       error
		eventByID     map[string]struct {
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
					event:    &domain.Event{ID: "ev-123", Name: "Conf 2025", OwnerID: "user-1"},
					rooms:    []*domain.Room{{ID: "room-1", EventID: "ev-123", Name: "Room A"}},
					sessions: []*domain.Session{{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", Tags: []*domain.Tag{}}},
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
			name:           "no user in context",
			eventID:        "ev-123",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
			checkResponse:  nil,
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
			name:           "service error",
			eventID:        "ev-123",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
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
			name:           "no user in context",
			eventID:        "ev-123",
			roomID:         "room-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
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
			fake := &fakeEventService{toggleRoomErr: tt.fakeErr, toggleRoomResult: tt.fakeResult}
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

func TestScheduleController_ListEventRooms(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		noUserContext  bool
		fakeErr        error
		fakeResult     []*domain.Room
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			fakeResult: []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastListEventRoomsEventID)
				assert.Equal(t, "user-123", fake.lastListEventRoomsOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
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
			name:           "forbidden",
			eventID:        "ev-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{listEventRoomsErr: tt.fakeErr, listEventRoomsResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodGet, "http://test/events/"+tt.eventID+"/rooms", nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ListEventRooms(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_GetEventRoom(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		roomID         string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Room
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			roomID:     "room-1",
			fakeResult: &domain.Room{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: true},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastGetEventRoomEventID)
				assert.Equal(t, "room-1", fake.lastGetEventRoomRoomID)
				assert.Equal(t, "user-123", fake.lastGetEventRoomOwnerID)
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
			eventID:        "ev-1",
			roomID:         "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or roomID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			roomID:         "room-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
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
			eventID:        "ev-1",
			roomID:         "room-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{getEventRoomErr: tt.fakeErr, getEventRoomResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/rooms/" + tt.roomID
			req := httptest.NewRequest(http.MethodGet, path, nil)
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
			ctrl.GetEventRoom(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_UpdateEventRoom(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		roomID         string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Room
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			roomID:     "room-1",
			body:       `{"capacity":50,"description":"Big room","how_to_get_there":"Floor 2","not_bookable":true}`,
			fakeResult: &domain.Room{ID: "room-1", EventID: "ev-1", Name: "Room A", Capacity: 50, Description: "Big room", HowToGetThere: "Floor 2", NotBookable: true},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastUpdateEventRoomEventID)
				assert.Equal(t, "room-1", fake.lastUpdateEventRoomRoomID)
				assert.Equal(t, "user-123", fake.lastUpdateEventRoomOwnerID)
				assert.Nil(t, fake.lastUpdateEventRoomName)
			},
		},
		{
			name:       "success with name",
			eventID:    "ev-1",
			roomID:     "room-1",
			body:       `{"name":"Main Hall","capacity":50,"description":"Big room","how_to_get_there":"Floor 2","not_bookable":true}`,
			fakeResult: &domain.Room{ID: "room-1", EventID: "ev-1", Name: "Main Hall", Capacity: 50, Description: "Big room", HowToGetThere: "Floor 2", NotBookable: true},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastUpdateEventRoomEventID)
				assert.Equal(t, "room-1", fake.lastUpdateEventRoomRoomID)
				assert.Equal(t, "user-123", fake.lastUpdateEventRoomOwnerID)
				require.NotNil(t, fake.lastUpdateEventRoomName)
				assert.Equal(t, "Main Hall", *fake.lastUpdateEventRoomName)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			roomID:         "room-1",
			body:           `{"capacity":0}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or roomID",
		},
		{
			name:           "missing roomID",
			eventID:        "ev-1",
			roomID:         "",
			body:           `{"capacity":0}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or roomID",
		},
		{
			name:           "negative capacity",
			eventID:        "ev-1",
			roomID:         "room-1",
			body:           `{"capacity":-1}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "capacity",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			roomID:         "room-1",
			body:           `{"capacity":10}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			roomID:         "room-1",
			body:           `{"capacity":10}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event or room not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			roomID:         "room-1",
			body:           `{"capacity":10}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{updateEventRoomErr: tt.fakeErr, updateEventRoomResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodPatch, "http://test/events/"+tt.eventID+"/rooms/"+tt.roomID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
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
			ctrl.UpdateEventRoom(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_DeleteEventRoom(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		roomID         string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			roomID:     "room-1",
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastDeleteEventRoomEventID)
				assert.Equal(t, "room-1", fake.lastDeleteEventRoomRoomID)
				assert.Equal(t, "user-123", fake.lastDeleteEventRoomOwnerID)
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
			eventID:        "ev-1",
			roomID:         "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or roomID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			roomID:         "room-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
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
			eventID:        "ev-1",
			roomID:         "room-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{deleteEventRoomErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/rooms/" + tt.roomID
			req := httptest.NewRequest(http.MethodDelete, path, nil)
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
			ctrl.DeleteEventRoom(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
				var data DeleteEventResponse
				dataBytes, _ := json.Marshal(envelope.Data)
				require.NoError(t, json.Unmarshal(dataBytes, &data))
				assert.Equal(t, "deleted", data.Status)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_ListEventSpeakers(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		noUserContext  bool
		fakeErr        error
		fakeResult     []*domain.Speaker
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			fakeResult: []*domain.Speaker{{ID: "sp-1", EventID: "ev-1", FirstName: "Alice", LastName: ""}},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastListEventSpeakersEventID)
				assert.Equal(t, "user-123", fake.lastListEventSpeakersOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{listEventSpeakersErr: tt.fakeErr, listEventSpeakersResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/speakers"
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ListEventSpeakers(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_ListEventTags(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		noUserContext  bool
		fakeErr        error
		fakeResult     []*domain.Tag
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			fakeResult: []*domain.Tag{{ID: "tag-1", Name: "Go"}, {ID: "tag-2", Name: "Rust"}},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastListEventTagsEventID)
				assert.Equal(t, "user-123", fake.lastListEventTagsCallerID)
			},
		},
		{
			name:       "success empty list",
			eventID:    "ev-1",
			fakeResult: []*domain.Tag{},
			wantStatus: http.StatusOK,
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
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
			name:           "forbidden",
			eventID:        "ev-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{listEventTagsErr: tt.fakeErr, listEventTagsResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/tags"
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ListEventTags(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var tags []domain.Tag
				require.NoError(t, json.Unmarshal(dataBytes, &tags))
				require.Len(t, tags, len(tt.fakeResult))
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_AddEventTags(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     []*domain.Tag
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			body:       `{"tags":["Go","Rust"]}`,
			fakeResult: []*domain.Tag{{ID: "tag-1", Name: "Go"}, {ID: "tag-2", Name: "Rust"}},
			wantStatus: http.StatusCreated,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastAddEventTagsEventID)
				assert.Equal(t, "user-123", fake.lastAddEventTagsOwnerID)
				assert.ElementsMatch(t, []string{"Go", "Rust"}, fake.lastAddEventTagsTagNames)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{"tags":["Go"]}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			body:           `{"tags":["Go"]}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "invalid body empty tags",
			eventID:        "ev-1",
			body:           `{"tags":[]}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "at least one tag",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			body:           `{"tags":["Go"]}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			body:           `{"tags":["Go"]}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{addEventTagsErr: tt.fakeErr, addEventTagsResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/tags"
			if tt.eventID == "" {
				path = "http://test/events//tags"
			}
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.AddEventTags(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusCreated && tt.checkCall != nil {
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_UpdateEventTag(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		tagID          string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Tag
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			tagID:      "tag-1",
			body:       `{"name":"Golang"}`,
			fakeResult: &domain.Tag{ID: "tag-1", Name: "Golang"},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastUpdateEventTagEventID)
				assert.Equal(t, "tag-1", fake.lastUpdateEventTagTagID)
				assert.Equal(t, "user-123", fake.lastUpdateEventTagOwnerID)
				assert.Equal(t, "Golang", fake.lastUpdateEventTagName)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			tagID:          "tag-1",
			body:           `{"name":"X"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or tagID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			tagID:          "tag-1",
			body:           `{"name":"X"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			tagID:          "tag-1",
			body:           `{"name":"X"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			tagID:          "tag-1",
			body:           `{"name":"X"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{updateEventTagErr: tt.fakeErr, updateEventTagResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := fmt.Sprintf("http://test/events/%s/tags/%s", tt.eventID, tt.tagID)
			req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("eventID", tt.eventID)
			req.SetPathValue("tagID", tt.tagID)
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.UpdateEventTag(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_AddSessionTag(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		sessionID      string
		body           string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:      "success",
			eventID:   "ev-1",
			sessionID: "sess-1",
			body:      `{"tag_id":"tag-1"}`,
			wantStatus: http.StatusNoContent,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastAddSessionTagEventID)
				assert.Equal(t, "sess-1", fake.lastAddSessionTagSessionID)
				assert.Equal(t, "user-123", fake.lastAddSessionTagOwnerID)
				assert.Equal(t, "tag-1", fake.lastAddSessionTagTagID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			sessionID:      "sess-1",
			body:           `{"tag_id":"tag-1"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{"tag_id":"tag-1"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "invalid body missing tag_id",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "tag_id",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			sessionID:      "sess-1",
			body:           `{"tag_id":"tag-1"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{"tag_id":"tag-1"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{addSessionTagErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := fmt.Sprintf("http://test/events/%s/sessions/%s/tags", tt.eventID, tt.sessionID)
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("eventID", tt.eventID)
			req.SetPathValue("sessionID", tt.sessionID)
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.AddSessionTag(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusNoContent {
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				return
			}
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_RemoveSessionTag(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		sessionID      string
		tagID          string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:   "ev-1",
			sessionID: "sess-1",
			tagID:     "tag-1",
			wantStatus: http.StatusNoContent,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastRemoveSessionTagEventID)
				assert.Equal(t, "sess-1", fake.lastRemoveSessionTagSessionID)
				assert.Equal(t, "user-123", fake.lastRemoveSessionTagOwnerID)
				assert.Equal(t, "tag-1", fake.lastRemoveSessionTagTagID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			sessionID:      "sess-1",
			tagID:          "tag-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID, sessionID, or tagID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			tagID:          "tag-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			sessionID:      "sess-1",
			tagID:          "tag-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			tagID:          "tag-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{removeSessionTagErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := fmt.Sprintf("http://test/events/%s/sessions/%s/tags/%s", tt.eventID, tt.sessionID, tt.tagID)
			req := httptest.NewRequest(http.MethodDelete, path, nil)
			req.SetPathValue("eventID", tt.eventID)
			req.SetPathValue("sessionID", tt.sessionID)
			req.SetPathValue("tagID", tt.tagID)
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.RemoveSessionTag(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusNoContent {
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				return
			}
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_RemoveEventTag(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		tagID          string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			tagID:      "tag-1",
			wantStatus: http.StatusNoContent,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastRemoveEventTagEventID)
				assert.Equal(t, "user-123", fake.lastRemoveEventTagOwnerID)
				assert.Equal(t, "tag-1", fake.lastRemoveEventTagTagID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			tagID:          "tag-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or tagID",
		},
		{
			name:           "missing tagID",
			eventID:        "ev-1",
			tagID:          "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or tagID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			tagID:          "tag-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event or tag not found",
			eventID:        "ev-missing",
			tagID:          "tag-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			tagID:          "tag-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "internal error",
			eventID:        "ev-1",
			tagID:          "tag-1",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{removeEventTagErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := fmt.Sprintf("http://test/events/%s/tags/%s", tt.eventID, tt.tagID)
			req := httptest.NewRequest(http.MethodDelete, path, nil)
			req.SetPathValue("eventID", tt.eventID)
			req.SetPathValue("tagID", tt.tagID)
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.RemoveEventTag(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusNoContent {
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				return
			}
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_GetEventSpeaker(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		speakerID      string
		noUserContext  bool
		fakeErr        error
		fakeSpeaker    *domain.Speaker
		fakeSessions   []*domain.Session
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:         "success",
			eventID:      "ev-1",
			speakerID:    "sp-1",
			fakeSpeaker:  &domain.Speaker{ID: "sp-1", EventID: "ev-1", FirstName: "Alice", LastName: ""},
			fakeSessions: []*domain.Session{{ID: "sess-1", Title: "Talk"}},
			wantStatus:   http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastGetEventSpeakerEventID)
				assert.Equal(t, "sp-1", fake.lastGetEventSpeakerSpeakerID)
				assert.Equal(t, "user-123", fake.lastGetEventSpeakerOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			speakerID:      "sp-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or speakerID",
		},
		{
			name:           "missing speakerID",
			eventID:        "ev-1",
			speakerID:      "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or speakerID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			speakerID:      "sp-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			speakerID:      "sp-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event or speaker not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			speakerID:      "sp-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
				getEventSpeakerErr:      tt.fakeErr,
				getEventSpeakerResult:   tt.fakeSpeaker,
				getEventSpeakerSessions: tt.fakeSessions,
			}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/speakers/" + tt.speakerID
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if tt.speakerID != "" {
				req.SetPathValue("speakerID", tt.speakerID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.GetEventSpeaker(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_DeleteEventSpeaker(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		speakerID      string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success returns 204",
			eventID:    "ev-1",
			speakerID:  "sp-1",
			wantStatus: http.StatusNoContent,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastDeleteEventSpeakerEventID)
				assert.Equal(t, "sp-1", fake.lastDeleteEventSpeakerSpeakerID)
				assert.Equal(t, "user-123", fake.lastDeleteEventSpeakerOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			speakerID:      "sp-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or speakerID",
		},
		{
			name:           "missing speakerID",
			eventID:        "ev-1",
			speakerID:      "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or speakerID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			speakerID:      "sp-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			speakerID:      "sp-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event or speaker not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			speakerID:      "sp-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{deleteEventSpeakerErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/speakers/" + tt.speakerID
			req := httptest.NewRequest(http.MethodDelete, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if tt.speakerID != "" {
				req.SetPathValue("speakerID", tt.speakerID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.DeleteEventSpeaker(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Empty(t, rr.Body.Bytes())
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				return
			}
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.checkCall != nil {
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_CreateEventSpeaker(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		eventID        string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			body:       `{"first_name":"Jane","last_name":"Doe","bio":"Bio","is_top_speaker":true}`,
			wantStatus: http.StatusCreated,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastCreateEventSpeakerEventID)
				assert.Equal(t, "user-123", fake.lastCreateEventSpeakerOwnerID)
				assert.Equal(t, "Jane", fake.lastCreateEventSpeakerFirstName)
				assert.Equal(t, "Doe", fake.lastCreateEventSpeakerLastName)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{"first_name":"Alice"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			body:           `{"first_name":"Alice"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "validation missing name",
			eventID:        "ev-1",
			body:           `{}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "at least one of first_name or last_name is required",
		},
		{
			name:           "invalid json",
			eventID:        "ev-1",
			body:           `{invalid`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "invalid",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			body:           `{"first_name":"Alice"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			body:           `{"first_name":"Alice"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{createEventSpeakerErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/speakers"
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.CreateEventSpeaker(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusCreated && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_CreateEventRoom(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Room
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:    "success",
			eventID: "ev-1",
			body:    `{"name":"Room A","capacity":50,"description":"Big room","how_to_get_there":"Floor 2","not_bookable":false}`,
			fakeResult: &domain.Room{
				ID:            "room-1",
				EventID:       "ev-1",
				Name:          "Room A",
				Capacity:      50,
				Description:   "Big room",
				HowToGetThere: "Floor 2",
				NotBookable:   false,
			},
			wantStatus: http.StatusCreated,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastCreateEventRoomEventID)
				assert.Equal(t, "user-123", fake.lastCreateEventRoomOwnerID)
				assert.Equal(t, "Room A", fake.lastCreateEventRoomName)
				assert.Equal(t, 50, fake.lastCreateEventRoomCapacity)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{"name":"Room A"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			body:           `{"name":"Room A"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "validation missing name",
			eventID:        "ev-1",
			body:           `{"capacity":10}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "name is required",
		},
		{
			name:           "validation negative capacity",
			eventID:        "ev-1",
			body:           `{"name":"Room A","capacity":-1}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "capacity must be non-negative",
		},
		{
			name:           "invalid json",
			eventID:        "ev-1",
			body:           `{invalid`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "invalid",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			body:           `{"name":"Room A"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			body:           `{"name":"Room A"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			body:           `{"name":"Room A"}`,
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
				createEventRoomErr:    tt.fakeErr,
				createEventRoomResult: tt.fakeResult,
			}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/rooms"
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.CreateEventRoom(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)

			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))

			if tt.wantStatus == http.StatusCreated && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_CreateEventSession(t *testing.T) {
	start := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		eventID        string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Session
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:    "success",
			eventID: "ev-1",
			body:    `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z","description":"Desc","tags":["go","conf"],"speaker_ids":["sp-1","sp-2"]}`,
			fakeResult: &domain.Session{
				ID:          "sess-1",
				RoomID:      "room-1",
				Title:       "Talk",
				Description: "Desc",
				StartTime:   start,
				EndTime:     end,
				Tags:        []*domain.Tag{{ID: "tag-go", Name: "go"}, {ID: "tag-conf", Name: "conf"}},
				SpeakerIDs:  []string{"sp-1", "sp-2"},
			},
			wantStatus: http.StatusCreated,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastCreateEventSessionEventID)
				assert.Equal(t, "user-123", fake.lastCreateEventSessionOwnerID)
				assert.Equal(t, "room-1", fake.lastCreateEventSessionRoomID)
				assert.Equal(t, "Talk", fake.lastCreateEventSessionTitle)
				assert.Equal(t, "Desc", fake.lastCreateEventSessionDesc)
				assert.True(t, fake.lastCreateEventSessionStart.Equal(start))
				assert.True(t, fake.lastCreateEventSessionEnd.Equal(end))
				assert.ElementsMatch(t, []string{"go", "conf"}, fake.lastCreateEventSessionTags)
				assert.ElementsMatch(t, []string{"sp-1", "sp-2"}, fake.lastCreateEventSessionSpeakers)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "validation missing room_id",
			eventID:        "ev-1",
			body:           `{"title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "room_id is required",
		},
		{
			name:           "validation missing title",
			eventID:        "ev-1",
			body:           `{"room_id":"room-1","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "title is required",
		},
		{
			name:           "validation end before start",
			eventID:        "ev-1",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T11:00:00Z","end_time":"2025-03-01T10:00:00Z"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "end_time must be after start_time",
		},
		{
			name:           "invalid json",
			eventID:        "ev-1",
			body:           `{invalid`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "invalid",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event, room, or speaker not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "invalid input from service",
			eventID:        "ev-1",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T09:00:00Z"}`,
			fakeErr:        fmt.Errorf("end_time must be after start_time: %w", domain.ErrInvalidInput),
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "end_time must be after start_time",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			body:           `{"room_id":"room-1","title":"Talk","start_time":"2025-03-01T10:00:00Z","end_time":"2025-03-01T11:00:00Z"}`,
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
				createEventSessionErr:    tt.fakeErr,
				createEventSessionResult: tt.fakeResult,
			}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/sessions"
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.CreateEventSession(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantStatus == http.StatusCreated && tt.checkCall != nil {
				var envelope helpers.APIResponse
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
				return
			}

			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_DeleteEventSession(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		sessionID      string
		noUserContext  bool
		fakeErr        error
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			sessionID:  "sess-1",
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastDeleteEventSessionEventID)
				assert.Equal(t, "sess-1", fake.lastDeleteEventSessionSessionID)
				assert.Equal(t, "user-123", fake.lastDeleteEventSessionOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			sessionID:      "sess-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionID",
		},
		{
			name:           "missing sessionID",
			eventID:        "ev-1",
			sessionID:      "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			sessionID:      "sess-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event or session not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{deleteEventSessionErr: tt.fakeErr}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/sessions/" + tt.sessionID
			req := httptest.NewRequest(http.MethodDelete, path, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if tt.sessionID != "" {
				req.SetPathValue("sessionID", tt.sessionID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.DeleteEventSession(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
				var data DeleteEventResponse
				dataBytes, _ := json.Marshal(envelope.Data)
				require.NoError(t, json.Unmarshal(dataBytes, &data))
				assert.Equal(t, "deleted", data.Status)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_UpdateSessionContent(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		sessionID      string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Session
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			sessionID:  "sess-1",
			body:       `{"title":"New Title","description":"New desc"}`,
			fakeResult: &domain.Session{ID: "sess-1", Title: "New Title", Description: "New desc"},
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastUpdateSessionContentEventID)
				assert.Equal(t, "sess-1", fake.lastUpdateSessionContentSessionID)
				assert.Equal(t, "user-123", fake.lastUpdateSessionContentOwnerID)
				require.NotNil(t, fake.lastUpdateSessionContentTitle)
				assert.Equal(t, "New Title", *fake.lastUpdateSessionContentTitle)
				require.NotNil(t, fake.lastUpdateSessionContentDesc)
				assert.Equal(t, "New desc", *fake.lastUpdateSessionContentDesc)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			sessionID:      "sess-1",
			body:           `{"title":"X"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionID",
		},
		{
			name:           "missing sessionID",
			eventID:        "ev-1",
			sessionID:      "",
			body:           `{"title":"X"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID or sessionID",
		},
		{
			name:           "empty title",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{"title":""}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "title cannot be empty",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{"title":"X"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "not found",
			eventID:        "ev-missing",
			sessionID:      "sess-1",
			body:           `{"title":"X"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event or session not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{"title":"X"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			sessionID:      "sess-1",
			body:           `{"title":"X"}`,
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{updateSessionContentErr: tt.fakeErr, updateSessionContentResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			path := "http://test/events/" + tt.eventID + "/sessions/" + tt.sessionID + "/content"
			req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if tt.sessionID != "" {
				req.SetPathValue("sessionID", tt.sessionID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.UpdateSessionContent(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error)
				tt.checkCall(t, fake)
				var data domain.Session
				dataBytes, _ := json.Marshal(envelope.Data)
				require.NoError(t, json.Unmarshal(dataBytes, &data))
				assert.Equal(t, "New Title", data.Title)
				assert.Equal(t, "New desc", data.Description)
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
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
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-123",
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
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
			name:           "no user in context",
			eventID:        "ev-123",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
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
			fake := &fakeEventService{deleteEventErr: tt.fakeErr}
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

func TestScheduleController_UpdateEvent(t *testing.T) {
	eventDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	desc := "Annual conference"
	lat, lng := 40.7128, -74.0060
	updatedEvent := &domain.Event{
		ID: "ev-123", Name: "Conf", EventCode: "abc1", OwnerID: "user-123",
		Date: &eventDate, Description: &desc, LocationLat: &lat, LocationLng: &lng,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		eventID        string
		body           string
		noUserContext  bool
		fakeErr        error
		fakeResult     *domain.Event
		wantStatus     int
		wantBodySubstr string
		checkCall      func(t *testing.T, fake *fakeEventService)
	}{
		{
			name:       "success",
			eventID:    "ev-123",
			body:       `{"description":"Annual conference"}`,
			fakeResult: updatedEvent,
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-123", fake.lastUpdateEventID)
				assert.Equal(t, "user-123", fake.lastUpdateEventOwnerID)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-123",
			body:           `{}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			body:           `{"description":"x"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden not owner",
			eventID:        "ev-123",
			body:           `{"description":"x"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "validation invalid location_lat",
			eventID:        "ev-123",
			body:           `{"location_lat": 100}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "location_lat",
		},
		{
			name:           "service error",
			eventID:        "ev-123",
			body:           `{}`,
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{updateEventErr: tt.fakeErr, updateEventResult: tt.fakeResult}
			ctrl := NewScheduleController(testLogger, fake)
			var body bytes.Buffer
			if tt.body != "" {
				body = *bytes.NewBufferString(tt.body)
			}
			req := httptest.NewRequest(http.MethodPatch, "http://test/events/"+tt.eventID, &body)
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.UpdateEvent(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope), "response must be valid JSON envelope")
			if tt.wantStatus == http.StatusOK && tt.checkCall != nil {
				require.Nil(t, envelope.Error, "success response must have error nil")
				tt.checkCall(t, fake)
				dataBytes, _ := json.Marshal(envelope.Data)
				var event domain.Event
				require.NoError(t, json.Unmarshal(dataBytes, &event))
				assert.Equal(t, "ev-123", event.ID)
				if tt.fakeResult != nil && tt.fakeResult.Description != nil {
					require.NotNil(t, event.Description)
					assert.Equal(t, *tt.fakeResult.Description, *event.Description)
				}
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
			name:       "forbidden",
			eventID:    "ev-1",
			body:       `{"email":"teammate@example.com"}`,
			fakeErr:    domain.ErrForbidden,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "conflict already member",
			eventID:    "ev-1",
			body:       `{"email":"teammate@example.com"}`,
			fakeErr:    domain.ErrAlreadyMember,
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{addTeamMemberByEmailErr: tt.fakeErr, addTeamMemberByEmailResult: tt.fakeResult}
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
			name:       "event not found",
			eventID:    "ev-1",
			fakeErr:    domain.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "forbidden",
			eventID:    "ev-1",
			fakeErr:    domain.ErrForbidden,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{listTeamMembersErr: tt.fakeErr, listTeamMembersResult: tt.fakeResult}
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
			name:       "event not found",
			eventID:    "ev-1",
			userID:     "user-2",
			fakeErr:    domain.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "forbidden",
			eventID:    "ev-1",
			userID:     "user-2",
			fakeErr:    domain.ErrForbidden,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{removeTeamMemberErr: tt.fakeErr}
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

func TestScheduleController_ListEventInvitations(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		query          string
		fakeErr        error
		fakeResult     []*domain.EventInvitation
		fakeTotal      int
		wantStatus     int
		wantBodySubstr string
		noUserContext  bool
		checkCall      func(t *testing.T, fake *fakeEventService)
		checkData      func(t *testing.T, data ListEventInvitationsResponse)
	}{
		{
			name:       "success with invitations default pagination",
			eventID:    "ev-1",
			fakeResult: []*domain.EventInvitation{{ID: "inv-1", EventID: "ev-1", Email: "a@example.com"}, {ID: "inv-2", EventID: "ev-1", Email: "b@example.com"}},
			fakeTotal:  2,
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastListInvitationsEventID)
				assert.Equal(t, "user-123", fake.lastListInvitationsCallerID)
				assert.Equal(t, 1, fake.lastListInvitationsParams.Page)
				assert.Equal(t, 20, fake.lastListInvitationsParams.PageSize)
			},
			checkData: func(t *testing.T, data ListEventInvitationsResponse) {
				require.Len(t, data.Items, 2)
				assert.Equal(t, "a@example.com", data.Items[0].Email)
				assert.Equal(t, "b@example.com", data.Items[1].Email)
				assert.Equal(t, 1, data.Pagination.Page)
				assert.Equal(t, 20, data.Pagination.PageSize)
				assert.Equal(t, 2, data.Pagination.Total)
				assert.Equal(t, 1, data.Pagination.TotalPages)
			},
		},
		{
			name:       "success with explicit page and page_size",
			eventID:    "ev-1",
			query:      "?page=2&page_size=5",
			fakeResult: []*domain.EventInvitation{},
			fakeTotal:  10,
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, 2, fake.lastListInvitationsParams.Page)
				assert.Equal(t, 5, fake.lastListInvitationsParams.PageSize)
			},
			checkData: func(t *testing.T, data ListEventInvitationsResponse) {
				require.Len(t, data.Items, 0)
				assert.Equal(t, 2, data.Pagination.Page)
				assert.Equal(t, 5, data.Pagination.PageSize)
				assert.Equal(t, 10, data.Pagination.Total)
				assert.Equal(t, 2, data.Pagination.TotalPages)
			},
		},
		{
			name:       "success empty list",
			eventID:    "ev-1",
			fakeResult: []*domain.EventInvitation{},
			fakeTotal:  0,
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data ListEventInvitationsResponse) {
				require.Len(t, data.Items, 0)
				assert.Equal(t, 0, data.Pagination.Total)
				assert.Equal(t, 0, data.Pagination.TotalPages)
			},
		},
		{
			name:       "success with search param",
			eventID:    "ev-1",
			query:      "?search=alice",
			fakeResult: []*domain.EventInvitation{{ID: "inv-1", EventID: "ev-1", Email: "alice@example.com"}},
			fakeTotal:  1,
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "alice", fake.lastListInvitationsSearch)
			},
			checkData: func(t *testing.T, data ListEventInvitationsResponse) {
				require.Len(t, data.Items, 1)
				assert.Equal(t, "alice@example.com", data.Items[0].Email)
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event not found",
			eventID:        "ev-1",
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
		{
			name:           "service error",
			eventID:        "ev-1",
			fakeErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
				listEventInvitationsErr:    tt.fakeErr,
				listEventInvitationsResult: tt.fakeResult,
				listEventInvitationsTotal:  tt.fakeTotal,
			}
			ctrl := NewScheduleController(testLogger, fake)
			url := "http://test/events/" + tt.eventID + "/invitations"
			if tt.query != "" {
				url += tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.ListEventInvitations(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				if tt.checkData != nil {
					dataBytes, err := json.Marshal(envelope.Data)
					require.NoError(t, err)
					var data ListEventInvitationsResponse
					require.NoError(t, json.Unmarshal(dataBytes, &data))
					tt.checkData(t, data)
				}
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestScheduleController_SendEventInvitations(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		body           string
		fakeErr        error
		fakeSent       int
		fakeFailed     []string
		wantStatus     int
		wantBodySubstr string
		noUserContext  bool
		checkCall      func(t *testing.T, fake *fakeEventService)
		checkData      func(t *testing.T, data SendEventInvitationsResponse)
	}{
		{
			name:       "success",
			eventID:    "ev-1",
			body:       `{"emails":"a@example.com, b@example.com"}`,
			fakeSent:   2,
			fakeFailed: nil,
			wantStatus: http.StatusOK,
			checkCall: func(t *testing.T, fake *fakeEventService) {
				assert.Equal(t, "ev-1", fake.lastSendInvitationsEventID)
				assert.Equal(t, "user-123", fake.lastSendInvitationsOwnerID)
				require.Len(t, fake.lastSendInvitationsEmails, 2)
				assert.Contains(t, fake.lastSendInvitationsEmails, "a@example.com")
				assert.Contains(t, fake.lastSendInvitationsEmails, "b@example.com")
			},
			checkData: func(t *testing.T, data SendEventInvitationsResponse) {
				assert.Equal(t, 2, data.Sent)
				assert.Nil(t, data.Failed)
			},
		},
		{
			name:       "success with partial failures",
			eventID:    "ev-1",
			body:       `{"emails":"ok@x.com fail@x.com"}`,
			fakeSent:   1,
			fakeFailed: []string{"fail@x.com"},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data SendEventInvitationsResponse) {
				assert.Equal(t, 1, data.Sent)
				require.Len(t, data.Failed, 1)
				assert.Equal(t, "fail@x.com", data.Failed[0])
			},
		},
		{
			name:           "missing eventID",
			eventID:        "",
			body:           `{"emails":"a@example.com"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "missing eventID",
		},
		{
			name:           "empty emails",
			eventID:        "ev-1",
			body:           `{"emails":""}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "emails is required",
		},
		{
			name:           "no valid emails after parse",
			eventID:        "ev-1",
			body:           `{"emails":"not-valid, also-invalid"}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "no valid emails",
		},
		{
			name:           "no user in context",
			eventID:        "ev-1",
			body:           `{"emails":"a@example.com"}`,
			noUserContext:  true,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "unauthorized",
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			body:           `{"emails":"a@example.com"}`,
			fakeErr:        domain.ErrNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "event not found",
		},
		{
			name:           "forbidden",
			eventID:        "ev-1",
			body:           `{"emails":"a@example.com"}`,
			fakeErr:        domain.ErrForbidden,
			wantStatus:     http.StatusForbidden,
			wantBodySubstr: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeEventService{
				sendEventInvitationsErr:    tt.fakeErr,
				sendEventInvitationsSent:   tt.fakeSent,
				sendEventInvitationsFailed: tt.fakeFailed,
			}
			ctrl := NewScheduleController(testLogger, fake)
			req := httptest.NewRequest(http.MethodPost, "http://test/events/"+tt.eventID+"/invitations", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.eventID != "" {
				req.SetPathValue("eventID", tt.eventID)
			}
			if !tt.noUserContext {
				req = req.WithContext(middleware.SetUserID(req.Context(), "user-123"))
			}
			rr := httptest.NewRecorder()
			ctrl.SendEventInvitations(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				if tt.checkCall != nil {
					tt.checkCall(t, fake)
				}
				if tt.checkData != nil {
					dataBytes, err := json.Marshal(envelope.Data)
					require.NoError(t, err)
					var data SendEventInvitationsResponse
					require.NoError(t, json.Unmarshal(dataBytes, &data))
					tt.checkData(t, data)
				}
			}
			if tt.wantBodySubstr != "" && envelope.Error != nil {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}
