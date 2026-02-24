package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"multitrackticketing/internal/domain"
	"multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/delivery/http/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUserService implements domain.UserService for handler tests.
type fakeUserService struct {
	getByIDUser        *domain.User
	getByIDErr         error
	updateErr          error
	lastUpdate         *domain.User
	requestLoginCodeErr error
	verifyToken        string
	verifyUser         *domain.User
	verifyErr          error
}

func (f *fakeUserService) RequestLoginCode(ctx context.Context, email string) error {
	return f.requestLoginCodeErr
}

func (f *fakeUserService) VerifyLoginCode(ctx context.Context, email, code string) (string, *domain.User, error) {
	if f.verifyErr != nil {
		return "", nil, f.verifyErr
	}
	return f.verifyToken, f.verifyUser, nil
}

func (f *fakeUserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.getByIDUser, nil
}

func (f *fakeUserService) Update(ctx context.Context, user *domain.User) error {
	f.lastUpdate = user
	return f.updateErr
}

func TestUserController_GetMe(t *testing.T) {
	userLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		contextUserID  string
		fakeUser       *domain.User
		fakeErr        error
		wantStatus     int
		wantBodyCode   string
		checkUser      func(t *testing.T, u *domain.User)
	}{
		{
			name:          "success",
			contextUserID: "user-123",
			fakeUser:      &domain.User{ID: "user-123", Email: "a@b.com", Name: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			wantStatus:    http.StatusOK,
			checkUser: func(t *testing.T, u *domain.User) {
				assert.Equal(t, "user-123", u.ID)
				assert.Equal(t, "a@b.com", u.Email)
				assert.Equal(t, "Alice", u.Name)
			},
		},
		{
			name:          "no user in context",
			contextUserID: "",
			wantStatus:    http.StatusUnauthorized,
			wantBodyCode:  helpers.ErrCodeUnauthorized,
		},
		{
			name:          "user not found",
			contextUserID: "user-123",
			fakeErr:       domain.ErrUserNotFound,
			wantStatus:    http.StatusNotFound,
			wantBodyCode:  helpers.ErrCodeNotFound,
		},
		{
			name:          "service error",
			contextUserID: "user-123",
			fakeErr:       assert.AnError,
			wantStatus:    http.StatusInternalServerError,
			wantBodyCode:  helpers.ErrCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeUserService{getByIDUser: tt.fakeUser, getByIDErr: tt.fakeErr}
			ctrl := NewUserController(userLogger, fake)

			req := httptest.NewRequest(http.MethodGet, "http://test/users/me", nil)
			if tt.contextUserID != "" {
				req = req.WithContext(middleware.SetUserID(req.Context(), tt.contextUserID))
			}
			rr := httptest.NewRecorder()

			ctrl.GetMe(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK && tt.checkUser != nil {
				require.Nil(t, envelope.Error)
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var u domain.User
				require.NoError(t, json.Unmarshal(dataBytes, &u))
				tt.checkUser(t, &u)
			}
			if tt.wantBodyCode != "" && tt.wantStatus != http.StatusOK {
				require.NotNil(t, envelope.Error)
				assert.Equal(t, tt.wantBodyCode, envelope.Error.Code)
			}
		})
	}
}

func TestUserController_UpdateMe(t *testing.T) {
	userLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	now := time.Now()

	tests := []struct {
		name           string
		contextUserID  string
		body           string
		fakeUser       *domain.User
		fakeUpdateErr  error
		wantStatus     int
		wantBodyCode   string
		wantBodySubstr string
	}{
		{
			name:          "success update name",
			contextUserID: "user-123",
			body:          `{"name":"Alice Updated"}`,
			fakeUser:      &domain.User{ID: "user-123", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now},
			wantStatus:    http.StatusOK,
		},
		{
			name:          "success update email",
			contextUserID: "user-123",
			body:          `{"email":"new@example.com"}`,
			fakeUser:      &domain.User{ID: "user-123", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now},
			wantStatus:    http.StatusOK,
		},
		{
			name:          "no user in context",
			contextUserID: "",
			body:          `{"name":"x"}`,
			wantStatus:    http.StatusUnauthorized,
			wantBodyCode:  helpers.ErrCodeUnauthorized,
		},
		{
			name:           "invalid json",
			contextUserID:  "user-123",
			body:           `{invalid`,
			fakeUser:       &domain.User{ID: "user-123", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now},
			wantStatus:     http.StatusBadRequest,
			wantBodyCode:   helpers.ErrCodeBadRequest,
			wantBodySubstr: "invalid",
		},
		{
			name:           "invalid email format",
			contextUserID:  "user-123",
			body:           `{"email":"not-an-email"}`,
			fakeUser:       &domain.User{ID: "user-123", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now},
			wantStatus:     http.StatusBadRequest,
			wantBodyCode:   helpers.ErrCodeBadRequest,
			wantBodySubstr: "email",
		},
		{
			name:          "duplicate email",
			contextUserID: "user-123",
			body:          `{"email":"taken@example.com"}`,
			fakeUser:      &domain.User{ID: "user-123", Email: "a@b.com", Name: "Alice", CreatedAt: now, UpdatedAt: now},
			fakeUpdateErr: domain.ErrDuplicateEmail,
			wantStatus:    http.StatusConflict,
			wantBodyCode:  helpers.ErrCodeConflict,
		},
		{
			name:          "user not found on get",
			contextUserID: "user-123",
			body:          `{"name":"x"}`,
			fakeUser:      nil,
			fakeUpdateErr: nil,
			wantStatus:    http.StatusNotFound,
			wantBodyCode:  helpers.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getErr := error(nil)
			if tt.fakeUser == nil && tt.name == "user not found on get" {
				getErr = domain.ErrUserNotFound
			}
			fake := &fakeUserService{
				getByIDUser: tt.fakeUser,
				getByIDErr:  getErr,
				updateErr:  tt.fakeUpdateErr,
			}
			ctrl := NewUserController(userLogger, fake)

			req := httptest.NewRequest(http.MethodPatch, "http://test/users/me", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.contextUserID != "" {
				req = req.WithContext(middleware.SetUserID(req.Context(), tt.contextUserID))
			}
			rr := httptest.NewRecorder()

			ctrl.UpdateMe(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				return
			}
			require.NotNil(t, envelope.Error)
			if tt.wantBodyCode != "" {
				assert.Equal(t, tt.wantBodyCode, envelope.Error.Code)
			}
			if tt.wantBodySubstr != "" {
				assert.Contains(t, envelope.Error.Message, tt.wantBodySubstr)
			}
		})
	}
}

