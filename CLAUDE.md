# AGENTS.md

## About this project

- This is a Go application that runs in OpenWRT routers
- Uses SQLite as the database (lightweight, embedded, perfect for edge devices)
- Plugin-based architecture allows third-party developers to extend functionality
- Core should remain minimal and stable - new features should be plugins when possible

## ⚠️ CRITICAL: Core vs Plugin Development

### Default Strategy: Prefer Plugin Development

**ALWAYS assume the user wants plugin-specific features unless explicitly told otherwise.**

- ✅ **New features** → Create as plugins
- ✅ **Custom functionality** → Develop in plugins
- ✅ **Optional features** → Plugin architecture
- ❌ **Core modifications** → Only with explicit user confirmation

### When to Modify Core

Only modify core files when:
- User **explicitly requests** core modification
- User **explicitly confirms** your plan to modify core
- Fixing bugs in existing core functionality
- Adding foundational APIs that plugins will use
- Modifying shared infrastructure (HTTP server, database layer, plugin system)

### Core File Protection

**NEVER modify without explicit permission:**
- `core/resources/migrations/` - **CRITICAL**: Core database schema
- `core/resources/queries/` - Core SQL queries
- `core/resources/views/` - Core view templates
- `core/internal/api/` - Core API implementation
- `sdk/` - Plugin API and utilities
- `tools/` - Build tools

**Safe to modify (plugin development):**
- `data/plugins/local/{plugin-name}/` - Plugin-specific code
- `plugins/system/{plugin-name}/` - System plugins (when explicitly developing that plugin)

## Critical Rules

### DO NOT

- Do not run docker container to check if the build succeeds - docker is already running and watching files
- NEVER import `sdk/api` into `sdk/utils` - this creates import cycles. `sdk/utils` should only contain basic utilities and types, while `sdk/api` can import from `sdk/utils`
- NEVER modify core migrations when building plugin features - each plugin must have its own migrations
- NEVER hardcode user-facing text - always use the translations API (`api.Translate()`)
- NEVER modify `go.work` or `go.mod` files without explicit permission - these control critical dependencies
- NEVER create any `SOME_RANDOM_SUMMARY.md` files after performing file modifications

### ALWAYS

- Use translations API for ALL user-facing text
- Use SQLite-compatible SQL syntax in all queries
- Use `int64` for database IDs
- Use ES5 JavaScript syntax (no ES6+)
- Check docker logs to verify builds (`Listening on port :3000` = success)
- Prefer plugin development over core modifications
- Use named parameters in SQL queries (`@parameter_name`)
- **Update SDK documentation** (`sdk/mkdocs/docs/`) when modifying core code that affects the plugin API (interfaces in `sdk/api/`, implementations in `core/internal/api/`)

## Build/Dev/Test

- `make` Runs the development app with Go build tags "dev"
- We only use `ES5` syntax in our javascript assets for maximum browser compatibility
- We don't implement or create test files and unit tests
- The `go`, `templ`, and `sqlc` files are being watched and built by the running docker container
- We don't build go, templ and sqlc files. Instead, we watch for the docker logs to see if the build succeeds
- Watch docker logs but don't run them. When it prints `Listening on port :3000 ` it means the build is successful
- When building the source, we use build tags "dev" for development or "prod" for production
- Sample Go build command: `go build -tags="dev" -o flare ./core/internal/cli/main.go`

## Project Structure

- `go.work.default` - Copied to `go.work`, to be able to work on multiple Go modules
- `scripts/` - Scripts that need to run outside of Go context
- `sdk/utils/` Go utilities that can be reused in the core and plugins
- `sdk/api/` Go interfaces and structs API to build a plugin
- `sdk/mkdocs/` Documentation for the `sdk/api/` usage
- `core/` The core of the system, it initializes the application and all the installed plugins
- `core/internal/api/` Contains the implementation of `sdk/api/`
- `core/db/` Contains the Go database queries generated from `core/resources/queries/`
- `core/resources/assets/` Contains the javascript and css
- `core/resources/views/` Contains the `templ` files for our views
- `core/internal/web/` Contains routing, navigation, middlewares and controllers/handlers
- Each plugin has a corresponding `resources` directory similar to `core/resources/`

## Tech Stack

- Using `Go` as primary programming language

- We are not allowed to exceed the go tool chain version defined in `go.work.default` when installing new libraries

- `docker compose` to run the app and database for easy development setup

