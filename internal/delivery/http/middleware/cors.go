package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowMethods = "GET, POST, PATCH, PUT, DELETE, OPTIONS"
	corsAllowHeaders = "Authorization, Content-Type, Accept"
	corsMaxAge       = "86400"
)

// CORS returns a handler that adds CORS headers for allowed origins and
// responds to OPTIONS preflight requests with 204.
func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		o = strings.TrimSuffix(o, "/")
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		_, ok := allowed[origin]

		if r.Method == http.MethodOptions {
			if ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
				w.Header().Set("Access-Control-Max-Age", corsMaxAge)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if ok {
			wrapped := &corsResponseWriter{ResponseWriter: w, origin: origin}
			next.ServeHTTP(wrapped, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// corsResponseWriter adds CORS headers to the response for an allowed origin.
type corsResponseWriter struct {
	http.ResponseWriter
	origin string
}

func (w *corsResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Set("Access-Control-Allow-Origin", w.origin)
	w.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
	w.ResponseWriter.WriteHeader(code)
}
