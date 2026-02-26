package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"multitrackticketing/internal/domain"
)

type mockEventRegistrationRepository struct {
	regsByUser         map[string][]*domain.EventRegistration
	regByEventAndUser  map[string]*domain.EventRegistration
	err                error
}

func (m *mockEventRegistrationRepository) Create(ctx context.Context, reg *domain.EventRegistration) error {
	return nil
}

func (m *mockEventRegistrationRepository) GetByEventAndUser(ctx context.Context, eventID, userID string) (*domain.EventRegistration, error) {
	if m.regByEventAndUser != nil {
		key := eventID + ":" + userID
		if reg, ok := m.regByEventAndUser[key]; ok {
			return reg, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockEventRegistrationRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.EventRegistration, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.regsByUser[userID], nil
}

type mockEventRepository struct {
	events       map[string]*domain.Event
	eventsByCode map[string]*domain.Event
	err          error
}

func (m *mockEventRepository) Create(ctx context.Context, event *domain.Event) error {
	return nil
}

func (m *mockEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	ev, ok := m.events[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ev, nil
}

func (m *mockEventRepository) GetByEventCode(ctx context.Context, eventCode string) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.eventsByCode != nil {
		ev, ok := m.eventsByCode[eventCode]
		if !ok {
			return nil, domain.ErrNotFound
		}
		return ev, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockEventRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	return nil, nil
}

func (m *mockEventRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockEventRepository) Update(ctx context.Context, eventID string, date *time.Time, description *string, locationLat, locationLng *float64) (*domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	ev, ok := m.events[eventID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ev, nil
}

type mockSessionRepository struct {
	roomsByEvent    map[string][]*domain.Room
	sessionsByEvent map[string][]*domain.Session
	err             error
}

func (m *mockSessionRepository) CreateRoom(ctx context.Context, room *domain.Room) error { return nil }
func (m *mockSessionRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	return nil
}
func (m *mockSessionRepository) CreateSpeaker(ctx context.Context, speaker *domain.Speaker) error {
	return nil
}
func (m *mockSessionRepository) CreateSessionSpeaker(ctx context.Context, sessionID, speakerID string) error {
	return nil
}
func (m *mockSessionRepository) DeleteScheduleByEventID(ctx context.Context, eventID string) error {
	return nil
}
func (m *mockSessionRepository) DeleteSpeakersByEventID(ctx context.Context, eventID string) error {
	return nil
}
func (m *mockSessionRepository) GetRoomByID(ctx context.Context, roomID string) (*domain.Room, error) {
	return nil, domain.ErrNotFound
}
func (m *mockSessionRepository) ListRoomsByEventID(ctx context.Context, eventID string) ([]*domain.Room, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.roomsByEvent != nil {
		return m.roomsByEvent[eventID], nil
	}
	return nil, nil
}
func (m *mockSessionRepository) ListSessionsByEventID(ctx context.Context, eventID string) ([]*domain.Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.sessionsByEvent != nil {
		return m.sessionsByEvent[eventID], nil
	}
	return nil, nil
}
func (m *mockSessionRepository) ListSpeakerIDsBySessionIDs(ctx context.Context, sessionIDs []string) (map[string][]string, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetSpeakerByID(ctx context.Context, speakerID string) (*domain.Speaker, error) {
	return nil, domain.ErrNotFound
}
func (m *mockSessionRepository) ListSpeakersByEventID(ctx context.Context, eventID string) ([]*domain.Speaker, error) {
	return nil, nil
}
func (m *mockSessionRepository) ListSessionIDsBySpeakerID(ctx context.Context, speakerID string) ([]string, error) {
	return nil, nil
}
func (m *mockSessionRepository) ListSessionsByIDs(ctx context.Context, sessionIDs []string) ([]*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepository) DeleteSpeaker(ctx context.Context, speakerID string) error {
	return nil
}
func (m *mockSessionRepository) SetRoomNotBookable(ctx context.Context, roomID string, notBookable bool) (*domain.Room, error) {
	return nil, nil
}
func (m *mockSessionRepository) UpdateRoomDetails(ctx context.Context, roomID string, name string, capacity int, description, howToGetThere string, notBookable bool) (*domain.Room, error) {
	return nil, nil
}
func (m *mockSessionRepository) DeleteRoom(ctx context.Context, roomID string) error { return nil }
func (m *mockSessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	return nil, domain.ErrNotFound
}

func (m *mockSessionRepository) UpdateSessionSchedule(ctx context.Context, sessionID string, roomID *string, startTime, endTime *time.Time) (*domain.Session, error) {
	return nil, nil
}

func (m *mockSessionRepository) UpdateSessionContent(ctx context.Context, sessionID string, title *string, description *string) (*domain.Session, error) {
	return nil, nil
}

func TestAttendeeService_ListMyRegisteredEvents(t *testing.T) {
	now := time.Now()
	event1 := &domain.Event{ID: "e1", Name: "Event 1"}
	event2 := &domain.Event{ID: "e2", Name: "Event 2"}

	tests := []struct {
		name          string
		regRepo       *mockEventRegistrationRepository
		eventRepo     *mockEventRepository
		userID        string
		wantCount     int
		wantErr       bool
	}{
		{
			name: "no registrations returns empty slice",
			regRepo: &mockEventRegistrationRepository{
				regsByUser: map[string][]*domain.EventRegistration{},
			},
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{},
			},
			userID:    "u1",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "multiple registrations across events",
			regRepo: &mockEventRegistrationRepository{
				regsByUser: map[string][]*domain.EventRegistration{
					"u1": {
						{ID: "r1", EventID: "e1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
						{ID: "r2", EventID: "e2", UserID: "u1", CreatedAt: now, UpdatedAt: now},
					},
				},
			},
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{
					"e1": event1,
					"e2": event2,
				},
			},
			userID:    "u1",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "registration repo error",
			regRepo: &mockEventRegistrationRepository{
				err: errors.New("db error"),
			},
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{},
			},
			userID:    "u1",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &attendeeService{
				eventRepo:        tt.eventRepo,
				registrationRepo: tt.regRepo,
				sessionRepo:      &mockSessionRepository{},
			}

			got, err := svc.ListMyRegisteredEvents(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v (err=%v)", tt.wantErr, err != nil, err)
			}
			if err == nil && len(got) != tt.wantCount {
				t.Fatalf("expected %d results, got %d", tt.wantCount, len(got))
			}
		})
	}
}

func TestAttendeeService_RegisterForEventByCode(t *testing.T) {
	now := time.Now()
	event1 := &domain.Event{ID: "e1", Name: "Event 1", EventCode: "abc1"}

	tests := []struct {
		name      string
		eventRepo *mockEventRepository
		regRepo   *mockEventRegistrationRepository
		eventCode string
		userID    string
		wantErr   bool
		isNotFound bool
		wantID    string
	}{
		{
			name: "success new registration",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
				eventsByCode: map[string]*domain.Event{"abc1": event1},
			},
			regRepo: &mockEventRegistrationRepository{
				regsByUser: map[string][]*domain.EventRegistration{},
			},
			eventCode: "abc1",
			userID:    "u1",
			wantErr:   false,
			wantID:    "", // ID is set by repo on create; mock does not set it
		},
		{
			name: "success normalizes code to lowercase",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
				eventsByCode: map[string]*domain.Event{"abc1": event1},
			},
			regRepo: &mockEventRegistrationRepository{
				regsByUser: map[string][]*domain.EventRegistration{},
			},
			eventCode: "ABC1",
			userID:    "u1",
			wantErr:   false,
		},
		{
			name: "idempotent already registered",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
				eventsByCode: map[string]*domain.Event{"abc1": event1},
			},
			regRepo: &mockEventRegistrationRepository{
				regByEventAndUser: map[string]*domain.EventRegistration{
					"e1:u1": {ID: "r1", EventID: "e1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
				},
			},
			eventCode: "abc1",
			userID:    "u1",
			wantErr:   false,
			wantID:    "r1",
		},
		{
			name: "event not found",
			eventRepo: &mockEventRepository{
				events:       map[string]*domain.Event{},
				eventsByCode: map[string]*domain.Event{},
			},
			regRepo:   &mockEventRegistrationRepository{regsByUser: map[string][]*domain.EventRegistration{}},
			eventCode: "none",
			userID:    "u1",
			wantErr:   true,
			isNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &attendeeService{
				eventRepo:        tt.eventRepo,
				registrationRepo: tt.regRepo,
				sessionRepo:      &mockSessionRepository{},
			}
			got, created, err := svc.RegisterForEventByCode(context.Background(), tt.eventCode, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v (err=%v)", tt.wantErr, err != nil, err)
			}
			if tt.wantErr {
				if tt.isNotFound && !errors.Is(err, domain.ErrNotFound) {
					t.Fatalf("expected ErrNotFound, got %v", err)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil registration")
			}
			if tt.wantID != "" && got.ID != tt.wantID {
				t.Errorf("expected registration ID %q, got %q", tt.wantID, got.ID)
			}
			// idempotent case: created should be false when we had existing reg
			if tt.wantID == "r1" && created {
				t.Error("expected created=false when already registered")
			}
			if got.EventID != "" && got.EventID != "e1" && tt.wantID == "r1" {
				t.Errorf("expected EventID e1, got %s", got.EventID)
			}
		})
	}
}