- `gorilla/mux` for handling the routes

- `templ` for our views

- `sqlc` for our database queries

- `esbuild` Go API for bundling our assets

- `@Makefile` To run common commands

## Database

- Database queries are generated using `sqlc` in `./scripts/sqlc-gen.sh`
- We use SQLite as our database (embedded, lightweight, perfect for edge devices)
- We use `sqlc` named params in our sql queries. For example: `select * from devices where mac_address = @mac_address`
- All queries must use SQLite-compatible syntax
- Column overrides are configured in `core/sqlc.yml`
- For IDs, we use `int64` type

### Plugin-Specific Migrations

- **CRITICAL**: Each plugin must have its own migrations directory (e.g., `data/plugins/local/{plugin-name}/resources/migrations/`)
- Plugin migrations should **only** create tables/schemas specific to that plugin
- Use proper foreign key constraints to reference core tables, but **never alter core tables**
- If a plugin needs data from core tables, use JOIN queries instead of modifying core schema
- Plugin queries should be in the plugin's own `resources/queries/` directory

## Translations & Internationalization

### Critical Rule: Always Use Translations API

**ALL user-facing text must use the translations API** - no hardcoded strings allowed.

### In Go Code
```go
// Flash messages
api.Http().Response().FlashMsg(
    w, r,
    api.Translate("error", "Failed to create session"),
    sdkapi.FlashMsgError,
)

// Error messages
errorMsg := api.Translate("error", "Invalid input", "field", "email")

// Labels and UI text
label := api.Translate("label", "Username")
```

### In Templ Templates
```templ
// Page titles
<h1>{ api.Translate("label", "Sessions") }</h1>

// Table headers
<th>{ api.Translate("label", "Device") }</th>

// Buttons and links
<button>{ api.Translate("label", "Save") }</button>

// Form placeholders
<input placeholder={ api.Translate("label", "Enter name") }/>
```

### In JavaScript
- User-facing strings (alerts, notifications, UI labels) must be translated
- Pass translated strings from backend or use translation endpoint
- Debug/console logs can remain in English (not user-facing)

### Translation Types
- `"label"` - UI labels, buttons, navigation, form fields
- `"error"` - Error messages shown to users
- `"success"` - Success messages
- `"info"` - Informational messages
- `"warning"` - Warning messages
- Custom types as needed for your plugin

### Translation Key Rules

**1. No Snake_case**
- ❌ `api.Translate("error", "invalid_form_values")`
- ✅ `api.Translate("error", "Invalid form values")`

**2. Key Length Limit: 10 Words**
- Keys with >10 words are automatically truncated to 10 words + " (truncated)"
- Build warnings: 8-10 words = INFO, 11+ words = WARNING
- Shorter keys preferred for readability

**3. Punctuation Allowed in Keys**
- ✅ `api.Translate("info", "You are connected.")` - Punctuation is fine
- ✅ `api.Translate("error", "Are you sure?")` - Question marks are fine
- Translation filenames will match the key exactly (no `.txt` extension)

### Exception: Debug Logs
- Internal debug logs and development console output can remain in English
- Anything displayed to end users **must** be translated

## Frontend Development

### CSS Frameworks
- **Bootstrap 3.4.1** - Portal pages only (captive portal for end users)
- **Bootstrap 5.3.3** - Admin dashboard only
- Never mix versions - check which section you're working in

### JavaScript
- **ES5 syntax only** - Maximum browser compatibility for embedded devices
- Use `var` instead of `let`/`const`
- Use `function() {}` instead of arrow functions `() => {}`
- No template literals - use string concatenation
- No modern ES6+ features

### Libraries
- **htmx v1.9.12** - Primary dynamic HTML framework
- **Alpine.js** - Reactive components (admin dashboard)
- **jQuery** - v1.12.4 (core), v3.7.1 (theme)

### Asset Loading
- Assets defined in `manifest.admin.json` and `manifest.portal.json`
- Use `ViewAssets` struct to specify JS/CSS per page
- Global assets (`global.js`, `global.css`) load automatically
- Docker watch mode auto-rebuilds assets

### URL Generation
- **NEVER hardcode URLs** - always use `api.Http().Helpers().UrlForRoute()`
- Route naming: `section:subsection:action` (e.g., `"admin:plugins:install"`)
- Parameters as key-value pairs: `UrlForRoute("admin:device:info", "id", deviceID)`
- Wrap with `templ.SafeURL()` for href/action attributes
