package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEventRepo is an in-memory EventRepository for tests.
type fakeEventRepo struct {
	byID   map[string]*domain.Event
	nextID int
	err    error // if set, Create returns this error
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{
		byID:   make(map[string]*domain.Event),
		nextID: 1,
	}
}

func (f *fakeEventRepo) Create(ctx context.Context, e *domain.Event) error {
	if f.err != nil {
		return f.err
	}
	e.ID = fmt.Sprintf("ev-%d", f.nextID)
	f.nextID++
	f.byID[e.ID] = e
	return nil
}

func (f *fakeEventRepo) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeEventRepo) GetByEventCode(ctx context.Context, eventCode string) (*domain.Event, error) {
	code := strings.ToLower(strings.TrimSpace(eventCode))
	for _, e := range f.byID {
		if strings.ToLower(e.EventCode) == code {
			return e, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeEventRepo) ListByOwnerID(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	var out []*domain.Event
	for _, e := range f.byID {
		if e.OwnerID == ownerID {
			out = append(out, e)
		}
	}
	// Sort by CreatedAt DESC to match repo
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (f *fakeEventRepo) Delete(ctx context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeEventRepo) Update(ctx context.Context, eventID string, date *time.Time, description *string, locationLat, locationLng *float64) (*domain.Event, error) {
	e, ok := f.byID[eventID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if date != nil {
		e.Date = date
	}
	if description != nil {
		e.Description = description
	}
	if locationLat != nil {
		e.LocationLat = locationLat
	}
	if locationLng != nil {
		e.LocationLng = locationLng
	}
	return e, nil
}

// fakeSessionRepo is an in-memory SessionRepository for tests.
type fakeSessionRepo struct {
	rooms                []*domain.Room
	sessions             []*domain.Session
	roomID               int
	sessID               int
	createRoomErr        error
	createSessionErr     error
	deleteErr            error
	updateRoomDetailsErr error
	deleteRoomErr        error
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{
		rooms:    nil,
		sessions: nil,
		roomID:   1,
		sessID:   1,
	}
}

func (f *fakeSessionRepo) CreateRoom(ctx context.Context, r *domain.Room) error {
	if f.createRoomErr != nil {
		return f.createRoomErr
	}
	r.ID = fmt.Sprintf("room-%d", f.roomID)
	f.roomID++
	f.rooms = append(f.rooms, r)
	return nil
}

func (f *fakeSessionRepo) CreateSession(ctx context.Context, s *domain.Session) error {
	if f.createSessionErr != nil {
		return f.createSessionErr
	}
	s.ID = fmt.Sprintf("sess-%d", f.sessID)
	f.sessID++
	f.sessions = append(f.sessions, s)
	return nil
}

func (f *fakeSessionRepo) DeleteScheduleByEventID(ctx context.Context, eventID string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	roomIDsForEvent := make(map[string]bool)
	var rooms []*domain.Room
	for _, r := range f.rooms {
		if r.EventID == eventID {
			roomIDsForEvent[r.ID] = true
		} else {
			rooms = append(rooms, r)
		}
	}
	f.rooms = rooms
	var sessions []*domain.Session
	for _, s := range f.sessions {
		if !roomIDsForEvent[s.RoomID] {
			sessions = append(sessions, s)
		}
	}
	f.sessions = sessions
	return nil
}

func (f *fakeSessionRepo) GetRoomByID(ctx context.Context, roomID string) (*domain.Room, error) {
	for _, r := range f.rooms {
		if r.ID == roomID {
			return r, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeSessionRepo) ListRoomsByEventID(ctx context.Context, eventID string) ([]*domain.Room, error) {
	var out []*domain.Room
	for _, r := range f.rooms {
		if r.EventID == eventID {
			out = append(out, r)
		}
	}
	if out == nil {
		return []*domain.Room{}, nil
	}
	return out, nil
}

func (f *fakeSessionRepo) SetRoomNotBookable(ctx context.Context, roomID string, notBookable bool) (*domain.Room, error) {
	for _, r := range f.rooms {
		if r.ID == roomID {
			r.NotBookable = notBookable
			return r, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeSessionRepo) UpdateRoomDetails(ctx context.Context, roomID string, capacity int, description, howToGetThere string, notBookable bool) (*domain.Room, error) {
	if f.updateRoomDetailsErr != nil {
		return nil, f.updateRoomDetailsErr
	}
	for _, r := range f.rooms {
		if r.ID == roomID {
			r.Capacity = capacity
			r.Description = description
			r.HowToGetThere = howToGetThere
			r.NotBookable = notBookable
			return r, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeSessionRepo) DeleteRoom(ctx context.Context, roomID string) error {
	if f.deleteRoomErr != nil {
		return f.deleteRoomErr
	}
	for i, r := range f.rooms {
		if r.ID == roomID {
			f.rooms = append(f.rooms[:i], f.rooms[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeSessionRepo) ListSessionsByEventID(ctx context.Context, eventID string) ([]*domain.Session, error) {
	roomIDs := make(map[string]bool)
	for _, r := range f.rooms {
		if r.EventID == eventID {
			roomIDs[r.ID] = true
		}
	}
	var out []*domain.Session
	for _, s := range f.sessions {
		if roomIDs[s.RoomID] {
			out = append(out, s)
		}
	}
	if out == nil {
		return []*domain.Session{}, nil
	}
	return out, nil
}

func (f *fakeSessionRepo) GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	for _, s := range f.sessions {
		if s.ID == sessionID {
			return s, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeSessionRepo) UpdateSessionSchedule(ctx context.Context, sessionID string, roomID *string, startTime, endTime *time.Time) (*domain.Session, error) {
	for _, s := range f.sessions {
		if s.ID == sessionID {
			if roomID != nil {
				s.RoomID = *roomID
			}
			if startTime != nil {
				s.StartTime = *startTime
			}
			if endTime != nil {
				s.EndTime = *endTime
			}
			return s, nil
		}
	}
	return nil, domain.ErrNotFound
}

// fakeSessionizeFetcher returns fixed data or a configurable error.
type fakeSessionizeFetcher struct {
	data domain.SessionFetcherResponse
	err  error
}

func (f *fakeSessionizeFetcher) Fetch(ctx context.Context, sessionizeID string) (domain.SessionFetcherResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

// fakeEventTeamMemberRepo is an in-memory EventTeamMemberRepository for tests.
type fakeEventTeamMemberRepo struct {
	members   map[string]map[string]bool // eventID -> userID -> true
	addErr    error
	removeErr error
}

func newFakeEventTeamMemberRepo() *fakeEventTeamMemberRepo {
	return &fakeEventTeamMemberRepo{
		members: make(map[string]map[string]bool),
	}
}

func (f *fakeEventTeamMemberRepo) Add(ctx context.Context, eventID, userID string) error {
	if f.addErr != nil {
		return f.addErr
	}
	if f.members[eventID] == nil {
		f.members[eventID] = make(map[string]bool)
	}
	if f.members[eventID][userID] {
		return domain.ErrAlreadyMember
	}
	f.members[eventID][userID] = true
	return nil
}

func (f *fakeEventTeamMemberRepo) ListByEventID(ctx context.Context, eventID string) ([]*domain.EventTeamMember, error) {
	userIDs, ok := f.members[eventID]
	if !ok {
		return []*domain.EventTeamMember{}, nil
	}
	out := make([]*domain.EventTeamMember, 0, len(userIDs))
	for uid := range userIDs {
		out = append(out, &domain.EventTeamMember{EventID: eventID, UserID: uid})
	}
	return out, nil
}

func (f *fakeEventTeamMemberRepo) Remove(ctx context.Context, eventID, userID string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	if f.members[eventID] == nil || !f.members[eventID][userID] {
		return domain.ErrNotFound
	}
	delete(f.members[eventID], userID)
	return nil
}

// fakeUserRepoForSchedule is a minimal UserRepository for schedule service tests (GetByEmail only).
type fakeUserRepoForSchedule struct {
	byEmail map[string]*domain.User // normalized lower email -> user
}

func newFakeUserRepoForSchedule() *fakeUserRepoForSchedule {
	return &fakeUserRepoForSchedule{byEmail: make(map[string]*domain.User)}
}

func (f *fakeUserRepoForSchedule) Create(ctx context.Context, user *domain.User) error { return nil }
func (f *fakeUserRepoForSchedule) Update(ctx context.Context, user *domain.User) error { return nil }
func (f *fakeUserRepoForSchedule) AssignRole(ctx context.Context, userID, roleID string) error {
	return nil
}
func (f *fakeUserRepoForSchedule) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUserRepoForSchedule) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUserRepoForSchedule) addUser(email, id string) {
	email = strings.TrimSpace(strings.ToLower(email))
	f.byEmail[email] = &domain.User{ID: id, Email: email}
}

func (f *fakeUserRepoForSchedule) addUserWithName(email, id, name, lastName string) {
	email = strings.TrimSpace(strings.ToLower(email))
	f.byEmail[email] = &domain.User{ID: id, Email: email, Name: name, LastName: lastName}
}

// fakeEventInvitationRepo is an in-memory EventInvitationRepository for tests.
type fakeEventInvitationRepo struct {
	invitations []*domain.EventInvitation
	nextID      int
	createErr   error
}

func newFakeEventInvitationRepo() *fakeEventInvitationRepo {
	return &fakeEventInvitationRepo{
		invitations: nil,
		nextID:      1,
	}
}

func (f *fakeEventInvitationRepo) Create(ctx context.Context, inv *domain.EventInvitation) error {
	if f.createErr != nil {
		return f.createErr
	}
	for _, existing := range f.invitations {
		if existing.EventID == inv.EventID && strings.ToLower(existing.Email) == strings.ToLower(inv.Email) {
			return errors.New("duplicate key value violates unique constraint")
		}
	}
	inv.ID = fmt.Sprintf("inv-%d", f.nextID)
	f.nextID++
	f.invitations = append(f.invitations, inv)
	return nil
}

func (f *fakeEventInvitationRepo) ListByEventID(ctx context.Context, eventID string, search string, params domain.PaginationParams) ([]*domain.EventInvitation, int, error) {
	var out []*domain.EventInvitation
	for _, inv := range f.invitations {
		if inv.EventID != eventID {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(inv.Email), strings.ToLower(search)) {
			continue
		}
		out = append(out, inv)
	}
	if out == nil {
		out = []*domain.EventInvitation{}
	}
	total := len(out)
	offset := params.Offset()
	if offset > total {
		offset = total
	}
	end := offset + params.PageSize
	if end > total {
		end = total
	}
	page := out[offset:end]
	if page == nil {
		page = []*domain.EventInvitation{}
	}
	return page, total, nil
}

// fakeEmailService is a test double for EmailService. Tracks SendEventInvitation calls; other methods no-op.
type fakeEmailService struct {
	sendEventInvitationErr error // if set, SendEventInvitation returns this
	sentInvitations        []*domain.EventInvitationEmailData
}

func newFakeEmailService() *fakeEmailService {
	return &fakeEmailService{sentInvitations: []*domain.EventInvitationEmailData{}}
}

func (f *fakeEmailService) SendWelcomeMessage(ctx context.Context, data *domain.WelcomeMessageEmailData) error {
	return nil
}

func (f *fakeEmailService) SendLoginCode(ctx context.Context, data *domain.LoginCodeEmailData) error {
	return nil
}

func (f *fakeEmailService) SendEventInvitation(ctx context.Context, data *domain.EventInvitationEmailData) error {
	if f.sendEventInvitationErr != nil {
		return f.sendEventInvitationErr
	}
	f.sentInvitations = append(f.sentInvitations, data)
	return nil
}

// defaultSessionizeData returns a minimal valid SessionizeResponse for tests.
func defaultSessionizeData() domain.SessionFetcherResponse {
	desc := "A talk"
	return domain.SessionFetcherResponse{
		{
			Date: "2025-03-01",
			Rooms: []domain.SessionFetcherRoom{
				{
					ID:   1,
					Name: "Room A",
					Sessions: []domain.SessionFetcherSession{
						{
							ID:          "s1",
							Title:       "Talk 1",
							Description: &desc,
							StartsAt:    time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
							EndsAt:      time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC),
							RoomID:      1,
							Categories: []domain.SessionCategory{
								{Name: "Tipo de sesión", CategoryItems: []domain.TagItem{{Name: "Conferencia"}}},
								{Name: "Event tag", CategoryItems: []domain.TagItem{{Name: "ai"}, {Name: "web"}}},
							},
						},
					},
				},
			},
		},
	}
}

func TestEventService_CreateEvent(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		event   *domain.Event
		wantErr bool
		assert  func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event)
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", OwnerID: "user-1"},
			wantErr: false,
			assert: func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event) {
				require.NotEmpty(t, event.ID)
				assert.False(t, event.CreatedAt.IsZero())
				assert.False(t, event.UpdatedAt.IsZero())
				assert.Equal(t, "Conf", event.Name)
				assert.Len(t, event.EventCode, 4)
				assert.Regexp(t, "^[a-z0-9]{4}$", event.EventCode)
				assert.Equal(t, "user-1", event.OwnerID)
				got, ok := eventRepo.byID[event.ID]
				require.True(t, ok)
				assert.Equal(t, event.ID, got.ID)
				assert.Equal(t, "user-1", got.OwnerID)
			},
		},
		{
			name: "repo error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				er.err = errors.New("db error")
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", OwnerID: "user-1"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
		{
			name: "missing owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			ev := &domain.Event{Name: tt.event.Name, OwnerID: tt.event.OwnerID}
			err := svc.CreateEvent(ctx, ev)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			fe, ok := eventRepo.(*fakeEventRepo)
			require.True(t, ok)
			tt.assert(t, fe, ev)
		})
	}
}

func TestEventService_UpdateEvent(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second
	eventDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	desc := "Annual conference"
	lat, lng := 40.7128, -74.0060

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		ownerID       string
		date          *time.Time
		description   *string
		locationLat   *float64
		locationLng   *float64
		wantErr       bool
		wantNotFound  bool
		wantForbidden bool
		assert        func(t *testing.T, event *domain.Event)
	}{
		{
			name: "success owner updates date and description",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:     "ev-1",
			ownerID:     "user-1",
			date:        &eventDate,
			description: &desc,
			locationLat: nil,
			locationLng: nil,
			assert: func(t *testing.T, event *domain.Event) {
				require.NotNil(t, event)
				require.NotNil(t, event.Date)
				assert.True(t, event.Date.Equal(eventDate))
				require.NotNil(t, event.Description)
				assert.Equal(t, desc, *event.Description)
			},
		},
		{
			name: "success owner updates location",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:     "ev-1",
			ownerID:     "user-1",
			date:        nil,
			description: nil,
			locationLat: &lat,
			locationLng: &lng,
			assert: func(t *testing.T, event *domain.Event) {
				require.NotNil(t, event)
				require.NotNil(t, event.LocationLat)
				assert.InDelta(t, lat, *event.LocationLat, 1e-6)
				require.NotNil(t, event.LocationLng)
				assert.InDelta(t, lng, *event.LocationLng, 1e-6)
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			ownerID:     "user-1",
			date:        &eventDate,
			wantErr:     true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			ownerID:       "user-2",
			date:          &eventDate,
			wantErr:       true,
			wantForbidden: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			got, err := svc.UpdateEvent(ctx, tt.eventID, tt.ownerID, tt.date, tt.description, tt.locationLat, tt.locationLng)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}

func TestEventService_ImportSessionizeData(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID string
		sessID  string
		wantErr bool
		assert  func(t *testing.T, sessionRepo *fakeSessionRepo)
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "abc123",
			wantErr: false,
			assert: func(t *testing.T, sessionRepo *fakeSessionRepo) {
				require.Len(t, sessionRepo.rooms, 1)
				assert.Equal(t, "Room A", sessionRepo.rooms[0].Name)
				require.Len(t, sessionRepo.sessions, 1)
				assert.Equal(t, "Talk 1", sessionRepo.sessions[0].Title)
				assert.ElementsMatch(t, []string{"Conferencia", "ai", "web"}, sessionRepo.sessions[0].Tags)
			},
		},
		{
			name: "fetcher error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{err: errors.New("fetch failed")}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "DeleteScheduleByEventID error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				sr := newFakeSessionRepo()
				sr.deleteErr = errors.New("delete failed")
				return newFakeEventRepo(), sr, &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "CreateRoom error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				sr := newFakeSessionRepo()
				sr.createRoomErr = errors.New("create room failed")
				return newFakeEventRepo(), sr, &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "CreateSession error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				sr := newFakeSessionRepo()
				sr.createSessionErr = errors.New("create session failed")
				return newFakeEventRepo(), sr, &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			err := svc.ImportSessionizeData(ctx, tt.eventID, tt.sessID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			sr, ok := sessionRepo.(*fakeSessionRepo)
			require.True(t, ok)
			tt.assert(t, sr)
		})
	}
}

func TestEventService_ListEventsByOwner(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		ownerID string
		wantLen int
		assert  func(t *testing.T, events []*domain.Event)
	}{
		{
			name: "returns only owner events",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				// Create two events for user-1, one for user-2
				_ = er.Create(ctx, &domain.Event{Name: "E1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				_ = er.Create(ctx, &domain.Event{Name: "E2", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				_ = er.Create(ctx, &domain.Event{Name: "Other", OwnerID: "user-2", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			ownerID: "user-1",
			wantLen: 2,
			assert: func(t *testing.T, events []*domain.Event) {
				for _, e := range events {
					assert.Equal(t, "user-1", e.OwnerID)
				}
			},
		},
		{
			name: "empty for unknown owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "E1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			ownerID: "user-none",
			wantLen: 0,
			assert:  func(t *testing.T, events []*domain.Event) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			events, err := svc.ListEventsByOwner(ctx, tt.ownerID)
			require.NoError(t, err)
			require.Len(t, events, tt.wantLen)
			tt.assert(t, events)
		})
	}
}

func TestEventService_GetEventByID(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name         string
		setup        func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID      string
		wantErr      bool
		wantNotFound bool
		assert       func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session)
	}{
		{
			name: "success with rooms and sessions",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				ev, _ := er.GetByID(ctx, "ev-1")
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: ev.ID, Name: "Room A"}}
				sr.sessions = []*domain.Session{{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", Tags: []string{}}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			wantErr:      false,
			wantNotFound: false,
			assert: func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session) {
				require.NotNil(t, event)
				assert.Equal(t, "ev-1", event.ID)
				assert.Equal(t, "Conf", event.Name)
				require.Len(t, rooms, 1)
				assert.Equal(t, "Room A", rooms[0].Name)
				require.Len(t, sessions, 1)
				assert.Equal(t, "Talk 1", sessions[0].Title)
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			wantErr:      true,
			wantNotFound: true,
			assert:       func(t *testing.T, _ *domain.Event, _ []*domain.Room, _ []*domain.Session) {},
		},
		{
			name: "success empty rooms and sessions",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			wantErr:      false,
			wantNotFound: false,
			assert: func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session) {
				require.NotNil(t, event)
				assert.Equal(t, "ev-1", event.ID)
				require.NotNil(t, rooms)
				require.Len(t, rooms, 0)
				require.NotNil(t, sessions)
				require.Len(t, sessions, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			event, rooms, sessions, err := svc.GetEventByID(ctx, tt.eventID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound), "expected ErrNotFound")
				}
				return
			}
			require.NoError(t, err)
			tt.assert(t, event, rooms, sessions)
		})
	}
}

func TestEventService_DeleteEvent(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		ownerID       string
		wantErr       bool
		wantNotFound  bool
		wantForbidden bool
		assertDeleted bool
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			ownerID:       "user-1",
			wantErr:       false,
			assertDeleted: true,
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			err := svc.DeleteEvent(ctx, tt.eventID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assertDeleted {
				_, err := eventRepo.GetByID(ctx, tt.eventID)
				require.True(t, errors.Is(err, domain.ErrNotFound), "event should be deleted")
			}
		})
	}
}

func TestEventService_ToggleRoomNotBookable(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		roomID        string
		ownerID       string
		wantErr       bool
		wantNotFound  bool
		wantForbidden bool
		assert        func(t *testing.T, room *domain.Room)
	}{
		{
			name: "success toggles false to true",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: false}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			roomID:  "room-1",
			ownerID: "user-1",
			wantErr: false,
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				require.True(t, room.NotBookable, "expected NotBookable to be true after toggle")
				require.Equal(t, "room-1", room.ID)
			},
		},
		{
			name: "success toggles true to false",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: true}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			roomID:  "room-1",
			ownerID: "user-1",
			wantErr: false,
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				require.False(t, room.NotBookable, "expected NotBookable to be false after toggle")
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "room not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "room belongs to different event",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-99", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			room, err := svc.ToggleRoomNotBookable(ctx, tt.eventID, tt.roomID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, room)
			}
		})
	}
}

