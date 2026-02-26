package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	tagRepo             domain.TagRepository
	eventTeamMemberRepo domain.EventTeamMemberRepository
	userRepo            domain.UserRepository
	invitationRepo      domain.EventInvitationRepository
	emailService        domain.EmailService
	sf                  domain.SessionFetcher
	contextTimeout      time.Duration
}

func NewEventService(eventRepo domain.EventRepository,
	sessionRepo domain.SessionRepository,
	tagRepo domain.TagRepository,
	eventTeamMemberRepo domain.EventTeamMemberRepository,
	userRepo domain.UserRepository,
	invitationRepo domain.EventInvitationRepository,
	emailService domain.EmailService,
	sessionFetcher domain.SessionFetcher,
	timeout time.Duration,
) domain.EventService {
	return &eventService{
		eventRepo:           eventRepo,
		sessionRepo:         sessionRepo,
		tagRepo:             tagRepo,
		eventTeamMemberRepo: eventTeamMemberRepo,
		userRepo:            userRepo,
		invitationRepo:      invitationRepo,
		emailService:        emailService,
		sf:                  sessionFetcher,
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

	// Enrich sessions with speaker IDs for GET event response
	if len(sessions) > 0 {
		sessionIDs := make([]string, 0, len(sessions))
		for _, sess := range sessions {
			sessionIDs = append(sessionIDs, sess.ID)
		}
		speakerIDsBySession, err := s.sessionRepo.ListSpeakerIDsBySessionIDs(ctx, sessionIDs)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("list speaker IDs by session: %w", err)
		}
		for _, sess := range sessions {
			if ids, ok := speakerIDsBySession[sess.ID]; ok {
				sess.SpeakerIDs = ids
			} else {
				sess.SpeakerIDs = []string{}
			}
		}
	}

	return event, rooms, sessions, nil
}

func (s *eventService) UpdateEvent(ctx context.Context, eventID, ownerID string, date *time.Time, description *string, locationLat, locationLng *float64) (*domain.Event, error) {
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
	updated, err := s.eventRepo.Update(ctx, eventID, date, description, locationLat, locationLng)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update event: %w", err)
	}
	return updated, nil
}

// buildCategoryItemIDToName flattens All API categories into categoryItemID -> name.
func buildCategoryItemIDToName(categories []domain.SessionFetcherCategory) map[int]string {
	m := make(map[int]string)
	for _, cat := range categories {
		for _, item := range cat.Items {
			if item.Name != "" {
				m[item.ID] = item.Name
			}
		}
	}
	return m
}

