---
trigger: always_on
---

# QuickTix Project Coding & Architecture Standards

These guidelines enforce strict coding, architectural, testing, and logging standards across the QuickTix codebase.

## 1. Modular Architecture Layout
Each domain module in `internal/<module_name>/` must follow the Clean Architecture layer breakdown:
- `model.go`: Domain models, request/response DTOs with `validate:"..."` tags (`go-playground/validator/v10`).
- `repository.go`: Database persistence interface and `sqlx` PostgreSQL implementation.
- `service.go`: Core business logic interface and implementation.
- `handler.go`: HTTP handler methods using `quicktix/internal/platform/response` helpers.
- `routes.go`: Implements `(h *Handler) RegisterRoutes(r *router.Router, jwtSecret string)` registering endpoints onto the application router.
- `*_test.go`: Co-located unit and integration tests.

## 2. HTTP Responses & Validation
- **Standard Envelopes**: All HTTP handlers MUST use `quicktix/internal/platform/response`:
  - Success: `response.JSON(w, http.StatusOK, data)` or `response.JSON(w, http.StatusCreated, data)`
  - Errors: `response.BadRequest(w, msg)`, `response.Unauthorized(w, msg)`, `response.Forbidden(w, msg)`, `response.NotFound(w, msg)`, `response.Conflict(w, msg)`, `response.InternalServerError(w, msg)`
  - Validation: `response.ValidationError(w, errs)` returning 400 with field-level details.
- **DTO Validation**: All request DTOs MUST execute `validator.Validate(r)` via their `.Validate()` method.

## 3. Observability & Logging (FR12.3)
- All incoming HTTP requests are assigned a unique `X-Request-ID` via `middleware.RequestID`.
- Log statements must use Go 1.24 `slog` with contextual key-value pairs (including `request_id`).

## 4. Database & Concurrency Safety
- Use parameterized SQL queries with `$1, $2` place-holders (PostgreSQL) via `sqlx`.
- Prevent overselling and race conditions using Redis distributed locks (`SETNX` with TTL) and PostgreSQL transactions (`SELECT ... FOR UPDATE` where applicable).

## 5. Version Control & Git Standard
- Commit messages MUST adhere to Conventional Commits:
  - `feat(<scope>): ...`
  - `fix(<scope>): ...`
  - `refactor(<scope>): ...`
  - `test(<scope>): ...`
  - `chore(<scope>): ...`
