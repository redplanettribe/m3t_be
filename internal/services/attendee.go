package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"multitrackticketing/internal/domain"
)

type attendeeService struct {
	eventRepo         domain.EventRepository
	registrationRepo  domain.EventRegistrationRepository
}

// NewAttendeeService creates an AttendeeService with the given repositories.
func NewAttendeeService(
	eventRepo domain.EventRepository,
	registrationRepo domain.EventRegistrationRepository,
) domain.AttendeeService {
	return &attendeeService{
		eventRepo:        eventRepo,
		registrationRepo: registrationRepo,
	}
}

func (s *attendeeService) RegisterForEvent(ctx context.Context, eventID, userID string) (*domain.EventRegistration, bool, error) {
	// Ensure the event exists.
	if _, err := s.eventRepo.GetByID(ctx, eventID); err != nil {
		if err == domain.ErrNotFound {
			return nil, false, domain.ErrNotFound
		}
		return nil, false, fmt.Errorf("get event: %w", err)
	}

	// Check if the user is already registered; make registration idempotent.
	if existing, err := s.registrationRepo.GetByEventAndUser(ctx, eventID, userID); err == nil {
		return existing, false, nil
	} else if err != domain.ErrNotFound {
		return nil, false, fmt.Errorf("get event registration: %w", err)
	}

	now := time.Now()
	reg := domain.NewEventRegistration(eventID, userID, now, now)
	if err := s.registrationRepo.Create(ctx, reg); err != nil {
		return nil, false, fmt.Errorf("create event registration: %w", err)
	}
	return reg, true, nil
}

func (s *attendeeService) RegisterForEventByCode(ctx context.Context, eventCode, userID string) (*domain.EventRegistration, bool, error) {
	code := strings.ToLower(strings.TrimSpace(eventCode))
	event, err := s.eventRepo.GetByEventCode(ctx, code)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, false, domain.ErrNotFound
		}
		return nil, false, fmt.Errorf("get event by code: %w", err)
	}

	if existing, err := s.registrationRepo.GetByEventAndUser(ctx, event.ID, userID); err == nil {
		return existing, false, nil
	} else if err != domain.ErrNotFound {
		return nil, false, fmt.Errorf("get event registration: %w", err)
	}

	now := time.Now()
	reg := domain.NewEventRegistration(event.ID, userID, now, now)
	if err := s.registrationRepo.Create(ctx, reg); err != nil {
		return nil, false, fmt.Errorf("create event registration: %w", err)
	}
	return reg, true, nil
}

func (s *attendeeService) ListMyRegisteredEvents(ctx context.Context, userID string) ([]*domain.EventRegistrationWithEvent, error) {
	regs, err := s.registrationRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list registrations: %w", err)
	}
	if len(regs) == 0 {
		return []*domain.EventRegistrationWithEvent{}, nil
	}

	// Fetch events one by one (N+1). This keeps the implementation simple; we can optimize later if needed.
	eventsByID := make(map[string]*domain.Event)
	var result []*domain.EventRegistrationWithEvent

	for _, reg := range regs {
		ev, ok := eventsByID[reg.EventID]
		if !ok {
			ev, err = s.eventRepo.GetByID(ctx, reg.EventID)
			if err != nil {
				if err == domain.ErrNotFound {
					// Event deleted but registration remains; skip this entry defensively.
					continue
				}
				return nil, fmt.Errorf("get event for registration: %w", err)
			}
			eventsByID[reg.EventID] = ev
		}
		result = append(result, &domain.EventRegistrationWithEvent{
			Registration: reg,
			Event:        ev,
		})
	}

	if result == nil {
		result = []*domain.EventRegistrationWithEvent{}
	}
	return result, nil
}

