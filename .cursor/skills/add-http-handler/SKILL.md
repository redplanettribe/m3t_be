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

1. **Domain service**: Define or extend the service interface in `internal/domain` (e.g. `ManageScheduleService`). Add the new method signature.

2. **Service**: Implement the method in the appropriate service (e.g. `internal/services/manage_schedule.go`). Use `context.Context`, wrap errors with `fmt.Errorf(..., %w, err)`.

3. **Controller**: Add a handler method on the controller (e.g. `ScheduleController` in `internal/delivery/http/schedule_controller.go`):
   - **Path params**: Use `r.PathValue("paramName")`.
   - **Request body (create/update)**:
     - Define a **request DTO** in the same package (e.g. `CreateEventRequest`) with only the fields the API accepts (e.g. `Name`, `Slug`). Do not decode into domain entities; domain fields like `id`, `created_at` are server-generated.
     - Use the same validation strategy for all handlers: call `DecodeAndValidate(w, r, &req)`; if it returns false, return immediately (it already wrote a 400 JSON error). Implement the `Validator` interface on the DTO: `Validate() []string` returning a slice of error messages (nil or empty when valid). See `internal/delivery/http/validate.go`. DecodeAndValidate uses `DisallowUnknownFields()` and runs Validate() when the DTO implements Validator; multiple validation errors are joined with "; " in one 400 response.
     - Build the domain entity from the DTO (e.g. `event := &domain.Event{Name: req.Name, Slug: req.Slug}`) and pass it to the service.
   - Call the service; on error return `WriteJSONError(w, status, code, message)` using `ErrCodeBadRequest`, `ErrCodeUnauthorized`, or `ErrCodeInternalError` as appropriate.
   - On success return `WriteJSONSuccess(w, statusCode, data)`. All responses use the standardized envelope (`APIResponse`: `data` + `error`); see `internal/delivery/http/response.go`.

4. **Swagger**: Add a swaggo comment block above the handler: `// HandlerName godoc`, then `@Summary`, `@Description`, `@Tags`, `@Accept`/`@Produce`, `@Param`, `@Success`, `@Failure`, `@Router` (path with `{paramName}`). For JSON body params use the **request DTO type** (e.g. `CreateEventRequest`), not the domain entity. Use `{object} APIResponse` for `@Success` and `@Failure` so the docs describe the standardized envelope (e.g. `@Success 201 {object} APIResponse "data contains the created resource"`).

5. **Router**: Register the route in `internal/delivery/http/router.go` with `mux.HandleFunc("METHOD /path/{param}", controller.HandlerName)`.

6. **Regenerate docs**: Run `make swag` from the project root.