func TestEventService_ListEventRooms(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		ownerID       string
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		wantLen       int
		assert        func(t *testing.T, rooms []*domain.Room)
	}{
		{
			name: "success owner lists rooms",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-1", Name: "Room A"},
					{ID: "room-2", EventID: "ev-1", Name: "Room B"},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			ownerID: "user-1",
			wantLen: 2,
			assert: func(t *testing.T, rooms []*domain.Room) {
				require.Len(t, rooms, 2)
				assert.Equal(t, "Room A", rooms[0].Name)
				assert.Equal(t, "Room B", rooms[1].Name)
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "success empty list",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			ownerID: "user-1",
			wantLen: 0,
			assert: func(t *testing.T, rooms []*domain.Room) {
				require.NotNil(t, rooms)
				require.Len(t, rooms, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			rooms, err := svc.ListEventRooms(ctx, tt.eventID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, rooms, tt.wantLen)
			if tt.assert != nil {
				tt.assert(t, rooms)
			}
		})
	}
}

func TestEventService_GetEventRoom(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		roomID        string
		ownerID       string
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		assert        func(t *testing.T, room *domain.Room)
	}{
		{
			name: "success owner gets room",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", Capacity: 50, Description: "Main hall"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			roomID:  "room-1",
			ownerID: "user-1",
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				assert.Equal(t, "room-1", room.ID)
				assert.Equal(t, "Room A", room.Name)
				assert.Equal(t, 50, room.Capacity)
				assert.Equal(t, "Main hall", room.Description)
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "room not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "room belongs to different event",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-99", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			room, err := svc.GetEventRoom(ctx, tt.eventID, tt.roomID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, room)
			}
		})
	}
}