// deriveTagsFromCategoryItems returns tag names for the given category item IDs (All API).
func deriveTagsFromCategoryItems(categoryItemIDs []int, idToName map[int]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, id := range categoryItemIDs {
		name := idToName[id]
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func (s *eventService) ImportSessionizeData(ctx context.Context, eventID string, sourceID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	// 1. Fetch data from Sessionize All API
	sessionData, err := s.sf.Fetch(ctx, sourceID)
	if err != nil {
		return err
	}

	// 2. Clear existing schedule and speakers
	if err := s.sessionRepo.DeleteScheduleByEventID(ctx, eventID); err != nil {
		return fmt.Errorf("failed to delete existing schedule: %w", err)
	}
	if err := s.sessionRepo.DeleteSpeakersByEventID(ctx, eventID); err != nil {
		return fmt.Errorf("failed to delete existing speakers: %w", err)
	}

	// 3. Insert rooms from flat list
	roomMap := make(map[int]string) // Sessionize room ID -> domain room ID
	for _, room := range sessionData.Rooms {
		now := time.Now()
		r := domain.NewRoom(eventID, room.Name, room.ID, "sessionize", false, 0, "", "", now, now)
		if err := s.sessionRepo.CreateRoom(ctx, r); err != nil {
			return fmt.Errorf("failed to create room %s: %w", room.Name, err)
		}
		roomMap[room.ID] = r.ID
	}

	// 4. Build category item ID -> name for tag derivation
	categoryIDToName := buildCategoryItemIDToName(sessionData.Categories)

	// 5. Insert sessions
	sessionMap := make(map[string]string) // Sessionize session ID -> domain session ID
	for _, sess := range sessionData.Sessions {
		domainRoomID, ok := roomMap[sess.RoomID]
		if !ok {
			continue // Skip session if room not found
		}
		tagNames := deriveTagsFromCategoryItems(sess.CategoryItems, categoryIDToName)
		now := time.Now()
		domainSess := domain.NewSession(domainRoomID, sess.ID, "sessionize", sess.Title, sess.Description, sess.StartsAt, sess.EndsAt, tagNames, now, now)
		if err := s.sessionRepo.CreateSession(ctx, domainSess); err != nil {
			return fmt.Errorf("failed to create session %s: %w", sess.Title, err)
		}
		var tagIDs []string
		for _, tagName := range tagNames {
			if tagName == "" {
				continue
			}
			tagID, err := s.tagRepo.EnsureTagForEvent(ctx, eventID, tagName)
			if err != nil {
				return fmt.Errorf("ensure tag %q for event: %w", tagName, err)
			}
			tagIDs = append(tagIDs, tagID)
		}
		if err := s.tagRepo.SetSessionTags(ctx, domainSess.ID, tagIDs); err != nil {
			return fmt.Errorf("failed to set session tags: %w", err)
		}
		sessionMap[sess.ID] = domainSess.ID
	}

	// 6. Insert speakers
	speakerMap := make(map[string]string) // Sessionize speaker UUID -> domain speaker ID
	for _, sp := range sessionData.Speakers {
		now := time.Now()
		domainSp := domain.NewSpeaker(eventID, sp.ID, "sessionize", sp.FirstName, sp.LastName, sp.FullName, sp.Bio, sp.TagLine, sp.ProfilePicture, sp.IsTopSpeaker, now, now)
		if err := s.sessionRepo.CreateSpeaker(ctx, domainSp); err != nil {
			return fmt.Errorf("failed to create speaker %s: %w", sp.FullName, err)
		}
		speakerMap[sp.ID] = domainSp.ID
	}

	// 7. Link sessions to speakers
	for _, sess := range sessionData.Sessions {
		domainSessionID, ok := sessionMap[sess.ID]
		if !ok {
			continue
		}
		for _, speakerUUID := range sess.Speakers {
			domainSpeakerID, ok := speakerMap[speakerUUID]
			if !ok {
				continue
			}
			if err := s.sessionRepo.CreateSessionSpeaker(ctx, domainSessionID, domainSpeakerID); err != nil {
				return fmt.Errorf("failed to link session to speaker: %w", err)
			}
		}
	}

	return nil
}

func generateManualSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "manual-" + hex.EncodeToString(b), nil
}

func (s *eventService) CreateEventSession(
	ctx context.Context,
	eventID, ownerID, roomID, title, description string,
	startTime, endTime time.Time,
	tagNames, speakerIDs []string,
) (*domain.Session, error) {
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

	if !endTime.After(startTime) {
		return nil, fmt.Errorf("end_time must be after start_time: %w", domain.ErrInvalidInput)
	}

	sourceSessionID, err := generateManualSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate manual session id: %w", err)
	}

	now := time.Now()
	sess := domain.NewSession(roomID, sourceSessionID, "admin_app", title, description, startTime, endTime, nil, now, now)
	if err := s.sessionRepo.CreateSession(ctx, sess); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("create session: %w", err)
	}

	var tagIDs []string
	for _, tagName := range tagNames {
		name := strings.TrimSpace(tagName)
		if name == "" {
			continue
		}
		tagID, err := s.tagRepo.EnsureTagForEvent(ctx, eventID, name)
		if err != nil {
			return nil, fmt.Errorf("ensure tag %q for event: %w", name, err)
		}
		tagIDs = append(tagIDs, tagID)
	}
	if len(tagIDs) > 0 {
		if err := s.tagRepo.SetSessionTags(ctx, sess.ID, tagIDs); err != nil {
			return nil, fmt.Errorf("set session tags: %w", err)
		}
	}

	for _, speakerID := range speakerIDs {
		id := strings.TrimSpace(speakerID)
		if id == "" {
			continue
		}
		sp, err := s.sessionRepo.GetSpeakerByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ErrNotFound
			}
			return nil, fmt.Errorf("get speaker: %w", err)
		}
		if sp.EventID != eventID {
			return nil, fmt.Errorf("speaker does not belong to event: %w", domain.ErrInvalidInput)
		}
		if err := s.sessionRepo.CreateSessionSpeaker(ctx, sess.ID, id); err != nil {
			return nil, fmt.Errorf("link session to speaker: %w", err)
		}
	}

	created, err := s.sessionRepo.GetSessionByID(ctx, sess.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	return created, nil
}

