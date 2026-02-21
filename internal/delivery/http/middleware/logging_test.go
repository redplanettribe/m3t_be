package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// capturingHandler records the last log record for assertions.
type capturingHandler struct {
	record slog.Record
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.record = r.Clone()
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }

func (h *capturingHandler) WithGroup(_ string) slog.Handler { return h }

func TestLoggingMiddleware(t *testing.T) {
	var cap capturingHandler
	logger := slog.New(&cap)

	tests := []struct {
		name            string
		handlerStatus   int
		path            string
		method          string
	}{
		{"ok status", http.StatusOK, "/events", http.MethodPost},
		{"created", http.StatusCreated, "/auth/signup", http.MethodPost},
		{"server error", http.StatusInternalServerError, "/events", http.MethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
			})
			handler := LoggingMiddleware(logger, next)
			req := httptest.NewRequest(tt.method, "http://test"+tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			require.Equal(t, "request", cap.record.Message)
			attrs := make(map[string]slog.Value)
			cap.record.Attrs(func(a slog.Attr) bool {
				attrs[a.Key] = a.Value
				return true
			})
			require.Contains(t, attrs, "method")
			require.Contains(t, attrs, "path")
			require.Contains(t, attrs, "status")
			require.Contains(t, attrs, "duration_ms")
			require.Equal(t, tt.method, attrs["method"].String())
			require.Equal(t, tt.path, attrs["path"].String())
			require.Equal(t, int64(tt.handlerStatus), attrs["status"].Int64())
			require.GreaterOrEqual(t, attrs["duration_ms"].Int64(), int64(0))
			require.Equal(t, tt.handlerStatus, rr.Code)
		})
	}
}