func TestUserController_RequestLoginCode(t *testing.T) {
	userLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name         string
		body         string
		fakeErr      error
		wantStatus   int
		wantBodyCode string
	}{
		{
			name:       "success",
			body:       `{"email":"alice@example.com"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:         "invalid json",
			body:         `{invalid`,
			wantStatus:   http.StatusBadRequest,
			wantBodyCode: helpers.ErrCodeBadRequest,
		},
		{
			name:         "missing email",
			body:         `{}`,
			wantStatus:   http.StatusBadRequest,
			wantBodyCode: helpers.ErrCodeBadRequest,
		},
		{
			name:         "invalid email from service",
			body:         `{"email":"bad"}`,
			fakeErr:      errors.New("invalid email format"),
			wantStatus:   http.StatusBadRequest,
			wantBodyCode: helpers.ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeUserService{requestLoginCodeErr: tt.fakeErr}
			ctrl := NewUserController(userLogger, fake)
			req := httptest.NewRequest(http.MethodPost, "http://test/auth/login/request", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			ctrl.RequestLoginCode(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus != http.StatusOK && tt.wantBodyCode != "" {
				require.NotNil(t, envelope.Error)
				assert.Equal(t, tt.wantBodyCode, envelope.Error.Code)
			}
		})
	}
}

func TestUserController_VerifyLoginCode(t *testing.T) {
	userLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	now := time.Now()

	tests := []struct {
		name          string
		body          string
		fakeToken     string
		fakeUser      *domain.User
		fakeErr       error
		wantStatus    int
		wantBodyCode  string
		checkToken    string
		checkUser     func(t *testing.T, u *domain.User)
	}{
		{
			name:       "success",
			body:       `{"email":"alice@example.com","code":"123456"}`,
			fakeToken:  "bearer-token-xyz",
			fakeUser:   &domain.User{ID: "id-1", Email: "alice@example.com", Name: "Alice", CreatedAt: now, UpdatedAt: now},
			wantStatus: http.StatusOK,
			checkToken: "bearer-token-xyz",
			checkUser: func(t *testing.T, u *domain.User) {
				assert.Equal(t, "id-1", u.ID)
				assert.Equal(t, "alice@example.com", u.Email)
				assert.Equal(t, "Alice", u.Name)
			},
		},
		{
			name:         "invalid or expired code",
			body:         `{"email":"alice@example.com","code":"000000"}`,
			fakeErr:      errors.New("invalid or expired code"),
			wantStatus:   http.StatusUnauthorized,
			wantBodyCode: helpers.ErrCodeUnauthorized,
		},
		{
			name:         "missing code",
			body:         `{"email":"alice@example.com"}`,
			wantStatus:   http.StatusBadRequest,
			wantBodyCode: helpers.ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeUserService{verifyToken: tt.fakeToken, verifyUser: tt.fakeUser, verifyErr: tt.fakeErr}
			ctrl := NewUserController(userLogger, fake)
			req := httptest.NewRequest(http.MethodPost, "http://test/auth/login/verify", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			ctrl.VerifyLoginCode(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			var envelope helpers.APIResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
			if tt.wantStatus == http.StatusOK {
				require.Nil(t, envelope.Error)
				dataBytes, err := json.Marshal(envelope.Data)
				require.NoError(t, err)
				var resp struct {
					Token     string       `json:"token"`
					TokenType string       `json:"token_type"`
					User      *domain.User `json:"user"`
				}
				require.NoError(t, json.Unmarshal(dataBytes, &resp))
				assert.Equal(t, tt.checkToken, resp.Token)
				assert.Equal(t, "Bearer", resp.TokenType)
				if tt.checkUser != nil && resp.User != nil {
					tt.checkUser(t, resp.User)
				}
				return
			}
			if tt.wantBodyCode != "" {
				require.NotNil(t, envelope.Error)
				assert.Equal(t, tt.wantBodyCode, envelope.Error.Code)
			}
		})
	}
}
