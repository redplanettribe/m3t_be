---
name: add-migration
description: Add a new database migration for schema changes. Use when adding a migration, schema change, or new table/column.
---

# Add Migration

Follow project rules in `.cursor/rules/` (migrations).

## When to use

- Adding a new database migration
- Changing schema (new table, column, index)
- Creating up/down migration files

## Instructions

1. **Next sequence number**: List existing files in `migrations/` and choose the next 6-digit number (e.g. after `000001_...` use `000002_...`).

2. **Create paired files**:
   - `migrations/NNNNNN_short_description.up.sql` — SQL to apply the change (CREATE TABLE, ALTER TABLE, etc.).
   - `migrations/NNNNNN_short_description.down.sql` — SQL to revert the change (DROP TABLE, ALTER TABLE revert, etc.).

3. **Use raw SQL only**; no ORM or application code.

4. **Apply**: Run `make migrate-up` from the project root to run the new migration.
