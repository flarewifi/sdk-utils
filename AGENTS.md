# AGENTS.md

## About this project

- Go application for OpenWRT routers using SQLite (embedded, lightweight)
- Plugin-based architecture - core remains minimal, features go in plugins

## ⚠️ CRITICAL: Core vs Plugin Development

**DEFAULT: Plugin Development** (unless user explicitly requests core modification)

### Protected Core Files (NEVER modify without explicit permission):
- `core/resources/migrations/` - Core database schema
- `core/resources/queries/` - Core SQL queries  
- `core/resources/views/` - Core view templates
- `core/internal/api/` - Core API implementation
- `sdk/` - Plugin API and utilities
- `tools/` - Build tools

### Safe to Modify:
- `data/plugins/local/{plugin-name}/` - Plugin-specific code
- `plugins/system/{plugin-name}/` - System plugins

## Workflow

1. **Research** - Use Read/Glob/Grep to understand existing patterns
2. **Consult** specialists: @frontend (templates/CSS/JS), @backend (routing/auth), @translator (i18n)
3. **Plan** - Detailed implementation plan with specific file changes
4. **Confirm** - Get user approval before implementing
5. **Implement** - Code the solution
6. **Review** - Check for errors, logic bugs, security issues (see checklist below)

### Implementation Review Checklist (MANDATORY)

**Error Handling:**
- ✅ Catch ALL errors, no silent failures
- ✅ Rollback on partial failures (e.g., `CreateSession()` → `RecordUsage()` fails → `DeleteSession()`)
- ❌ NEVER `_ = functionThatCanError()` or log-and-continue for critical operations

**Logic & Security:**
- ✅ Race conditions (add DB unique constraints)
- ✅ Data consistency (transaction-like behavior)
- ✅ Boundary conditions (time ranges: use `endTime.Add(59s, 999ms)`)
- ✅ Authorization & input validation
- ✅ Re-validate before actions (not just at UI)

**Example:**
```go
// ❌ BAD: Silent error, data inconsistency
session := CreateSession()
RecordUsage()  // If fails, inconsistent state!

// ✅ GOOD: Rollback on failure
session, err := CreateSession()
if err := RecordUsage(); err != nil {
    DeleteSession(session.ID())  // Rollback
    return err
}
```

## Critical Rules

**DO NOT:**
- Import `sdk/api` into `sdk/utils` (import cycle)
- Use ES6+ JavaScript (ES5 only: `var`, `function() {}`)
- Hardcode text/URLs (use `api.Translate()` / `api.Http().Helpers().UrlForRoute()`)
- Modify core files without permission
- Discard errors or create resources without rollback

**ALWAYS:**
- Use `int64` for IDs, named params (`@param`) for SQL
- Wrap URLs with `templ.SafeURL()`
- Handle ALL errors, implement rollback for multi-step ops
- Add DB constraints (UNIQUE, FOREIGN KEY) for business rules
- Check docker logs for `Listening on port :3000`

## Build/Dev

- `make` - Development build (tags: "dev")
- Docker auto-watches `*.go`, `*.templ`, `*.sql` - check logs, never build manually

## Structure

```
core/internal/api/       # SDK implementation (protected)
core/resources/          # Migrations, queries, views (protected)
sdk/api/                 # Plugin interfaces
sdk/utils/               # Shared utilities (NEVER imports sdk/api)
data/plugins/local/      # Custom plugins (safe to modify)
```

## Tech Stack

- **Go**, **SQLite**, **templ**, **sqlc**, **gorilla/mux**
- **Bootstrap 3.4.1** - Portal/login (`PortalView`)
- **Bootstrap 5.3.3** - Admin/dashboard (`AdminView`)
- **htmx v1.9.12**, **Alpine.js**, **jQuery**

## Database

- SQLite with sqlc-generated queries
- Migrations: `YYYYMMDD_NNNN_description.{up,down}.sql`
- Plugins: Create own tables with foreign keys to core, never alter core tables
- MCP SQLite tools available: `db_info`, `list_tables`, `read_records`, `query`, etc.

## Translations

**ALL user-facing text:** `api.Translate("label", "Username")` or `api.Translate("error", "Invalid input")`

Types: `label`, `error`, `success`, `info`, `warning` | Max 120 chars, natural language (no snake_case)

## Frontend

**CSS:** Bootstrap 3 (portal) vs Bootstrap 5 (admin) - never mix  
**JavaScript:** ES5 only (`var`, `function() {}`, no template literals)  
**Interactivity:** Use htmx and Alpine.js - avoid custom JavaScript when possible  
**Real-time Updates:** Use Server-Sent Events (SSE) for live UI updates, not polling  
**URLs:** `templ.SafeURL(api.Http().Helpers().UrlForRoute("route:name"))`

## Plugin Development

**Entry:** `func Init(api sdkapi.IPluginApi) error` in `main.go`  
**Structure:** `data/plugins/local/{plugin}/` → `main.go`, `plugin.json`, `resources/{migrations,queries,views,assets,translations}`  
**APIs:** `api.SqlDB()`, `api.Http()`, `api.Translate()`, `api.SessionsMgr()`, etc. (see `sdk/mkdocs/docs/`)

### Common Functions & Helpers

**ALWAYS check `sdk/utils/` first before creating custom functions:**
- UUID generation, string/slice helpers, formatters
- Pagination, validators, file system operations
- Payment utilities, translations, retry logic
- Database utilities, system info (OpenWRT, OS release)

Only create custom functions if needed functionality doesn't exist in `sdk/utils/`



## Common Pitfalls

| Problem | Solution |
|---------|----------|
| Import cycle `sdk/api` → `sdk/utils` | Move types to `sdk/utils`, keep interfaces in `sdk/api` |
| Build failure after edit | Check docker logs, fix syntax, wait for auto-rebuild |
| Hardcoded text showing English | Use `api.Translate()` for ALL user-facing text |
| 404 on plugin route | Check route registration in `Init()` function |
| ID type errors | Always use `int64` for IDs |
| Plugin modifying core tables | Create plugin tables with foreign keys, use JOINs |
| URL not working in templ | Wrap with `templ.SafeURL()` |
| Asset not loading | Check manifest.json matches `ViewAssets{JsFile: "key"}` |
| ES6 syntax error | Convert to ES5: `var`, `function() {}` |
| 2+ templ edit failures | Stop and consult @frontend |
| Creating custom helper function | Check `sdk/utils/` first (UUID, strings, validators, pagination, etc.) |
| **Silent error in critical path** | **ALWAYS handle errors; rollback on failure** |
| **Data inconsistency** | **Implement transaction-like behavior with rollback** |
| **Race condition in check-then-act** | **Add DB unique constraints; re-validate before action** |
| **Inclusive time range bug** | **Use `endTime.Add(59s, 999ms)` or `<=` comparison** |
| **Skipping implementation review** | **Review EVERY implementation before completion** |

## UI Testing

Playwright MCP (`http://localhost:3000`): `browser_navigate` → `browser_snapshot` → test → verify both admin/portal