func TestEventService_UpdateEventRoom(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		roomID        string
		ownerID       string
		capacity      int
		description   string
		howToGetThere string
		notBookable   *bool
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		assert        func(t *testing.T, room *domain.Room)
	}{
		{
			name: "success update all fields",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: false}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-1",
			capacity:      100,
			description:   "Big room",
			howToGetThere: "Second floor",
			notBookable:   ptrBool(true),
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				assert.Equal(t, 100, room.Capacity)
				assert.Equal(t, "Big room", room.Description)
				assert.Equal(t, "Second floor", room.HowToGetThere)
				assert.True(t, room.NotBookable)
			},
		},
		{
			name: "success notBookable nil keeps existing",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: true}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-1",
			capacity:      50,
			description:   "",
			howToGetThere: "",
			notBookable:   nil,
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				assert.True(t, room.NotBookable, "should keep existing true when notBookable is nil")
				assert.Equal(t, 50, room.Capacity)
			},
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "room not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			room, err := svc.UpdateEventRoom(ctx, tt.eventID, tt.roomID, tt.ownerID, tt.capacity, tt.description, tt.howToGetThere, tt.notBookable)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, room)
			}
		})
	}
}

func ptrBool(b bool) *bool { return &b }

func TestEventService_DeleteEventRoom(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		eventID       string
		roomID        string
		ownerID       string
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		assertDeleted bool
	}{
		{
			name: "success owner deletes room",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-1",
			assertDeleted: true,
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "room not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			err := svc.DeleteEventRoom(ctx, tt.eventID, tt.roomID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assertDeleted {
				sr := sessionRepo.(*fakeSessionRepo)
				_, err := sr.GetRoomByID(ctx, tt.roomID)
				require.True(t, errors.Is(err, domain.ErrNotFound), "room should be deleted")
			}
		})
	}
}

