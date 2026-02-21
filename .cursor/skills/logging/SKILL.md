---
name: logging
description: Apply or extend logging. Use when adding/changing logging, adding HTTP handlers that return 5xx, adding middleware, or configuring the app logger in cmd.
---

# Logging

Follow [.cursor/rules/logging.mdc](.cursor/rules/logging.mdc) for all logging conventions.

## When to use

- Adding or changing logging anywhere in the codebase
- Adding HTTP handlers that may return 5xx
- Adding new middleware (e.g. request logging)
- Configuring the application logger in `cmd`

## Workflow

- **New handler that returns 5xx**: In addition to `WriteJSONError(w, http.StatusInternalServerError, ...)`, log the error with the controller’s logger (e.g. `c.Logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "err", err)`). Controllers receive the logger from `cmd`; ensure the constructor takes `logger *slog.Logger` as the first argument.
- **New request-logging middleware**: Use the same pattern as `LoggingMiddleware` in `internal/delivery/http/middleware/logging.go`: wrap the response writer to capture status, record start time, call next, then log method, path, status, duration_ms. Do not log request or response bodies. Add new middlewares as separate files under `internal/delivery/http/middleware/` (e.g. `middleware/recovery.go`).
- **Repository**: Do not add logging inside the repository by default. If logging is required later, introduce a Logger interface in `internal/domain` and implement it in `cmd`; inject via the repo constructor.
