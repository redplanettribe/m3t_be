---
name: add-repository
description: Add a new repository or data access layer for an entity. Use when adding a repository, new entity storage, or data access for a domain entity.
---

# Add Repository

Follow project rules in `.cursor/rules/` (Clean Architecture, Go conventions).

## When to use

- Adding a new repository or data access for an entity
- Creating storage for a new domain entity
- Implementing a new domain repository interface

## Instructions

1. **Domain**: In `internal/domain`, define the entity struct (if new) and the repository interface. Interface methods take `context.Context` and domain types; e.g. `Create(ctx context.Context, e *Entity) error`, `GetByID(ctx context.Context, id string) (*Entity, error)`.

2. **Postgres implementation**: Create `internal/repository/postgres/xxx_repo.go`:
   - Unexported struct (e.g. `xxxRepository`) with field `DB *sql.DB`.
   - Constructor `NewXxxRepository(db *sql.DB) domain.XxxRepository` returning the interface.
   - Implement all interface methods using **raw SQL** only (`database/sql`; no ORM). Use `QueryRowContext`/`ExecContext`/`QueryContext` as appropriate.

3. **Wire in main**: In `cmd/api/main.go`, instantiate the new repo (e.g. `xxxRepo := postgres.NewXxxRepository(db)`) and pass it into the use case constructor. If the use case interface does not yet accept this repo, extend the domain use case interface and the usecase implementation to accept and use it.

4. **Use case (if needed)**: If the new repo is used by an existing use case, add it to the use case constructor and struct in `internal/usecase`, and call the repo from the use case methods as needed.
