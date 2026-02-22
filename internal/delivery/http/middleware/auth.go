package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	h "multitrackticketing/internal/delivery/http/helpers"
	"multitrackticketing/internal/domain"
)

type contextKey string

const userIDKey contextKey = "userID"

// SetUserID returns a context with the user ID set. Used by auth middleware.
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext returns the authenticated user ID from the context, if present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// RequireAuth returns a wrapper that validates the Bearer token and sets the user ID in the request context.
// If the token is missing or invalid, it responds with 401 and does not call next.
func RequireAuth(verifier domain.TokenVerifier, logger *slog.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "missing authorization header")
				return
			}
			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "invalid authorization format")
				return
			}
			token := strings.TrimSpace(auth[len(prefix):])
			if token == "" {
				h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "missing token")
				return
			}
			userID, err := verifier.Verify(token)
			if err != nil {
				h.WriteJSONError(w, http.StatusUnauthorized, h.ErrCodeUnauthorized, "invalid or expired token")
				return
			}
			r = r.WithContext(SetUserID(r.Context(), userID))
			next(w, r)
		}
	}
}