func (s *eventService) UpdateSessionSchedule(ctx context.Context, eventID, sessionID, ownerID string, roomID *string, startTime, endTime *time.Time) (*domain.Session, error) {
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

	sess, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Ensure current session belongs to this event via its room.
	currentRoom, err := s.sessionRepo.GetRoomByID(ctx, sess.RoomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get room: %w", err)
	}
	if currentRoom.EventID != eventID {
		return nil, domain.ErrNotFound
	}

	newRoomID := sess.RoomID
	if roomID != nil {
		newRoomID = *roomID
		// Validate new room belongs to the same event.
		newRoom, err := s.sessionRepo.GetRoomByID(ctx, newRoomID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ErrNotFound
			}
			return nil, fmt.Errorf("get room: %w", err)
		}
		if newRoom.EventID != eventID {
			return nil, domain.ErrNotFound
		}
	}

	newStart := sess.StartTime
	if startTime != nil {
		newStart = *startTime
	}
	newEnd := sess.EndTime
	if endTime != nil {
		newEnd = *endTime
	}

	if !newEnd.After(newStart) {
		return nil, domain.ErrInvalidInput
	}

	var roomIDArg *string
	if roomID != nil {
		roomIDArg = &newRoomID
	}
	var startArg *time.Time
	if startTime != nil {
		startArg = &newStart
	}
	var endArg *time.Time
	if endTime != nil {
		endArg = &newEnd
	}

	updated, err := s.sessionRepo.UpdateSessionSchedule(ctx, sessionID, roomIDArg, startArg, endArg)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update session schedule: %w", err)
	}

	return updated, nil
}

func (s *eventService) UpdateSessionContent(ctx context.Context, eventID, sessionID, ownerID string, title *string, description *string) (*domain.Session, error) {
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

	sess, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	currentRoom, err := s.sessionRepo.GetRoomByID(ctx, sess.RoomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get room: %w", err)
	}
	if currentRoom.EventID != eventID {
		return nil, domain.ErrNotFound
	}

	updated, err := s.sessionRepo.UpdateSessionContent(ctx, sessionID, title, description)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update session content: %w", err)
	}

	return updated, nil
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

func (s *eventService) CreateEventRoom(ctx context.Context, eventID, ownerID, name string, capacity int, description, howToGetThere string, notBookable bool) (*domain.Room, error) {
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

	now := time.Now()
	room := domain.NewRoom(eventID, name, 0, "admin_app", notBookable, capacity, description, howToGetThere, now, now)
	if err := s.sessionRepo.CreateRoom(ctx, room); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("create room: %w", err)
	}
	return room, nil
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

func (s *eventService) ListEventRooms(ctx context.Context, eventID, ownerID string) ([]*domain.Room, error) {
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
	rooms, err := s.sessionRepo.ListRoomsByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	if rooms == nil {
		rooms = []*domain.Room{}
	}
	return rooms, nil
}

func (s *eventService) GetEventRoom(ctx context.Context, eventID, roomID, ownerID string) (*domain.Room, error) {
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
	return room, nil
}

func (s *eventService) UpdateEventRoom(ctx context.Context, eventID, roomID, ownerID string, capacity int, description, howToGetThere string, notBookable *bool) (*domain.Room, error) {
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
	finalNotBookable := room.NotBookable
	if notBookable != nil {
		finalNotBookable = *notBookable
	}
	updated, err := s.sessionRepo.UpdateRoomDetails(ctx, roomID, capacity, description, howToGetThere, finalNotBookable)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update room details: %w", err)
	}
	return updated, nil
}

func (s *eventService) DeleteEventRoom(ctx context.Context, eventID, roomID, ownerID string) error {
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
	room, err := s.sessionRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get room: %w", err)
	}
	if room.EventID != eventID {
		return domain.ErrNotFound
	}
	if err := s.sessionRepo.DeleteRoom(ctx, roomID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete room: %w", err)
	}
	return nil
}

func (s *eventService) DeleteEventSession(ctx context.Context, eventID, sessionID, ownerID string) error {
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
	sess, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get session: %w", err)
	}
	room, err := s.sessionRepo.GetRoomByID(ctx, sess.RoomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get room: %w", err)
	}
	if room.EventID != eventID {
		return domain.ErrNotFound
	}
	if err := s.sessionRepo.DeleteSession(ctx, sessionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *eventService) ListEventSpeakers(ctx context.Context, eventID, ownerID string) ([]*domain.Speaker, error) {
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
	speakers, err := s.sessionRepo.ListSpeakersByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("list speakers: %w", err)
	}
	if speakers == nil {
		speakers = []*domain.Speaker{}
	}
	return speakers, nil
}

