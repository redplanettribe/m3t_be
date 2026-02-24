package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"multitrackticketing/internal/domain"
)

type mockEventRegistrationRepository struct {
	regsByUser map[string][]*domain.EventRegistration
	err        error
}

func (m *mockEventRegistrationRepository) Create(ctx context.Context, reg *domain.EventRegistration) error {
	return nil
}

func (m *mockEventRegistrationRepository) GetByEventAndUser(ctx context.Context, eventID, userID string) (*domain.EventRegistration, error) {
	return nil, domain.ErrNotFound
}

func (m *mockEventRegistrationRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.EventRegistration, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.regsByUser[userID], nil
}

type mockEventRepository struct {
	events map[string]*domain.Event
	err    error
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

func (m *mockEventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
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