func TestAttendeeService_GetEventSchedule(t *testing.T) {
	now := time.Now()
	event1 := &domain.Event{ID: "e1", Name: "Event 1", OwnerID: "owner1"}
	roomBookable := &domain.Room{ID: "r1", EventID: "e1", Name: "Room A", NotBookable: false, Capacity: 10}
	roomNotBookable := &domain.Room{ID: "r2", EventID: "e1", Name: "Room B", NotBookable: true, Capacity: 5}
	sess1 := &domain.Session{ID: "s1", RoomID: "r1", Title: "Talk 1", StartTime: now, EndTime: now.Add(time.Hour)}
	sess2 := &domain.Session{ID: "s2", RoomID: "r1", Title: "Talk 2", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour)}
	sess3 := &domain.Session{ID: "s3", RoomID: "r2", Title: "Talk in non-bookable", StartTime: now, EndTime: now.Add(time.Hour)}

	tests := []struct {
		name           string
		eventRepo      *mockEventRepository
		regRepo        *mockEventRegistrationRepository
		sessionRepo    *mockSessionRepository
		eventID        string
		userID         string
		wantErr        bool
		wantErrForbidden bool
		wantErrNotFound bool
		wantRoomCount  int
		wantSessionCountPerRoom map[string]int // room ID -> number of sessions
	}{
		{
			name: "owner gets schedule with only bookable rooms",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
			},
			regRepo: &mockEventRegistrationRepository{},
			sessionRepo: &mockSessionRepository{
				roomsByEvent: map[string][]*domain.Room{
					"e1": {roomBookable, roomNotBookable},
				},
				sessionsByEvent: map[string][]*domain.Session{
					"e1": {sess1, sess2, sess3},
				},
			},
			eventID: "e1",
			userID:  "owner1",
			wantErr: false,
			wantRoomCount: 1,
			wantSessionCountPerRoom: map[string]int{"r1": 2},
		},
		{
			name: "registered attendee gets schedule",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
			},
			regRepo: &mockEventRegistrationRepository{
				regByEventAndUser: map[string]*domain.EventRegistration{
					"e1:u1": {ID: "reg1", EventID: "e1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
				},
			},
			sessionRepo: &mockSessionRepository{
				roomsByEvent: map[string][]*domain.Room{
					"e1": {roomBookable},
				},
				sessionsByEvent: map[string][]*domain.Session{
					"e1": {sess1},
				},
			},
			eventID: "e1",
			userID:  "u1",
			wantErr: false,
			wantRoomCount: 1,
			wantSessionCountPerRoom: map[string]int{"r1": 1},
		},
		{
			name: "not registered and not owner returns forbidden",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
			},
			regRepo:     &mockEventRegistrationRepository{},
			sessionRepo: &mockSessionRepository{},
			eventID:     "e1",
			userID:      "other-user",
			wantErr:     true,
			wantErrForbidden: true,
		},
		{
			name: "event not found returns not found",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{},
			},
			regRepo:     &mockEventRegistrationRepository{},
			sessionRepo: &mockSessionRepository{},
			eventID:     "e-none",
			userID:      "owner1",
			wantErr:     true,
			wantErrNotFound: true,
		},
		{
			name: "no bookable rooms returns empty rooms slice",
			eventRepo: &mockEventRepository{
				events: map[string]*domain.Event{"e1": event1},
			},
			regRepo: &mockEventRegistrationRepository{
				regByEventAndUser: map[string]*domain.EventRegistration{
					"e1:u1": {ID: "reg1", EventID: "e1", UserID: "u1", CreatedAt: now, UpdatedAt: now},
				},
			},
			sessionRepo: &mockSessionRepository{
				roomsByEvent: map[string][]*domain.Room{
					"e1": {roomNotBookable},
				},
				sessionsByEvent: map[string][]*domain.Session{
					"e1": {sess3},
				},
			},
			eventID: "e1",
			userID:  "u1",
			wantErr: false,
			wantRoomCount: 0,
			wantSessionCountPerRoom: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &attendeeService{
				eventRepo:        tt.eventRepo,
				registrationRepo: tt.regRepo,
				sessionRepo:      tt.sessionRepo,
			}
			got, err := svc.GetEventSchedule(context.Background(), tt.eventID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v (err=%v)", tt.wantErr, err != nil, err)
			}
			if tt.wantErr {
				if tt.wantErrForbidden && !errors.Is(err, domain.ErrForbidden) {
					t.Fatalf("expected ErrForbidden, got %v", err)
				}
				if tt.wantErrNotFound && !errors.Is(err, domain.ErrNotFound) {
					t.Fatalf("expected ErrNotFound, got %v", err)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil schedule")
			}
			if got.Event == nil || got.Event.ID != tt.eventID {
				t.Errorf("expected event ID %q, got %v", tt.eventID, got.Event)
			}
			if len(got.Rooms) != tt.wantRoomCount {
				t.Errorf("expected %d rooms, got %d", tt.wantRoomCount, len(got.Rooms))
			}
			if tt.wantSessionCountPerRoom != nil {
				for _, rws := range got.Rooms {
					wantSess := tt.wantSessionCountPerRoom[rws.Room.ID]
					if wantSess != len(rws.Sessions) {
						t.Errorf("room %s: expected %d sessions, got %d", rws.Room.ID, wantSess, len(rws.Sessions))
					}
				}
			}
		})
	}
}

