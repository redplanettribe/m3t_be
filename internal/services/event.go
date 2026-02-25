package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"multitrackticketing/internal/domain"
)

type eventService struct {
	eventRepo           domain.EventRepository
	sessionRepo         domain.SessionRepository
	eventTeamMemberRepo domain.EventTeamMemberRepository
	userRepo            domain.UserRepository
	sessionize          domain.SessionizeFetcher
	contextTimeout      time.Duration
}

func NewEventService(eventRepo domain.EventRepository,
	sessionRepo domain.SessionRepository,
	eventTeamMemberRepo domain.EventTeamMemberRepository,
	userRepo domain.UserRepository,
	sessionize domain.SessionizeFetcher,
	timeout time.Duration,
) domain.EventService {
	return &eventService{
		eventRepo:           eventRepo,
		sessionRepo:         sessionRepo,
		eventTeamMemberRepo: eventTeamMemberRepo,
		userRepo:            userRepo,
		sessionize:          sessionize,
		contextTimeout:      timeout,
	}
}

func (s *eventService) CreateEvent(ctx context.Context, event *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	if event.OwnerID == "" {
		return fmt.Errorf("event owner is required")
	}

	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	if event.EventCode == "" {
		code, err := generateEventCode()
		if err != nil {
			return fmt.Errorf("generate event code: %w", err)
		}
		event.EventCode = code
	}

	return s.eventRepo.Create(ctx, event)
}

const eventCodeLength = 4

var eventCodeAlphabet = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func generateEventCode() (string, error) {
	b := make([]rune, eventCodeLength)
	max := big.NewInt(int64(len(eventCodeAlphabet)))
	for i := 0; i < eventCodeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = eventCodeAlphabet[n.Int64()]
	}
	return string(b), nil
}

func (s *eventService) GetEventByID(ctx context.Context, eventID string) (*domain.Event, []*domain.Room, []*domain.Session, error) {
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

func (s *eventService) ImportSessionizeData(ctx context.Context, eventID string, sessionizeID string) error {
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
		r := domain.NewRoom(eventID, name, sID, false, now, now)
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

func (s *eventService) ListEventsByOwner(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()
	return s.eventRepo.ListByOwnerID(ctx, ownerID)
}

func (s *eventService) DeleteEvent(ctx context.Context, eventID string, ownerID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != ownerID {
		return domain.ErrForbidden
	}
	if err := s.eventRepo.Delete(ctx, eventID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

func (s *eventService) ToggleRoomNotBookable(ctx context.Context, eventID, roomID, ownerID string) (*domain.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != ownerID {
		return nil, domain.ErrForbidden
	}

	room, err := s.sessionRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get room: %w", err)
	}
	if room.EventID != eventID {
		return nil, domain.ErrNotFound
	}

	updated, err := s.sessionRepo.SetRoomNotBookable(ctx, roomID, !room.NotBookable)
	if err != nil {
		return nil, fmt.Errorf("set room not_bookable: %w", err)
	}
	return updated, nil
}

func (s *eventService) AddEventTeamMember(ctx context.Context, eventID, userIDToAdd, ownerID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != ownerID {
		return domain.ErrForbidden
	}
	if userIDToAdd == event.OwnerID {
		return domain.ErrInvalidInput
	}
	if err := s.eventTeamMemberRepo.Add(ctx, eventID, userIDToAdd); err != nil {
		if errors.Is(err, domain.ErrAlreadyMember) {
			return domain.ErrAlreadyMember
		}
		return fmt.Errorf("add team member: %w", err)
	}
	return nil
}

func (s *eventService) AddEventTeamMemberByEmail(ctx context.Context, eventID, email, ownerID string) (*domain.EventTeamMember, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, domain.ErrInvalidInput
	}
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	if err := s.AddEventTeamMember(ctx, eventID, user.ID, ownerID); err != nil {
		return nil, err
	}
	return &domain.EventTeamMember{
		EventID:  eventID,
		UserID:   user.ID,
		Name:     user.Name,
		LastName: user.LastName,
		Email:    user.Email,
	}, nil
}

func (s *eventService) ListEventTeamMembers(ctx context.Context, eventID, callerID string) ([]*domain.EventTeamMember, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != callerID {
		return nil, domain.ErrForbidden
	}
	members, err := s.eventTeamMemberRepo.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	if members == nil {
		members = []*domain.EventTeamMember{}
	}
	return members, nil
}

func (s *eventService) RemoveEventTeamMember(ctx context.Context, eventID, userIDToRemove, ownerID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != ownerID {
		return domain.ErrForbidden
	}
	if err := s.eventTeamMemberRepo.Remove(ctx, eventID, userIDToRemove); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("remove team member: %w", err)
	}
	return nil
}
