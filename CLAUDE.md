# AGENTS.md

## About this project

- Go application for OpenWRT routers using SQLite (embedded, lightweight)
- Plugin-based architecture - core remains minimal, features go in plugins
- Plugins in `data/plugins/local/*` are in **separate git repositories** (not tracked by this repo's git — they are gitignored)

## Terminology: "machine" vs "device"

- **"machine"** = the OpenWRT router/hardware that runs this app (the hotspot box). Use "machine" for anything about the host itself: its internet connectivity, boot/flash, install scripts, plugin installation, machine ID/licensing, online monitor, etc. See `IMachineApi` (`api.Machine()`).
- **"device"** = a **client device** (a client host such as a phone/laptop) connecting *through* the machine. Reserve "device" for these: `IClientDevice`, `client:*` events, sessions, vouchers, MAC/IP of a client, etc.
- ⚠️ "device" alone is ambiguous — never use it to mean the machine. In docs (`sdk/mkdocs`), comments, log/user-facing text, and translations, say "machine" when you mean the router and "client device" when you mean a client host. Example fixed in `sdk/mkdocs`: `OnInternetEvent` / `plugin.json` install-script docs now say "machine has internet", "machine went offline", "production machine".

## Naming & branding

- ⚠️ Spell the product name **"Flarewifi"** (capital F, rest lowercase) — **never "FlareWiFi"**, "FlareWifi", or "flarewifi" (except in code identifiers/paths). Apply this in all docs (`sdk/mkdocs`), comments, and user-facing text/translations.

## Plugin README files

- ⚠️ Each plugin's `README.md` (`data/plugins/{local,devel}/{plugin}/README.md`) is **marketing copy for the plugin marketplace/store listing** — written for **plugin buyers** (machine operators deciding whether to install it), not for developers. It is NOT technical documentation.
- Keep it short (roughly 15-25 lines), lead with the business benefit, and list features as plain-language bullets. Do not describe internal architecture, SDK/API calls, database/table details, file paths, config field tables, or step-by-step "how it works" internals — that belongs in code comments or a separate developer doc, not the README.
- Follow the same "Flarewifi" spelling and machine/device terminology conventions above in README copy.

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
7. **Document** - Update `sdk/mkdocs/docs/api/` when adding or changing SDK API methods

### Implementation Review Checklist (MANDATORY)

**Error Handling:**
- ✅ Catch ALL errors, no silent failures
- ✅ Rollback on partial failures (e.g., `CreateSession()` → `RecordUsage()` fails → `DeleteSession()`)
- ❌ NEVER `_ = functionThatCanError()` or log-and-continue for critical operations
- ✅ When logging errors, wrap with generic but descriptive messages; never expose twrip URLs, domains, secrets, or important file locations

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
- Vendor your own Bootstrap or Alpine — both are core-provided globals (see "Frontend libraries" below)
- Hardcode text/URLs (use `api.Translate()` / `api.Http().Helpers().UrlForRoute()`)
- Modify core files without permission
- Discard errors or create resources without rollback

**ALWAYS:**
- Use `int64` for IDs, named params (`@param`) for SQL
- Wrap URLs with `templ.SafeURL()`
- Handle ALL errors, implement rollback for multi-step ops
- Add DB constraints (UNIQUE, FOREIGN KEY) for business rules
- Check docker logs for `Listening on port :3000`
- Convert database timestamps from UTC to local time when displayed to UI using `sdkutil.UtcToLocalTime(t)`
- Put temporary compiled/binary artifacts in `.tmp/` so they are not tracked by git
- Execute shell commands through the `core/utils/shell` util (`shell.Exec` / `ExecOutput` / `ExecAll` / `ExecWithContext`), never `os/exec` directly — it has a `dev` build-tag variant (`exec_dev.go`) so commands are mocked/guarded in the dev container, and it centralizes error/output handling. For best-effort calls that may fail (e.g. a missing tool), append `2>/dev/null || true` to the command string (see `nftables.go`).
- Use `api.Logger()` (`.Info()` / `.Debug()` / `.Error()`) for all debugging/diagnostic output in core and plugin code — never `fmt.Println`/`log.Println`/raw stdout prints. `api.Logger()` writes to stdout (captured by `docker logs` in dev and syslog/logread on a real device), the rotating `app.log` file the admin log viewer reads, and live SSE subscribers, all from one call (`core/internal/modules/logger.Emit`) — ad hoc prints bypass the file/SSE sinks and the admin viewer's package/level filtering entirely. Prefer `.Debug()` for step-by-step breadcrumbs (e.g. "about to run X", "X succeeded") so a hang or failure mid-sequence shows exactly which step was reached, not just a final wrapped error.

## Go File Organization

**Standard ordering for Go files:**
```
1. package declaration
2. imports
3. constants
4. variables
5. types/structs
6. constructor (New*) functions
7. PUBLIC methods (exported, capitalized)
8. HELPER functions (unexported, lowercase) at BOTTOM
```

**Example:**
```go
package example

import "context"

const maxItems = 100

type Service struct { ... }

func NewService() *Service { ... }

// PUBLIC METHODS (exported)
func (s *Service) DoSomething(ctx context.Context) error { ... }
func (s *Service) GetData() []Item { ... }

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func (s *Service) validateInput(input string) bool { ... }
func (s *Service) formatOutput(data []byte) string { ... }
```

## Build/Dev & Auto-Rebuild System

### Development Workflow

The development environment uses **reflex** to watch for file changes and automatically rebuild:

**Watched file types:**
- `.go` - Go source files
- `.templ` - Template files (generates `*_templ.go`)
- `.sql` - SQL query files (generates `db/queries/*.go` via sqlc)
- `.js`, `.css` - Frontend assets
- `.json` - Config files (except plugin.json, package.json)
- `.sh` - Shell scripts

**Excluded from watch** (to prevent rebuild loops):
- `*_templ.go` - Generated by templ
- `db/queries/*` - Generated by sqlc
- `node_modules/`, `data/config/`, `data/storage/`
- `plugin.json`, `package.json`, `package-lock.json`

### Build Process (Automatic)

When you edit a watched file, reflex triggers this sequence:

1. **Copy workspace:** `go.work.default` → `go.work`
2. **Clean generated files:** Remove old `*_templ.go` files
3. **Fix workspace:** Run `flare fix-workspace` (updates module references)
4. **Build plugins:** Run `flare build-plugins` (compiles all plugins)
5. **Start server:** Run `flare server`
6. **Wait for:** `Listening on port :3000` in logs

**Typical rebuild time:** 5-15 seconds depending on changes

### What Gets Auto-Generated

| You Edit | Auto-Generated | Tool |
|----------|----------------|------|
| `*.templ` | `*_templ.go` | templ |
| `resources/queries/*.sql` | `db/queries/*.sql.go` | sqlc |
| `resources/queries/*.sql` | `db/queries/models.go` | sqlc |
| Plugin files | Plugin binaries | Go build |

### Troubleshooting Rebuilds

**No rebuild triggered:**
- ✅ File is in excluded list (e.g., `plugin.json`)
- ✅ Check file extension is watched
- ✅ Ensure file is not in ignored directory

**Build fails after edit:**
- ✅ Check docker logs: `docker logs flarewifi-app-1 --tail 50`
- ✅ Look for syntax errors, import issues, sqlc errors
- ✅ Fix errors and save file again (triggers new rebuild)

**Build succeeds but changes not reflected:**
- ✅ Wait for `Listening on port :3000` in logs
- ✅ Hard refresh browser (Cmd+Shift+R / Ctrl+Shift+R)
- ✅ Check if file is in excluded list

**Stuck in rebuild loop:**
- ✅ Check if you're editing generated files (`*_templ.go`, `db/queries/*.go`)
- ✅ Only edit source files (`.templ`, `.sql`), never generated files

**Complete rebuild needed:**
- ✅ Restart container: `docker restart flarewifi-app-1`
- ✅ Check logs for successful startup

### Key Points for AI Agents

1. **Never edit generated files** - Edit source files (`.templ`, `.sql`) only
2. **Don't manually build** - Reflex handles all builds automatically
3. **Wait for rebuild** - Check logs for `Listening on port :3000` before testing
4. **sqlc regeneration** - Editing `.sql` files triggers sqlc to regenerate queries
5. **templ regeneration** - Editing `.templ` files triggers templ to regenerate Go code

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
- **Bootstrap 5.3.3** - core global on **both** admin (`AdminView`) and portal/login (`PortalView`); Bootstrap 3 is removed
- **bootstrap-icons 1.13.1** - core global (admin)
- **htmx v1.9.12**, **Alpine.js v3.15.1** (core global on both admin and portal), **jQuery 3.7.1**
- Modern browsers only; asset bundles target **ES2017** (admin + portal)

## Database

- SQLite with sqlc-generated queries
- Migrations: `YYYYMMDD_NNNN_description.{up,down}.sql`
- Plugins: Create own tables with foreign keys to core, never alter core tables
- MCP SQLite tools available: `db_info`, `list_tables`, `read_records`, `query`, etc.

### ⚠️ CRITICAL: SQLite WAL Mode

**The database uses WAL (Write-Ahead Logging) mode. This affects file handling:**

**WAL Files:**
- `database.sqlite` - Main database file
- `database.sqlite-wal` - Write-ahead log (uncommitted transactions)
- `database.sqlite-shm` - Shared memory file (index for WAL)

**When copying/backing up the database:**
- ✅ Copy ALL THREE files together (`database.sqlite`, `-wal`, `-shm`)
- ✅ Run `PRAGMA wal_checkpoint(TRUNCATE);` before copying to flush WAL to main file
- ❌ NEVER copy just `database.sqlite` - you'll lose uncommitted data

**When working with database outside container:**
```bash
# 1. Checkpoint to flush WAL
docker exec flarewifi-app-1 sqlite3 /opt/flarewifi/app/data/db/database.sqlite "PRAGMA wal_checkpoint(TRUNCATE);"

# 2. Copy all three files
docker cp flarewifi-app-1:/opt/flarewifi/app/data/db/database.sqlite ./
docker cp flarewifi-app-1:/opt/flarewifi/app/data/db/database.sqlite-wal ./
docker cp flarewifi-app-1:/opt/flarewifi/app/data/db/database.sqlite-shm ./

# 3. After modifications, copy all three back
docker cp ./database.sqlite flarewifi-app-1:/opt/flarewifi/app/data/db/
docker cp ./database.sqlite-wal flarewifi-app-1:/opt/flarewifi/app/data/db/
docker cp ./database.sqlite-shm flarewifi-app-1:/opt/flarewifi/app/data/db/
```

**Note:** The Docker container may not have `sqlite3` installed. Work with database files locally after copying them out.

### ⚠️ CRITICAL: Timestamps and Timezones

**SQLite has NO timezone support. Follow these rules strictly:**

**Storage:**
- ✅ Store ALL timestamps in UTC using `CURRENT_TIMESTAMP` or `time.Now().UTC()`
- ✅ Use `DATETIME NOT NULL` (no DEFAULT for application-set timestamps)
- ❌ NEVER use SQLite's `datetime('now', 'localtime')` functions
- ❌ NEVER store local time in the database

**Queries:**
- ✅ Calculate time bounds in Go, pass as params: `time.Now().UTC().AddDate(0, 0, -30)`
- ✅ Use range queries: `WHERE timestamp >= @start_utc AND timestamp <= @end_utc`
- ❌ NEVER use SQLite date functions (`datetime('now')`, `datetime('now', '-30 days')`)

**Display:** Convert UTC → local in Go for display

**Cloud-Sync Architecture:**
- ⚠️ Sessions may only exist in cloud, not locally with stable IDs
- ✅ Use `session_uuid` (VARCHAR) instead of `session_id` (INTEGER) for references
- ❌ NEVER add foreign key constraints to `sessions.id` from plugin tables

## Translations

**ALL user-facing text:** `api.Translate("label", "Username")` or `api.Translate("error", "Invalid input")`

Types: `label`, `error`, `success`, `info`, `warning` | Max 120 chars, natural language (no snake_case)

### Dynamic values (paired params)

- `api.Translate(type, text, pairs...)` accepts **key/value pairs** after the text and interpolates them into the message — **do NOT** build messages with `fmt.Sprintf` or string concatenation.
- **Placeholders use `<% .key %>` delimiters, NOT `{{ .key }}`** — the translation engine (`flaretmpl.GetTextTemplate`) parses the text as a Go `text/template` with custom `Delims("<%", "%>")`. Using `{{ }}` prints the placeholder **literally**.
- Keys are lowercase and match the pair names; values can be any type (string, int, etc.):
  ```go
  // ✅ CORRECT — paired params with <% %> delimiters
  api.Translate("success", "Local version bumped to <% .version %>", "version", newVersion)
  api.Translate("error", "Pull request #<% .number %> is still open", "number", pr.Number)

  // ❌ WRONG — {{ }} delimiters print literally: "... {{.version}}"
  api.Translate("success", "Local version bumped to {{.version}}", "version", newVersion)

  // ❌ WRONG — don't Sprintf around Translate for interpolation
  fmt.Sprintf(api.Translate("success", "Local version bumped to %s"), newVersion)
  ```
- The message text doubles as the translation **filename/key**, so the same text always resolves to the same translation. On the **first** call for a new text the raw text is returned (placeholders not yet interpolated) while the translation file is created; subsequent calls interpolate.

## Frontend

**CSS:** Bootstrap 5 everywhere (admin + portal) - a core-provided global; never vendor your own Bootstrap  
**JavaScript:** Modern browsers only; bundles target ES2017 (admin + portal), so modern JS is fine  
**DOM Manipulation:** Use jQuery (3.7.1) from the core `global.js` - it's already loaded and available  
**Interactivity:** Use htmx and Alpine.js (v3 on both surfaces) - avoid custom JavaScript when possible; never vendor your own copy (see "Frontend libraries" below)  
**Real-time Updates:** Use Server-Sent Events (SSE) for live UI updates, not polling  
**URLs:** `templ.SafeURL(api.Http().Helpers().UrlForRoute("route:name"))`

### Frontend libraries: vendor, don't npm

- **There is no `npm`/`node_modules` on the machine.** Plugin assets are bundled by a Go-native esbuild that only resolves **real files in the source tree** — a bare specifier like `require("jquery")` or `import X from "somelib"` **fails the build** ("could not resolve"). Nothing runs `npm install` (asset bundling is `core/utils/plugins/build-assets.go`).
- **Vendor a library** by dropping its browser/ESM dist into the plugin's `resources/assets/lib/vendor/` as `<lib>-v<version>.<ext>`, then importing it by a **relative path**, e.g. `var $ = require("../../lib/vendor/jquery-v3.7.1.js");`. CSS libs vendor the same way.
- **Shared libs are already vendored by core** in `core/resources/assets/lib/vendor/` and reachable via the esbuild alias `@flare/lib` (configured in `core/utils/plugins/esbuild.go`), e.g. `import Alpine from "@flare/lib/vendor/alpinejs-v3.15.1.js";`. jQuery (3.7.1) and `window.Alpine` are also exposed as globals, so most plugins need no import.
- **⚠️ Bootstrap 5.3.3 is a core-provided global** — auto-loaded on **every** admin AND portal page via core's global bundles (`globals.js` + `global.css` and the portal equivalents), exactly like jQuery/htmx/Alpine. **Never vendor or reference your own Bootstrap** in a plugin/theme (no `vendor/bootstrap-*`, no manifest bootstrap entries, no `@import`/`require` of bootstrap) — you inherit it globally. Bootstrap 3 is removed; it's Bootstrap 5 everywhere. bootstrap-icons 1.13.1 is a core global (admin).
- **⚠️ Alpine.js v3.15.1 runs on BOTH admin and portal — core loads/auto-starts it.** Both asset bundles target **ES2017** (the app supports modern browsers only), so the full Alpine v3 API works on both surfaces (`@click.outside`, `Alpine.store`/`$store`, `Alpine.data`, `x-effect`, adding reactive props after init, etc.). **Never vendor your own Alpine** in a plugin/theme — it double-loads against core's. Use the global `window.Alpine` / `x-*` attributes. (The old portal Alpine v2 / ES5 / IE11 constraints no longer apply.)
- `package.json`/`package-lock.json` are gone from core and plugins (vestigial). Full plugin-author guide: `sdk/mkdocs/docs/api/assets-manifest.md` → "Vendoring frontend libraries (no npm)" / "Bootstrap is a core global" / "Alpine.js: v3 on both surfaces".

### Admin content-area padding

- **Every admin theme's layout wraps `data.Components.PageContent()` in a content-area element that supplies uniform padding on all 4 sides (24px, or the theme's equivalent) — individual page `.templ` files must NOT add their own outer padding/margin** (no `p-4`/`px-4`/`container` wrapper etc. around the page root). This keeps spacing consistent across every admin page without each page author reinventing it.
- That content-area element must use Bootstrap's **`container-fluid`** class (full-bleed), not `container` (which centers content with a max-width) — `container`'s own default gutter is horizontal-only and mismatched with vertical padding, and its centering breaks the full-width layout the sidenav/topbar chrome expects.
- Component-internal padding (e.g. `p-4` inside a bordered `tab-content` box, or a card body) is fine — the rule only applies to the page's **outer** wrapper.
- If a page's content looks cramped against the sidebar/topbar, the fix belongs in the active theme's content-area CSS rule, not in the page template.
- Applies to all four admin themes: `com.flarego.adopisoft-theme` (`.admin-content` in `admin.css`), `com.flarego.devkit` (`main.container-fluid` in `devkit.css`), `com.flarego.flarewifi-theme` (`.layout main` in `layout.css`, already uniform via `padding: 1.5rem`), and the core fallback theme (`.fw-main` in `core/resources/assets/themes/fallback/admin.css`, already uniform via `padding: 20px`).

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
- **Pointer helpers** (`sdk/utils/valconv.go`):
  - Creation: `IntPtr()`, `Int64Ptr()`, `Float64Ptr()`, `BoolPtr()`, `StringPtr()`, `TimePtr()`
  - Deep copy: `CopyIntPtr()`, `CopyInt64Ptr()`, `CopyFloat64Ptr()`, `CopyBoolPtr()`, `CopyStringPtr()`, `CopyTimePtr()`
  - Equality: `IntPtrEqual()`, `Int64PtrEqual()`, `Float64PtrEqual()`, `BoolPtrEqual()`, `StringPtrEqual()`, `TimePtrEqual()`
  - Value extraction: `IntPtrVal()`, `Int64PtrVal()`, `Float64PtrVal()`, `BoolPtrVal()`, `StringPtrVal()`

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
| Vendoring Bootstrap/Alpine | Don't — both are core-provided globals (Bootstrap 5, Alpine v3) on admin + portal |
| 2+ templ edit failures | Stop and consult @frontend |
| Creating custom helper function | Check `sdk/utils/` first (UUID, strings, validators, pagination, etc.) |
| **Copying only database.sqlite** | **Copy ALL 3 WAL files; checkpoint first: `PRAGMA wal_checkpoint(TRUNCATE);`** |
| **Silent error in critical path** | **ALWAYS handle errors; rollback on failure** |
| **Data inconsistency** | **Implement transaction-like behavior with rollback** |
| **Race condition in check-then-act** | **Add DB unique constraints; re-validate before action** |
| **Inclusive time range bug** | **Use `endTime.Add(59s, 999ms)` or `<=` comparison** |
| **Skipping implementation review** | **Review EVERY implementation before completion** |
| **Foreign key to sessions.id from plugin** | **Use session_uuid (VARCHAR) instead; cloud-sync may not have local IDs** |
| **Editing generated files** | **Only edit source files: `.templ`, `.sql`, not `*_templ.go` or `db/queries/*.go`** |
| **Changes not appearing** | **Wait for `Listening on port :3000` in logs, then hard refresh browser** |
| **Build stuck/looping** | **Restart container: `docker restart flarewifi-app-1`** |
| **sqlc errors after SQL edit** | **Check SQL syntax, ensure `@param` names match Go struct fields** |
| **Adding/changing SDK API methods** | **Update `sdk/mkdocs/docs/api/` docs immediately after implementation** |

## UI Testing

Playwright MCP (`http://localhost:3000`): `browser_navigate` → `browser_snapshot` → test → verify both admin/portal

**Screenshots & temp files:** Save all Playwright-generated files (screenshots, console logs, etc.) to `.tmp/playwright/` directory.
