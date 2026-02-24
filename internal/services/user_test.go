package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRoleRepo implements domain.RoleRepository for tests.
type fakeRoleRepo struct {
	byCode    map[string]*domain.Role
	listByUID map[string][]*domain.Role
	getErr    error
}

func newFakeRoleRepo() *fakeRoleRepo {
	return &fakeRoleRepo{
		byCode:    make(map[string]*domain.Role),
		listByUID: make(map[string][]*domain.Role),
	}
}

func (f *fakeRoleRepo) GetByCode(ctx context.Context, code string) (*domain.Role, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if r, ok := f.byCode[code]; ok {
		return r, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakeRoleRepo) ListByUserID(ctx context.Context, userID string) ([]*domain.Role, error) {
	if roles, ok := f.listByUID[userID]; ok {
		return roles, nil
	}
	return nil, nil
}

// fakePasswordHasher implements domain.PasswordHasher for tests.
type fakePasswordHasher struct {
	salt string
	hash string
}

func (f *fakePasswordHasher) GenerateSalt() (string, error) { return f.salt, nil }
func (f *fakePasswordHasher) Hash(salt, password string) (string, error) {
	if f.hash != "" {
		return f.hash, nil
	}
	return "hash-" + password, nil
}
func (f *fakePasswordHasher) Compare(hash, salt, password string) error {
	if f.hash != "" && hash != f.hash {
		return errors.New("mismatch")
	}
	return nil
}

// fakeTokenIssuer implements domain.TokenIssuer for tests.
type fakeTokenIssuer struct {
	token string
	err   error
}

func (f *fakeTokenIssuer) Issue(userID, email string, roles []string, expiry time.Duration) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.token != "" {
		return f.token, nil
	}
	return "token-" + userID, nil
}

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

func (f *fakeUserRepo) Create(ctx context.Context, u *domain.User) error {
	u.ID = "created-1"
	f.byID[u.ID] = u
	if u.Email != "" {
		f.byEmail[u.Email] = u
	}
	return nil
}

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

	roleRepo := newFakeRoleRepo()
	roleRepo.byCode["attendee"] = domain.NewRole("role-1", "attendee")
	hasher := &fakePasswordHasher{}
	issuer := &fakeTokenIssuer{}
	tokenExpiry := 1 * time.Hour

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeUserRepo()
			tt.setup(fake)
			svc := NewUserService(fake, roleRepo, hasher, issuer, tokenExpiry, nil)

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

	roleRepo := newFakeRoleRepo()
	roleRepo.byCode["attendee"] = domain.NewRole("role-1", "attendee")
	hasher := &fakePasswordHasher{}
	issuer := &fakeTokenIssuer{}
	tokenExpiry := 1 * time.Hour

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeUserRepo()
			tt.setup(fake)
			svc := NewUserService(fake, roleRepo, hasher, issuer, tokenExpiry, nil)

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

func TestUserService_SignUp(t *testing.T) {
	ctx := context.Background()
	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	roleRepo.byCode["attendee"] = domain.NewRole("role-1", "attendee")
	roleRepo.byCode["admin"] = domain.NewRole("role-2", "admin")
	hasher := &fakePasswordHasher{salt: "s", hash: "h"}
	issuer := &fakeTokenIssuer{}
	svc := NewUserService(userRepo, roleRepo, hasher, issuer, time.Hour, nil)

	user, err := svc.SignUp(ctx, "alice@example.com", "password8", "Alice", "", "attendee")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "created-1", user.ID)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "h", user.PasswordHash)
	assert.Equal(t, "s", user.Salt)
	// default role when invalid
	user2, err := svc.SignUp(ctx, "bob@example.com", "password9", "Bob", "", "invalid")
	require.NoError(t, err)
	require.NotNil(t, user2)
	assert.Equal(t, "bob@example.com", user2.Email)
}

func TestUserService_Login(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	userRepo := newFakeUserRepo()
	u := &domain.User{ID: "u1", Email: "login@example.com", PasswordHash: "h", Salt: "s", Name: "Login User", CreatedAt: now, UpdatedAt: now}
	userRepo.byID["u1"] = u
	userRepo.byEmail["login@example.com"] = u
	roleRepo := newFakeRoleRepo()
	roleRepo.listByUID["u1"] = []*domain.Role{domain.NewRole("r1", "attendee")}
	hasher := &fakePasswordHasher{salt: "s", hash: "h"}
	issuer := &fakeTokenIssuer{token: "jwt-token-123"}
	svc := NewUserService(userRepo, roleRepo, hasher, issuer, time.Hour, nil)

	token, user, err := svc.Login(ctx, "login@example.com", "anypassword")
	require.NoError(t, err)
	assert.Equal(t, "jwt-token-123", token)
	require.NotNil(t, user)
	assert.Equal(t, "u1", user.ID)
	assert.Equal(t, "login@example.com", user.Email)
	assert.Equal(t, "Login User", user.Name)

	_, _, err = svc.Login(ctx, "wrong@example.com", "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}
