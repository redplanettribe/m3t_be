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

