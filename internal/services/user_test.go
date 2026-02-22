package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUserRepo implements domain.UserRepository for tests.
type fakeUserRepo struct {
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
	getErr  error
	updateErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[string]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *domain.User) error { return nil }

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if u, ok := f.byID[id]; ok {
		// Return a copy so tests can mutate without affecting stored
		cp := *u
		return &cp, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakeUserRepo) Update(ctx context.Context, u *domain.User) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.byID[u.ID]; !ok {
		return domain.ErrUserNotFound
	}
	if u.Email != "" {
		if existing, ok := f.byEmail[u.Email]; ok && existing.ID != u.ID {
			return domain.ErrDuplicateEmail
		}
	}
	f.byID[u.ID] = u
	if u.Email != "" {
		f.byEmail[u.Email] = u
	}
	return nil
}

func (f *fakeUserRepo) AssignRole(ctx context.Context, userID, roleID string) error { return nil }

func TestUserService_GetByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		id        string
		setup     func(*fakeUserRepo)
		wantUser  *domain.User
		wantErr   error
	}{
		{
			name: "success",
			id:   "user-1",
			setup: func(f *fakeUserRepo) {
				u := &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}
				f.byID["user-1"] = u
				f.byEmail["a@b.com"] = u
			},
			wantUser: &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice"},
			wantErr:  nil,
		},
		{
			name:  "not found",
			id:    "missing",
			setup: func(f *fakeUserRepo) {},
			wantUser: nil,
			wantErr: domain.ErrUserNotFound,
		},
		{
			name:  "repo error",
			id:    "user-1",
			setup: func(f *fakeUserRepo) { f.getErr = sql.ErrConnDone },
			wantUser: nil,
			wantErr: nil, // service wraps; we assert error is not ErrUserNotFound
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeUserRepo()
			tt.setup(fake)
			svc := NewUserService(fake)

			user, err := svc.GetByID(ctx, tt.id)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, err == tt.wantErr || (tt.wantErr == domain.ErrUserNotFound && err == domain.ErrUserNotFound))
				assert.Nil(t, user)
				return
			}
			if tt.wantUser != nil {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.wantUser.ID, user.ID)
				assert.Equal(t, tt.wantUser.Email, user.Email)
				assert.Equal(t, tt.wantUser.Name, user.Name)
				return
			}
			// repo error case
			require.Error(t, err)
			assert.False(t, err == domain.ErrUserNotFound)
		})
	}
}

func TestUserService_Update(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name      string
		user      *domain.User
		setup     func(*fakeUserRepo)
		wantErr   error
	}{
		{
			name: "success",
			user: &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice Updated", UpdatedAt: now},
			setup: func(f *fakeUserRepo) {
				u := &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now}
				f.byID["user-1"] = u
				f.byEmail["a@b.com"] = u
			},
			wantErr: nil,
		},
		{
			name: "duplicate email",
			user: &domain.User{ID: "user-1", Email: "other@b.com", Name: "Alice", UpdatedAt: now},
			setup: func(f *fakeUserRepo) {
				u := &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now}
				f.byID["user-1"] = u
				f.byEmail["a@b.com"] = u
				f.byEmail["other@b.com"] = &domain.User{ID: "user-2", Email: "other@b.com"}
			},
			wantErr: domain.ErrDuplicateEmail,
		},
		{
			name: "invalid email format",
			user: &domain.User{ID: "user-1", Email: "not-an-email", Name: "Alice", UpdatedAt: now},
			setup: func(f *fakeUserRepo) {
				u := &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now}
				f.byID["user-1"] = u
				f.byEmail["a@b.com"] = u
			},
			wantErr: nil, // we assert error message contains invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeUserRepo()
			tt.setup(fake)
			svc := NewUserService(fake)

			err := svc.Update(ctx, tt.user)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, err == tt.wantErr || (tt.wantErr == domain.ErrDuplicateEmail && err == domain.ErrDuplicateEmail))
				return
			}
			if tt.name == "invalid email format" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid email")
				return
			}
			require.NoError(t, err)
		})
	}
}
