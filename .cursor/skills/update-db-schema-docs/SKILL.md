---
name: update-db-schema-docs
description: Update the DBML schema file after migration changes. Use when adding or changing migrations, or when docs/database/schema.dbml is out of sync with migrations/*.up.sql.
---

# Update DB Schema Docs (DBML)

Keep `docs/database/schema.dbml` in sync with the actual schema defined in `migrations/*.up.sql` so diagrams (e.g. dbdiagram.io) stay accurate.

## When to use

- After adding a new migration (new or changed tables/columns/indexes)
- After editing an existing `.up.sql` migration
- When the user asks to update or regenerate the database schema description

## Instructions

1. **Source of truth**: Read all `migrations/*.up.sql` files in order (by migration number). The combined schema is what the DBML must describe (ignore `INSERT` and other non-DDL; only `CREATE TABLE`, `CREATE INDEX`, and constraints).

2. **Target file**: Update [docs/database/schema.dbml](docs/database/schema.dbml).

3. **DBML mapping from PostgreSQL**:
   - **Tables**: One `Table name { ... }` block per `CREATE TABLE`.
   - **Columns**: `column_name type [pk, not null, unique, default: \`expression\`]`. Use `ref: > other_table.id` for `REFERENCES other_table(id)` (many-to-one).
   - **Types**: `uuid`, `varchar(n)`, `text`, `int` / `integer`, `timestamptz` (for `TIMESTAMP WITH TIME ZONE`), `boolean`.
   - **Primary key**: `[pk]` on the column, or in junction tables use `indexes { (col_a, col_b) [pk] }`.
   - **Unique**: `[unique]` on column, or `indexes { (col_a, col_b) [unique] }` for composite.
   - **Indexes**: `indexes { column_name }` or `indexes { (a, b) [unique] }`. Match `CREATE INDEX` and `UNIQUE(...)` from migrations.
   - **Relationships**: `ref: > table.id` for foreign key to one; both sides for many-to-many junction tables.

4. **Header**: Keep the top-of-file comment that says to keep in sync with migrations and points to this skill.

5. **Verify**: Ensure every table, column, FK, and index from the migrations appears in the DBML (no extra tables; seed data like `INSERT INTO roles` is not modeled in DBML).

## Quick reference

- One-to-many: child table has `ref: > parent_table.id`.
- Many-to-many: junction table has two `ref: >` columns and `indexes { (id_a, id_b) [pk] }`.
- Composite unique: `indexes { (event_id, source_session_id) [unique] }`.
