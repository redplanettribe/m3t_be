---
name: write-tests
description: Add or extend unit tests for HTTP handlers, use cases, and repositories in the Go backend. Use when adding tests, writing handler tests, use case tests, or repository tests, or when asked to test new or existing code.
---

# Write Tests

Follow [.cursor/rules/testing.mdc](.cursor/rules/testing.mdc) and [.cursor/rules/go-conventions.mdc](.cursor/rules/go-conventions.mdc).

## 1. Handlers

- Build the controller with a fake or mock implementing the service interface (e.g. `domain.EventService`).
- For each case: create request with `httptest.NewRequest`, create `httptest.ResponseRecorder`, call the handler, then use `assert`/`require` on status code and response body (e.g. decode JSON and compare, or check substring).
- Use table-driven tests with `t.Run(tt.name, ...)`.

## 2. Service

- Build the service with fake repositories (e.g. in-memory maps/slices that implement `domain.EventRepository` and `domain.SessionRepository`).
- For methods that call external HTTP (e.g. `ImportSessionizeData`): either inject an HTTP client that hits a test server returning fixed JSON, or introduce a small “Sessionize fetcher” (or similar) interface and inject a fake so unit tests do not call the real API.
- Use table-driven tests where multiple cases are needed.

## 3. Repositories

- Use **go-sqlmock**: create a mock DB with `sqlmock.New()`, set expectations (`ExpectQuery`, `ExpectExec`, `ExpectQueryRow`) and their results, call the repo method, then `require.NoError(t, mock.ExpectationsWereMet())` and assert on return values.
- Use table-driven tests for multiple SQL scenarios (e.g. success, not found, error).

## 4. After adding tests

- Run `go test ./...` from the repo root.
- Optionally run `go test -cover ./...` or `make test-cover` if available.