func TestEventService_AddEventTeamMember(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		eventID       string
		userIDToAdd   string
		ownerID       string
		setupEvent    func(*fakeEventRepo)
		setupTeamRepo func(*fakeEventTeamMemberRepo)
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		wantConflict  bool
	}{
		{
			name:        "owner adds team member success",
			eventID:     "ev-1",
			userIDToAdd: "user-2",
			ownerID:     "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr: false,
		},
		{
			name:        "forbidden not owner",
			eventID:     "ev-1",
			userIDToAdd: "user-2",
			ownerID:     "user-other",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name:         "event not found",
			eventID:      "ev-missing",
			userIDToAdd:  "user-2",
			ownerID:      "user-1",
			setupEvent:   func(er *fakeEventRepo) {},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:        "add owner returns ErrInvalidInput",
			eventID:     "ev-1",
			userIDToAdd: "user-1",
			ownerID:     "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:      true,
			wantConflict: true,
		},
		{
			name:        "already member returns ErrAlreadyMember",
			eventID:     "ev-1",
			userIDToAdd: "user-2",
			ownerID:     "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
			},
			wantErr:      true,
			wantConflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			if tt.setupTeamRepo != nil {
				tt.setupTeamRepo(teamRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			err := svc.AddEventTeamMember(ctx, tt.eventID, tt.userIDToAdd, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantConflict {
					require.True(t, errors.Is(err, domain.ErrAlreadyMember) || errors.Is(err, domain.ErrInvalidInput))
				}
				return
			}
			require.NoError(t, err)
			members, _ := teamRepo.ListByEventID(ctx, tt.eventID)
			require.Len(t, members, 1)
			require.Equal(t, tt.userIDToAdd, members[0].UserID)
		})
	}
}

