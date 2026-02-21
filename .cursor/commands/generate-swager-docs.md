# Generate Swagger docs

Workflow to regenerate Swagger/OpenAPI documentation after API or handler changes.

## Prerequisites

- **swag** CLI installed: `go install github.com/swaggo/swag/cmd/swag@latest`
- Handler comments in swaggo format (see [add-http-handler](.cursor/skills/add-http-handler/SKILL.md) and [regenerate-swagger](.cursor/skills/regenerate-swagger/SKILL.md))

## Steps

1. **Ensure handlers are documented**  
   Each HTTP handler should have a comment block with:
   - `// HandlerName godoc`
   - `@Summary`, `@Description`, `@Tags`
   - `@Accept` / `@Produce` (e.g. `application/json`)
   - `@Param` for path/query/body
   - `@Success` / `@Failure` (use `{object} APIResponse` for the envelope)
   - `@Router` with path and method (e.g. `GET /schedules/{id}`)

2. **Regenerate docs from project root**

   ```bash
   make swag
   ```

   This runs: `swag init -g cmd/api/main.go -o docs`

3. **Result**  
   - `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` are updated.  
   - Do not edit these files by hand.

4. **Verify**  
   Start the API and open [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).

## When to run

- After adding or changing HTTP handlers or routes
- When API docs are out of date
- Before opening a PR that touches the API (see [code-review-checklist](.cursor/commands/code-review-checklist.md))
