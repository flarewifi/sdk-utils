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

## Agent Workflow

### Plan First, Implement After Confirmation

1. **Research** - Use Read/Glob/Grep to understand existing patterns
2. **Consult** - Ask specialists BEFORE planning:
   - **@frontend** - Templ templates, CSS/JS, asset loading
   - **@backend** - HTTP routing, controllers, authentication
   - **@translator** - i18n, translation scanning
3. **Plan** - Create detailed implementation plan with specific file changes
4. **Confirm** - Get user approval before implementing
5. **Implement** - Only after explicit approval

### When to Escalate

**@frontend:** 2+ failed templ edits, complex template changes, CSS/JS issues
**@backend:** HTTP routing issues, form processing, authentication errors
**@translator:** Translation scanning, batch updates, i18n validation

## Critical Rules

### DO NOT
- Run docker manually (already watching files - check logs instead)
- Import `sdk/api` into `sdk/utils` (creates import cycles)
- Create test files (not part of this project)
- Use ES6+ JavaScript (ES5 only: `var`, `function() {}`)
- Modify `go.work` or `go.mod` without permission
- Hardcode user-facing text (use `api.Translate()`)
- Hardcode URLs (use `api.Http().Helpers().UrlForRoute()`)
- Modify core migrations for plugins

### ALWAYS
- Use `int64` for database IDs
- Use SQLite-compatible SQL with named params (`@param`)
- Check docker logs for `Listening on port :3000` (build success)
- Wrap URLs in templ with `templ.SafeURL()`
- Update SDK docs when modifying `sdk/api/` or `core/internal/api/`

## Build/Dev/Test

- `make` - Development build (tags: "dev")
- Docker auto-watches `*.go`, `*.templ`, `*.sql` files
- Never manually build - watch docker logs instead
- Build tags: `dev` (development), `prod` (production)
- Go build example: `go build -tags="dev" -o flare ./core/internal/cli/main.go`

## Project Structure

```
core/                     # Core application
  internal/api/          # SDK implementation
  internal/web/          # Routing, controllers
  db/queries/            # Generated sqlc code
  resources/             # Migrations, queries, views, assets, translations
sdk/
  api/                   # Plugin interfaces
  utils/                 # Shared utilities (NEVER imports sdk/api)
  mkdocs/docs/           # SDK documentation
data/plugins/local/      # Custom plugins
plugins/system/          # System plugins
```

## Tech Stack

- **Go** - Primary language (version locked to `go.work.default`)
- **gorilla/mux** - HTTP routing
- **templ** - Type-safe templates
- **sqlc** - Type-safe SQL generation
- **esbuild** - Asset bundling
- **SQLite** - Embedded database
- **Bootstrap 3.4.1** - Portal/login pages only — rendered via `PortalView`
- **Bootstrap 5.3.3** - Admin dashboard and all post-login pages — rendered via `AdminView`
- **htmx v1.9.12** - Dynamic HTML
- **Alpine.js** - Reactive components (admin)
- **jQuery** - v1.12.4 (core), v3.7.1 (theme)

## Database

- SQLite with sqlc-generated queries
- Migration format: `YYYYMMDD_NNNN_description.{up,down}.sql`
- Plugin migrations: `data/plugins/local/{plugin}/resources/migrations/`
- Plugin tables only (foreign keys to core, never alter core tables)

### MCP SQLite Access

The `mcp-sqlite` MCP server provides direct database access to `./data/db/database.sqlite` (flarehotspot db).

**Available Tools:**
- `db_info` - Get database information
- `list_tables` - List all tables
- `get_table_schema` - Get table schema (params: `tableName`)
- `read_records` - Query records (params: `table`, `conditions?`, `limit?`, `offset?`)
- `create_record` - Insert record (params: `table`, `data`)
- `update_records` - Update records (params: `table`, `data`, `conditions`)
- `delete_records` - Delete records (params: `table`, `conditions`)
- `query` - Execute custom SQL (params: `sql`, `values?`)

**Usage:** Use these tools to inspect database state, debug issues, or verify data integrity during development.

## Translations & Internationalization

**ALL user-facing text must use translations API**

```go
// Go
api.Translate("label", "Username")
api.Translate("error", "Invalid input", "field", "email")

// Templ
<h1>{ api.Translate("label", "Sessions") }</h1>
<button>{ api.Translate("label", "Save") }</button>
```

### Translation Types
- `"label"` - UI labels, buttons, form fields
- `"error"` - Error messages
- `"success"` - Success messages  
- `"info"` - Informational messages
- `"warning"` - Warning messages

### Key Rules
- No snake_case (use natural language)
- Max 120 chars (auto-truncated beyond this)
- Punctuation allowed

## Frontend Development

### CSS
- **Bootstrap 3.4.1** - Portal/login pages only (`PortalView`)
- **Bootstrap 5.3.3** - Admin/dashboard and all post-login pages (`AdminView`)
- Never mix versions — check which Go view function renders the page

### JavaScript (ES5 Only)
- Use `var` not `let`/`const`
- Use `function() {}` not arrow functions
- Use string concatenation, not template literals
- No ES6+ features

### Assets
- Manifests: `manifest.admin.json`, `manifest.portal.json`
- Global assets (`global.js`, `global.css`) load automatically

### URL Generation
```templ
<a href={ templ.SafeURL(api.Http().Helpers().UrlForRoute("admin:sessions:index")) }>
```

## Plugin Development

### Entry Point
```go
package main
import sdkapi "github.com/flarehotspot/sdk-api"

var Api sdkapi.IPluginApi

func Init(api sdkapi.IPluginApi) error {
    Api = api
    // Register routes, navigation, etc.
    return nil
}
```

### Structure
```
data/plugins/local/{plugin-name}/
├── main.go                   # Plugin entry point
├── plugin.json               # Plugin metadata
├── sqlc.yml                  # SQLite configuration
├── db/queries/               # Generated sqlc code
└── resources/
    ├── assets/
    │   ├── admin/           # Bootstrap 5 assets
    │   ├── portal/          # Bootstrap 3 assets
    │   ├── manifest.admin.json
    │   └── manifest.portal.json
    ├── migrations/          # Plugin migrations ONLY
    ├── queries/             # Plugin SQL queries
    ├── translations/        # i18n files
    └── views/               # Plugin templates
```

### Available APIs
- `api.SqlDB()` - Database access
- `api.Http()` - HTTP routing, responses, forms
- `api.Translate()` - i18n
- `api.Machine()` - System info, network interfaces
- `api.Config()` - Configuration
- `api.Logger()` - Logging
- `api.Network()` - Network operations
- `api.SessionsMgr()` - Session management
- `api.PluginsMgr()` - Plugin management
- `api.UI()` - UI components
- `api.Notification()` - Notifications
- Full reference: `sdk/mkdocs/docs/`

## Common Scenarios

### Adding a New Feature
1. Research existing patterns
2. Default to plugin (not core)
3. Consult specialists (@frontend, @backend, @translator)
4. Create detailed plan
5. Get user confirmation
6. Implement
7. Verify build in docker logs

### Core Modification
- Only with explicit user request/confirmation
- Provide detailed justification
- Consider plugin alternative first
- Update SDK docs if APIs change

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

## UI Verification with Playwright

Use Playwright MCP against `http://localhost:3000`:

```
browser_navigate → browser_snapshot → browser_click/type → browser_take_screenshot
```

- Save outputs to `.tmp/playwright/`
- Use accessibility snapshots as primary inspection method
- Test end-to-end flows
- Check both admin (Bootstrap 5) and portal (Bootstrap 3)
- Verify translations display correctly
- Close browser when done