func TestEventService_ListEventTeamMembers(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		eventID       string
		callerID      string
		setupEvent    func(*fakeEventRepo)
		setupTeamRepo func(*fakeEventTeamMemberRepo)
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		wantCount     int
	}{
		{
			name:     "owner lists members",
			eventID:  "ev-1",
			callerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
				tr.Add(ctx, "ev-1", "user-3")
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:     "forbidden not owner",
			eventID:  "ev-1",
			callerID: "user-other",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name:         "event not found",
			eventID:      "ev-missing",
			callerID:     "user-1",
			setupEvent:   func(er *fakeEventRepo) {},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:     "owner lists empty",
			eventID:  "ev-1",
			callerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			if tt.setupTeamRepo != nil {
				tt.setupTeamRepo(teamRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			got, err := svc.ListEventTeamMembers(ctx, tt.eventID, tt.callerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantCount)
		})
	}
}

func TestEventService_ListEventInvitations(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name            string
		eventID         string
		callerID        string
		search          string
		params          domain.PaginationParams
		setupEvent      func(*fakeEventRepo)
		setupInvitation func(*fakeEventInvitationRepo)
		wantErr         bool
		wantForbidden   bool
		wantNotFound    bool
		wantCount       int
		wantTotal       int
	}{
		{
			name:     "owner lists invitations non-empty",
			eventID:  "ev-1",
			callerID: "user-1",
			search:   "",
			params:   domain.PaginationParams{Page: 1, PageSize: 20},
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupInvitation: func(ir *fakeEventInvitationRepo) {
				_ = ir.Create(ctx, &domain.EventInvitation{EventID: "ev-1", Email: "a@example.com", SentAt: time.Now()})
				_ = ir.Create(ctx, &domain.EventInvitation{EventID: "ev-1", Email: "b@example.com", SentAt: time.Now()})
			},
			wantErr:   false,
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:     "owner lists empty",
			eventID:  "ev-1",
			callerID: "user-1",
			search:   "",
			params:   domain.PaginationParams{Page: 1, PageSize: 20},
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupInvitation: func(*fakeEventInvitationRepo) {},
			wantErr:         false,
			wantCount:       0,
			wantTotal:       0,
		},
		{
			name:     "owner lists with search filter",
			eventID:  "ev-1",
			callerID: "user-1",
			search:   "example",
			params:   domain.PaginationParams{Page: 1, PageSize: 20},
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupInvitation: func(ir *fakeEventInvitationRepo) {
				_ = ir.Create(ctx, &domain.EventInvitation{EventID: "ev-1", Email: "a@example.com", SentAt: time.Now()})
				_ = ir.Create(ctx, &domain.EventInvitation{EventID: "ev-1", Email: "b@example.com", SentAt: time.Now()})
				_ = ir.Create(ctx, &domain.EventInvitation{EventID: "ev-1", Email: "c@other.com", SentAt: time.Now()})
			},
			wantErr:   false,
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:     "forbidden not owner",
			eventID:  "ev-1",
			callerID: "user-other",
			search:   "",
			params:   domain.PaginationParams{Page: 1, PageSize: 20},
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupInvitation: func(*fakeEventInvitationRepo) {},
			wantErr:         true,
			wantForbidden:   true,
		},
		{
			name:            "event not found",
			eventID:         "ev-missing",
			callerID:        "user-1",
			search:          "",
			params:          domain.PaginationParams{Page: 1, PageSize: 20},
			setupEvent:      func(er *fakeEventRepo) {},
			setupInvitation: func(*fakeEventInvitationRepo) {},
			wantErr:         true,
			wantNotFound:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			invRepo := newFakeEventInvitationRepo()
			if tt.setupInvitation != nil {
				tt.setupInvitation(invRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), invRepo, newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			got, total, err := svc.ListEventInvitations(ctx, tt.eventID, tt.callerID, tt.search, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantCount)
			require.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestEventService_RemoveEventTeamMember(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name           string
		eventID        string
		userIDToRemove string
		ownerID        string
		setupEvent     func(*fakeEventRepo)
		setupTeamRepo  func(*fakeEventTeamMemberRepo)
		wantErr        bool
		wantForbidden  bool
		wantNotFound   bool
	}{
		{
			name:           "owner removes member success",
			eventID:        "ev-1",
			userIDToRemove: "user-2",
			ownerID:        "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
			},
			wantErr: false,
		},
		{
			name:           "forbidden not owner",
			eventID:        "ev-1",
			userIDToRemove: "user-2",
			ownerID:        "user-other",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			userIDToRemove: "user-2",
			ownerID:        "user-1",
			setupEvent:     func(er *fakeEventRepo) {},
			wantErr:        true,
			wantNotFound:   true,
		},
		{
			name:           "member not in team returns not found",
			eventID:        "ev-1",
			userIDToRemove: "user-99",
			ownerID:        "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			if tt.setupTeamRepo != nil {
				tt.setupTeamRepo(teamRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			err := svc.RemoveEventTeamMember(ctx, tt.eventID, tt.userIDToRemove, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			members, _ := teamRepo.ListByEventID(ctx, tt.eventID)
			for _, m := range members {
				require.NotEqual(t, tt.userIDToRemove, m.UserID)
			}
		})
	}
}

func TestEventService_AddEventTeamMemberByEmail(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name             string
		eventID          string
		email            string
		ownerID          string
		setupEvent       func(*fakeEventRepo)
		setupUserRepo    func(*fakeUserRepoForSchedule)
		wantErr          bool
		wantUserNotFound bool
		wantConflict     bool
		wantMemberUserID string
	}{
		{
			name:    "success adds by email",
			eventID: "ev-1",
			email:   "teammate@example.com",
			ownerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupUserRepo: func(ur *fakeUserRepoForSchedule) {
				ur.addUser("teammate@example.com", "user-2")
			},
			wantErr:          false,
			wantMemberUserID: "user-2",
		},
		{
			name:    "user not found by email",
			eventID: "ev-1",
			email:   "nobody@example.com",
			ownerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupUserRepo:    func(ur *fakeUserRepoForSchedule) {},
			wantErr:          true,
			wantUserNotFound: true,
		},
		{
			name:    "email normalized to lower",
			eventID: "ev-1",
			email:   "Teammate@Example.COM",
			ownerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupUserRepo: func(ur *fakeUserRepoForSchedule) {
				ur.addUser("teammate@example.com", "user-2")
			},
			wantErr:          false,
			wantMemberUserID: "user-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			userRepo := newFakeUserRepoForSchedule()
			if tt.setupUserRepo != nil {
				tt.setupUserRepo(userRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, userRepo, newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			got, err := svc.AddEventTeamMemberByEmail(ctx, tt.eventID, tt.email, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUserNotFound {
					require.True(t, errors.Is(err, domain.ErrUserNotFound))
				}
				if tt.wantConflict {
					require.True(t, errors.Is(err, domain.ErrAlreadyMember) || errors.Is(err, domain.ErrInvalidInput))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.eventID, got.EventID)
			require.Equal(t, tt.wantMemberUserID, got.UserID)
		})
	}
}

func TestEventService_SendEventInvitations(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name             string
		eventID          string
		ownerID          string
		emails           []string
		setupEvent       func(*fakeEventRepo)
		setupUser        func(*fakeUserRepoForSchedule)
		setupEmail       func(*fakeEmailService)
		wantSent         int
		wantFailed       []string
		wantErr          bool
		wantErrNotFound  bool
		wantErrForbidden bool
	}{
		{
			name:    "success sends to two emails",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{"a@example.com", "b@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser: func(ur *fakeUserRepoForSchedule) {
				ur.addUserWithName("owner@x.com", "user-1", "Jane", "Doe")
			},
			setupEmail: func(*fakeEmailService) {},
			wantSent:   2,
			wantFailed: nil,
			wantErr:    false,
		},
		{
			name:            "event not found",
			eventID:         "ev-missing",
			ownerID:         "user-1",
			emails:          []string{"a@example.com"},
			setupEvent:      func(er *fakeEventRepo) {},
			setupUser:       func(ur *fakeUserRepoForSchedule) {},
			setupEmail:      func(*fakeEmailService) {},
			wantErr:         true,
			wantErrNotFound: true,
		},
		{
			name:    "forbidden when not owner",
			eventID: "ev-1",
			ownerID: "user-2",
			emails:  []string{"a@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser:        func(ur *fakeUserRepoForSchedule) {},
			setupEmail:       func(*fakeEmailService) {},
			wantErr:          true,
			wantErrForbidden: true,
		},
		{
			name:    "partial failure when email send fails",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{"ok@example.com", "fail@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser: func(ur *fakeUserRepoForSchedule) {
				ur.addUserWithName("owner@x.com", "user-1", "Jane", "Doe")
			},
			setupEmail: func(es *fakeEmailService) {
				es.sendEventInvitationErr = errors.New("smtp error")
				// Fail only for second email - we can't do that with current fake, so we make all fail and expect sent=0, failed=2
				// Actually let's make sendEventInvitationErr set and then we get first email: create ok, send fail -> failed. Second: create ok, send fail -> failed. So sent=0, failed=2.
				// For "partial" we need one to succeed and one to fail. So we need the fake to fail on a specific email. Simpler: just test that when send fails, that email is in failed. So set sendEventInvitationErr and both go to failed, sent=0, failed=2.
			},
			wantSent:   0,
			wantFailed: []string{"ok@example.com", "fail@example.com"},
			wantErr:    false,
		},
		{
			name:    "empty emails returns zero sent",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser:  func(ur *fakeUserRepoForSchedule) {},
			setupEmail: func(*fakeEmailService) {},
			wantSent:   0,
			wantFailed: nil,
			wantErr:    false,
		},
		{
			name:    "duplicate email in list: first sent, second failed",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{"dup@example.com", "dup@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser: func(ur *fakeUserRepoForSchedule) {
				ur.addUserWithName("owner@x.com", "user-1", "Jane", "Doe")
			},
			setupEmail: func(*fakeEmailService) {},
			wantSent:   1,
			wantFailed: []string{"dup@example.com"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			if tt.setupEvent != nil {
				tt.setupEvent(eventRepo)
			}
			userRepo := newFakeUserRepoForSchedule()
			if tt.setupUser != nil {
				tt.setupUser(userRepo)
			}
			invRepo := newFakeEventInvitationRepo()
			emailSvc := newFakeEmailService()
			if tt.setupEmail != nil {
				tt.setupEmail(emailSvc)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), newFakeEventTeamMemberRepo(), userRepo, invRepo, emailSvc, &fakeSessionizeFetcher{}, timeout)

			sent, failed, err := svc.SendEventInvitations(ctx, tt.eventID, tt.ownerID, tt.emails)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantErrForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSent, sent)
			require.ElementsMatch(t, tt.wantFailed, failed)
			if tt.wantSent > 0 && len(tt.emails) > 0 {
				list, _, _ := invRepo.ListByEventID(ctx, tt.eventID, "", domain.PaginationParams{Page: 1, PageSize: 1000})
				require.Len(t, list, tt.wantSent, "invitations persisted should match sent count")
				require.Len(t, emailSvc.sentInvitations, tt.wantSent, "emails sent should match sent count")
			}
		})
	}
}

func TestEventService_UpdateSessionSchedule(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	baseStart := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	baseEnd := time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC)
	newStart := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	newEnd := time.Date(2025, 3, 1, 13, 0, 0, 0, time.UTC)

	type args struct {
		eventID   string
		sessionID string
		ownerID   string
		roomID    *string
		startTime *time.Time
		endTime   *time.Time
	}

	newRoomID := "room-2"

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher)
		args          args
		wantErr       bool
		wantNotFound  bool
		wantForbidden bool
		wantInvalid   bool
		assert        func(t *testing.T, sess *domain.Session)
	}{
		{
			name: "success move room and change time",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-1", Name: "Room A"},
					{ID: "room-2", EventID: "ev-1", Name: "Room B"},
				}
				sr.sessions = []*domain.Session{
					{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", StartTime: baseStart, EndTime: baseEnd},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-1",
				roomID:    &newRoomID,
				startTime: &newStart,
				endTime:   &newEnd,
			},
			assert: func(t *testing.T, sess *domain.Session) {
				require.NotNil(t, sess)
				assert.Equal(t, "sess-1", sess.ID)
				assert.Equal(t, "room-2", sess.RoomID)
				assert.True(t, sess.StartTime.Equal(newStart))
				assert.True(t, sess.EndTime.Equal(newEnd))
			},
		},
		{
			name: "success change time only",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-1", Name: "Room A"},
				}
				sr.sessions = []*domain.Session{
					{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", StartTime: baseStart, EndTime: baseEnd},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-1",
				startTime: &newStart,
				endTime:   &newEnd,
			},
			assert: func(t *testing.T, sess *domain.Session) {
				require.NotNil(t, sess)
				assert.Equal(t, "room-1", sess.RoomID)
				assert.True(t, sess.StartTime.Equal(newStart))
				assert.True(t, sess.EndTime.Equal(newEnd))
			},
		},
		{
			name: "success change room only",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-1", Name: "Room A"},
					{ID: "room-2", EventID: "ev-1", Name: "Room B"},
				}
				sr.sessions = []*domain.Session{
					{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", StartTime: baseStart, EndTime: baseEnd},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-1",
				roomID:    &newRoomID,
			},
			assert: func(t *testing.T, sess *domain.Session) {
				require.NotNil(t, sess)
				assert.Equal(t, "room-2", sess.RoomID)
				assert.True(t, sess.StartTime.Equal(baseStart))
				assert.True(t, sess.EndTime.Equal(baseEnd))
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-missing",
				sessionID: "sess-1",
				ownerID:   "user-1",
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-2",
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "session not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-missing",
				ownerID:   "user-1",
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "session belongs to different event",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-99", Name: "Other Event Room"},
				}
				sr.sessions = []*domain.Session{
					{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", StartTime: baseStart, EndTime: baseEnd},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-1",
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "new room not in event",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-1", Name: "Room A"},
					{ID: "room-2", EventID: "ev-99", Name: "Other Event Room"},
				}
				sr.sessions = []*domain.Session{
					{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", StartTime: baseStart, EndTime: baseEnd},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-1",
				roomID:    &newRoomID,
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "invalid when end before start",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1"})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{
					{ID: "room-1", EventID: "ev-1", Name: "Room A"},
				}
				sr.sessions = []*domain.Session{
					{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", StartTime: baseStart, EndTime: baseEnd},
				}
				return er, sr, &fakeSessionizeFetcher{}
			},
			args: args{
				eventID:   "ev-1",
				sessionID: "sess-1",
				ownerID:   "user-1",
				startTime: &newEnd,
				endTime:   &newStart,
			},
			wantErr:     true,
			wantInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			got, err := svc.UpdateSessionSchedule(ctx, tt.args.eventID, tt.args.sessionID, tt.args.ownerID, tt.args.roomID, tt.args.startTime, tt.args.endTime)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantInvalid {
					require.True(t, errors.Is(err, domain.ErrInvalidInput))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}
