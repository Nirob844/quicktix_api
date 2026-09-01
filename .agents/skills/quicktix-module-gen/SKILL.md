---
name: quicktix-module-gen
description: >-
  Standardized guide for creating a new domain module in QuickTix (e.g., events, tickets, orders, payments).
  Provides the exact file structure, layer interfaces, DTO validation, and route registration pattern.
---

# QuickTix Domain Module Generation Guide

Use this skill whenever you need to create a new domain module under `internal/<module_name>/`.

## Module File Structure
For a module named `<module>`:

```text
internal/<module>/
├── model.go         # Struct definitions, DTOs, validator tags
├── repository.go    # SQL database interface and sqlx PostgreSQL implementation
├── service.go       # Business logic interface and implementation
├── handler.go       # HTTP handler layer using platform/response helpers
├── routes.go        # RegisterRoutes method for HTTP router binding
└── service_test.go  # Unit test suite
```

## Step-by-Step Implementation Recipe

### Step 1: Database Migration
Create up/down migration files under `migrations/`:
```bash
make migrate-create NAME=create_<module>_table
```

### Step 2: Define Models & DTOs (`model.go`)
- Define the primary domain entity struct matching PostgreSQL table columns.
- Define request and response DTO structs.
- Add `validate:"..."` tags (`go-playground/validator/v10`).
- Implement `Validate() map[string]string` calling `validator.Validate(r)`.

### Step 3: Define Repository Layer (`repository.go`)
- Declare `Repository` interface.
- Implement PostgreSQL repository struct using `*sqlx.DB`.
- Map PostgreSQL errors (e.g. `23505` unique violation) to domain errors.

### Step 4: Define Service Layer (`service.go`)
- Declare `Service` interface.
- Implement business logic with `Repository`, `redis.Client` (if caching/locking needed), and logger.

### Step 5: Define Handler Layer (`handler.go`)
- Implement HTTP handlers.
- Decode JSON body, execute `req.Validate()`.
- Use `response.ValidationError(w, valErrs)` if validation fails.
- Use `response.JSON`, `response.BadRequest`, `response.NotFound`, `response.InternalServerError` for outputs.

### Step 6: Define Modular Routes (`routes.go`)
- Implement `(h *Handler) RegisterRoutes(r *router.Router, jwtSecret string)`.
- Wrap protected endpoints with `middleware.Authenticate(jwtSecret)` and RBAC guards (`middleware.RequireRole(...)`).

### Step 7: Wire in `cmd/api/main.go`
- Initialize repository, service, handler, and call `<module>Handler.RegisterRoutes(r, jwtSecret)`.
