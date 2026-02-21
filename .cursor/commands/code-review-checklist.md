# Code Review Checklist

## Overview

Comprehensive checklist for conducting thorough code reviews to ensure quality, security, and maintainability.

## Review Categories

### Functionality

- [ ] Code does what it's supposed to do
- [ ] Edge cases are handled
- [ ] Error handling is appropriate
- [ ] No obvious bugs or logic errors

### Code Quality

- [ ] Code is readable and well-structured
- [ ] Functions are small and focused
- [ ] Variable names are descriptive
- [ ] No code duplication
- [ ] Follows project conventions

### Security

- [ ] No obvious security vulnerabilities
- [ ] Input validation is present
- [ ] Sensitive data is handled properly
- [ ] No hardcoded secrets

### Project conventions

Verify against this codebase:

- [ ] Clean Architecture layers respected (domain → services → repo/delivery)
- [ ] No ORM (raw SQL in `internal/repository/postgres`)
- [ ] Errors wrapped with `fmt.Errorf(..., %w, err)` in services
- [ ] HTTP errors via `http.Error`, 400 for bad input and 500 for service/repo failures
- [ ] New/changed handlers have swaggo comments and `make swag` was run
- [ ] Migrations are paired up/down in `migrations/`
- [ ] Logging follows [.cursor/rules/logging.mdc](.cursor/rules/logging.mdc)

See `.cursor/rules/` for details.
