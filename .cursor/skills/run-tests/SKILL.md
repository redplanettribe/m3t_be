---
name: run-tests
description: Run Go tests, produce coverage, and fix failing tests using table-driven style and project testing rules. Use when running tests, fixing test failures, or generating coverage reports.
---

# Run Tests

Align with [.cursor/commands/run-all-tests-and-fix.md](.cursor/commands/run-all-tests-and-fix.md) and [.cursor/rules/testing.mdc](.cursor/rules/testing.mdc).

## Steps

1. **Run unit tests**: From the repo root, run `go test ./...`. Capture output and note any failures.
2. **Coverage (if needed)**: Run `go test -coverprofile=coverage.out ./...`. View with `go tool cover -html=coverage.out` or use `make test-cover` if defined.
3. **Fix failures**: For each failure, fix code or tests following [.cursor/rules/testing.mdc](.cursor/rules/testing.mdc) and [.cursor/rules/go-conventions.mdc](.cursor/rules/go-conventions.mdc). Prefer table-driven tests; use `require`/`assert`; ensure no real external calls or DB in unit tests. Re-run `go test ./...` after each change until all tests pass.