func (s *eventService) GetEventSpeaker(ctx context.Context, eventID, speakerID, ownerID string) (*domain.Speaker, []*domain.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != ownerID {
		return nil, nil, domain.ErrForbidden
	}
	speaker, err := s.sessionRepo.GetSpeakerByID(ctx, speakerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, fmt.Errorf("get speaker: %w", err)
	}
	if speaker.EventID != eventID {
		return nil, nil, domain.ErrNotFound
	}
	sessionIDs, err := s.sessionRepo.ListSessionIDsBySpeakerID(ctx, speakerID)
	if err != nil {
		return nil, nil, fmt.Errorf("list session IDs by speaker: %w", err)
	}
	var sessions []*domain.Session
	if len(sessionIDs) > 0 {
		sessions, err = s.sessionRepo.ListSessionsByIDs(ctx, sessionIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("list sessions: %w", err)
		}
	}
	return speaker, sessions, nil
}

func (s *eventService) DeleteEventSpeaker(ctx context.Context, eventID, speakerID, ownerID string) error {
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
	speaker, err := s.sessionRepo.GetSpeakerByID(ctx, speakerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get speaker: %w", err)
	}
	if speaker.EventID != eventID {
		return domain.ErrNotFound
	}
	if err := s.sessionRepo.DeleteSpeaker(ctx, speakerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete speaker: %w", err)
	}
	return nil
}

func generateManualSpeakerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "manual-" + hex.EncodeToString(b), nil
}

func (s *eventService) CreateEventSpeaker(ctx context.Context, eventID, ownerID string, firstName, lastName, fullName, bio, tagLine, profilePicture string, isTopSpeaker bool) (*domain.Speaker, error) {
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
	sessionizeSpeakerID, err := generateManualSpeakerID()
	if err != nil {
		return nil, fmt.Errorf("generate manual speaker id: %w", err)
	}
	if fullName == "" {
		fullName = strings.TrimSpace(firstName + " " + lastName)
	}
	now := time.Now()
	speaker := domain.NewSpeaker(eventID, sessionizeSpeakerID, "admin_app", firstName, lastName, fullName, bio, tagLine, profilePicture, isTopSpeaker, now, now)
	if err := s.sessionRepo.CreateSpeaker(ctx, speaker); err != nil {
		return nil, fmt.Errorf("create speaker: %w", err)
	}
	return speaker, nil
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

func (s *eventService) ListEventInvitations(ctx context.Context, eventID, callerID string, search string, params domain.PaginationParams) ([]*domain.EventInvitation, int, error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, 0, domain.ErrNotFound
		}
		return nil, 0, fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != callerID {
		return nil, 0, domain.ErrForbidden
	}
	invs, total, err := s.invitationRepo.ListByEventID(ctx, eventID, search, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list event invitations: %w", err)
	}
	if invs == nil {
		invs = []*domain.EventInvitation{}
	}
	return invs, total, nil
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

func (s *eventService) ListEventTags(ctx context.Context, eventID, callerID string) ([]*domain.Tag, error) {
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
	tags, err := s.tagRepo.ListTagsByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("list event tags: %w", err)
	}
	if tags == nil {
		tags = []*domain.Tag{}
	}
	return tags, nil
}

func (s *eventService) SendEventInvitations(ctx context.Context, eventID, ownerID string, emails []string) (sent int, failed []string, err error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, nil, domain.ErrNotFound
		}
		return 0, nil, fmt.Errorf("get event: %w", err)
	}
	if event.OwnerID != ownerID {
		return 0, nil, domain.ErrForbidden
	}

	owner, err := s.userRepo.GetByID(ctx, ownerID)
	if err != nil || owner == nil {
		// Proceed with empty owner name if user lookup fails
	}
	ownerName := "Event owner"
	if owner != nil {
		ownerName = strings.TrimSpace(owner.Name + " " + owner.LastName)
		if ownerName == "" {
			ownerName = owner.Email
		}
		if ownerName == "" {
			ownerName = "Event owner"
		}
	}

	for _, email := range emails {
		email = strings.TrimSpace(strings.ToLower(email))
		if email == "" {
			continue
		}
		sentAt := time.Now()
		inv := &domain.EventInvitation{
			EventID: eventID,
			Email:   email,
			SentAt:  sentAt,
		}
		if err := s.invitationRepo.Create(ctx, inv); err != nil {
			failed = append(failed, email)
			continue
		}
		data := &domain.EventInvitationEmailData{
			Email:     email,
			OwnerName: ownerName,
			EventName: event.Name,
			EventCode: event.EventCode,
		}
		if err := s.emailService.SendEventInvitation(ctx, data); err != nil {
			failed = append(failed, email)
			continue
		}
		sent++
	}
	return sent, failed, nil
}
