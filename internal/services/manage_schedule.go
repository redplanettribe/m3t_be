package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"multitrackticketing/internal/domain"
)

type manageScheduleService struct {
	eventRepo      domain.EventRepository
	sessionRepo    domain.SessionRepository
	sessionize     domain.SessionizeFetcher
	contextTimeout time.Duration
}

func NewManageScheduleService(eventRepo domain.EventRepository, sessionRepo domain.SessionRepository, sessionize domain.SessionizeFetcher, timeout time.Duration) domain.ManageScheduleService {
	return &manageScheduleService{
		eventRepo:      eventRepo,
		sessionRepo:    sessionRepo,
		sessionize:     sessionize,
		contextTimeout: timeout,
	}
}

func (s *manageScheduleService) CreateEvent(ctx context.Context, event *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	if event.OwnerID == "" {
		return fmt.Errorf("event owner is required")
	}

	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	return s.eventRepo.Create(ctx, event)
}

func (s *manageScheduleService) GetEventByID(ctx context.Context, eventID string) (*domain.Event, []*domain.Room, []*domain.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, nil, domain.ErrNotFound
		}
		return nil, nil, nil, fmt.Errorf("get event: %w", err)
	}

	rooms, err := s.sessionRepo.ListRoomsByEventID(ctx, eventID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list rooms: %w", err)
	}
	if rooms == nil {
		rooms = []*domain.Room{}
	}

	sessions, err := s.sessionRepo.ListSessionsByEventID(ctx, eventID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list sessions: %w", err)
	}
	if sessions == nil {
		sessions = []*domain.Session{}
	}

	return event, rooms, sessions, nil
}

// deriveTags collects all category item names from Sessionize categories, deduped.
func deriveTags(categories []domain.SessionizeCategory) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, cat := range categories {
		for _, item := range cat.CategoryItems {
			if item.Name == "" {
				continue
			}
			if _, ok := seen[item.Name]; ok {
				continue
			}
			seen[item.Name] = struct{}{}
			out = append(out, item.Name)
		}
	}
	return out
}

func (s *manageScheduleService) ImportSessionizeData(ctx context.Context, eventID string, sessionizeID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	// 1. Fetch data from Sessionize
	sessionizeData, err := s.sessionize.Fetch(ctx, sessionizeID)
	if err != nil {
		return err
	}

	// 2. Clear existing schedule
	if err := s.sessionRepo.DeleteScheduleByEventID(ctx, eventID); err != nil {
		return fmt.Errorf("failed to delete existing schedule: %w", err)
	}

	// 3. Import new data
	// Group rooms by ID to avoid duplicates across dates (if any)
	uniqueRooms := make(map[int]string) // id -> name

	for _, grid := range sessionizeData {
		for _, room := range grid.Rooms {
			uniqueRooms[room.ID] = room.Name
		}
	}

	// Insert Rooms
	roomMap := make(map[int]string) // sessionize_id -> domain_id
	for sID, name := range uniqueRooms {
		now := time.Now()
		r := domain.NewRoom(eventID, name, sID, now, now)
		if err := s.sessionRepo.CreateRoom(ctx, r); err != nil {
			return fmt.Errorf("failed to create room %s: %w", name, err)
		}
		roomMap[sID] = r.ID
	}

	// Insert Sessions
	for _, grid := range sessionizeData {
		for _, room := range grid.Rooms {
			domainRoomID, ok := roomMap[room.ID]
			if !ok {
				continue // Should not happen
			}

			for _, session := range room.Sessions {
				desc := ""
				if session.Description != nil {
					desc = *session.Description
				}
				tags := deriveTags(session.Categories)
				now := time.Now()
				sess := domain.NewSession(domainRoomID, session.ID, session.Title, desc, session.StartsAt, session.EndsAt, tags, now, now)

				if err := s.sessionRepo.CreateSession(ctx, sess); err != nil {
					return fmt.Errorf("failed to create session %s: %w", session.Title, err)
				}
			}
		}
	}

	return nil
}

func (s *manageScheduleService) ListEventsByOwner(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()
	return s.eventRepo.ListByOwnerID(ctx, ownerID)
}
