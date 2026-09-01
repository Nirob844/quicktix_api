---
name: migration-workflow
description: >-
  Step-by-step workflow for creating, applying, and rolling back database schema migrations in QuickTix using golang-migrate and Makefile.
---

# Database Migration Workflow Guide

Use this skill whenever you add or modify database tables in PostgreSQL.

## Migration Command Reference

### 1. Create a New Migration File Pair
To create a new `.up.sql` and `.down.sql` migration file pair:
```bash
make migrate-create NAME=create_<feature>_table
```
This generates files under `migrations/`:
- `migrations/00000X_create_<feature>_table.up.sql`
- `migrations/00000X_create_<feature>_table.down.sql`

### 2. Standard Up Migration SQL Rules
- Always use `CREATE TABLE IF NOT EXISTS`.
- Explicitly define column types, `NOT NULL` constraints, default timestamps `NOW()`, and foreign keys.
- Create explicit indexes for foreign keys and frequent query columns (`WHERE email = $1`, `WHERE user_id = $1`).

### 3. Programmatic Auto-Migration on Server Startup
The API server automatically runs pending migrations on container startup using `database.RunMigrationsPath(db.DB, "migrations")` in `cmd/api/main.go`.

### 4. Manual Execution
- Apply all pending migrations:
  ```bash
  make migrate-up
  ```
- Rollback the last migration:
  ```bash
  make migrate-down
  ```
