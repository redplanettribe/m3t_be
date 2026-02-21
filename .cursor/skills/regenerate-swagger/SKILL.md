---
name: regenerate-swagger
description: Regenerate or update Swagger/OpenAPI docs after API changes. Use when regenerating swagger, updating API docs, or after adding or changing handlers.
---

# Regenerate Swagger

Follow project rules in `.cursor/rules/` (Go conventions for swaggo).

## When to use

- After adding or changing HTTP handlers
- Regenerating or updating Swagger/OpenAPI documentation
- When API docs are out of date

## Instructions

1. Ensure handler comments use swaggo format: `// HandlerName godoc` and `@Summary`, `@Router`, etc. above each HTTP handler.

2. From the project root, run:
   ```bash
   make swag
   ```

3. This regenerates `docs/docs.go` (and related files). Swagger UI is available at `/swagger/index.html` when the server is running.
