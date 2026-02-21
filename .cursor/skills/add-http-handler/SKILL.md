---
name: add-http-handler
description: Add a new HTTP endpoint or handler to the API. Use when adding an endpoint, new handler, extending the API, or registering a route.
---

# Add HTTP Handler

Follow project rules in `.cursor/rules/` (Clean Architecture, Go conventions).

## When to use

- Adding a new API endpoint or HTTP handler
- Extending the API with a new route
- Registering a new route in the router

## Instructions

1. **Domain use case**: Define or extend the use case interface in `internal/domain` (e.g. `ManageScheduleUseCase`). Add the new method signature.

2. **Usecase**: Implement the method in the appropriate usecase (e.g. `internal/usecase/manage_schedule.go`). Use `context.Context`, wrap errors with `fmt.Errorf(..., %w, err)`.

3. **Controller**: Add a handler method on the controller (e.g. `ScheduleController` in `internal/delivery/http/schedule_controller.go`):
   - Use `r.PathValue("paramName")` for path parameters.
   - Validate input; return 400 via `http.Error(w, msg, http.StatusBadRequest)` for bad/missing input.
   - Call the use case; on error use `http.Error(w, err.Error(), http.StatusInternalServerError)`.
   - On success: set `w.Header().Set("Content-Type", "application/json")`, `w.WriteHeader(status)`, then `json.NewEncoder(w).Encode(...)`.

4. **Swagger**: Add a swaggo comment block above the handler: `// HandlerName godoc`, then `@Summary`, `@Description`, `@Tags`, `@Accept`/`@Produce`, `@Param`, `@Success`, `@Failure`, `@Router` (path with `{paramName}`).

5. **Router**: Register the route in `internal/delivery/http/router.go` with `mux.HandleFunc("METHOD /path/{param}", controller.HandlerName)`.

6. **Regenerate docs**: Run `make swag` from the project root.
