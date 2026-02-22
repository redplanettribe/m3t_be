package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTokenVerifier implements domain.TokenVerifier for tests.
type fakeTokenVerifier struct {
	userID string
	err    error
}

func (f *fakeTokenVerifier) Verify(_ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.userID, nil
}

func TestRequireAuth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		authHeader     string
		verifier       domain.TokenVerifier
		wantStatus     int
		wantBodyCode   string
		nextCalled     bool
		wantContextID  string
	}{
		{
			name:          "valid token sets context and calls next",
			authHeader:    "Bearer valid-token",
			verifier:      &fakeTokenVerifier{userID: "user-123"},
			wantStatus:    http.StatusOK,
			nextCalled:    true,
			wantContextID: "user-123",
		},
		{
			name:         "missing authorization header",
			authHeader:   "",
			verifier:     &fakeTokenVerifier{userID: "user-123"},
			wantStatus:   http.StatusUnauthorized,
			wantBodyCode: helpers.ErrCodeUnauthorized,
			nextCalled:   false,
		},
		{
			name:         "invalid authorization format no Bearer prefix",
			authHeader:   "Basic abc",
			verifier:     &fakeTokenVerifier{userID: "user-123"},
			wantStatus:   http.StatusUnauthorized,
			wantBodyCode: helpers.ErrCodeUnauthorized,
			nextCalled:   false,
		},
		{
			name:         "empty token after Bearer",
			authHeader:   "Bearer ",
			verifier:     &fakeTokenVerifier{userID: "user-123"},
			wantStatus:   http.StatusUnauthorized,
			wantBodyCode: helpers.ErrCodeUnauthorized,
			nextCalled:   false,
		},
		{
			name:         "verifier returns error",
			authHeader:   "Bearer bad-token",
			verifier:     &fakeTokenVerifier{err: errors.New("invalid or expired token")},
			wantStatus:   http.StatusUnauthorized,
			wantBodyCode: helpers.ErrCodeUnauthorized,
			nextCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			var capturedUserID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				id, ok := UserIDFromContext(r.Context())
				if ok {
					capturedUserID = id
				}
				w.WriteHeader(http.StatusOK)
			})
			wrap := RequireAuth(tt.verifier, logger)
			handler := wrap(next)

			req := httptest.NewRequest(http.MethodGet, "http://test/users/me", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "status code")
			assert.Equal(t, tt.nextCalled, nextCalled, "next handler called")
			if tt.nextCalled && tt.wantContextID != "" {
				assert.Equal(t, tt.wantContextID, capturedUserID, "user ID in context")
			}
			if tt.wantStatus != http.StatusOK && tt.wantBodyCode != "" {
				var envelope helpers.APIResponse
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&envelope))
				require.NotNil(t, envelope.Error)
				assert.Equal(t, tt.wantBodyCode, envelope.Error.Code)
			}
		})
	}
}
