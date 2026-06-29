# AGENTS.md — Flarewifi Plugin Devkit

Guidance for AI coding agents (Claude Code, Cursor, etc.) working **inside this
devkit** to help a developer build a Flarewifi plugin. Read this before editing.

This is the **devkit distribution**, not the Flarewifi source repo. The Flarewifi
core is **closed source** and ships here only as prebuilt binaries
(`bin/<arch>/flare`, `core/plugin.<arch>.so`). You build plugins **against the
public SDK** in `sdk/` — you cannot read or change the core.

See `README.md` for the human-facing quick start. This file is the rulebook.

---

## Golden rules

- **Only edit under `data/plugins/devel/<your.package>/`.** That is the one place
  for your work. Everything else is the shipped environment.
- **Never edit** `core/`, `bin/`, `plugins/installed/`, `sdk/`, `node_modules/`,
  or any generated file (`*_templ.go`, `db/queries/*.go`). They are prebuilt or
  machine-generated; changes are overwritten or break the build.
- **Don't run build tools by hand.** `docker compose up` watches your files and
  auto-rebuilds (templ → sqlc → `flare build-plugins` → restart). Just save and
  wait for the server to come back up.
- **The core is closed source.** If a task seems to need core changes, it almost
  certainly belongs in your plugin instead — work through the SDK.

## Layout (what you can touch)

| Path | What it is | Edit? |
|------|------------|-------|
| `data/plugins/devel/<pkg>/` | **Your plugin(s).** Rebuilt on change. | ✅ yes |
| `data/plugins/devel/com.flarego.devkit-sample/` | Starter sample — copy it as a template. | ✅ copy from |
| `data/config/`, `data/db/` | Runtime config + SQLite DB. | ⚠️ data only, not code |
| `sdk/api/` | Plugin API **interfaces** you call. | 📖 read only |
| `sdk/utils/` | Shared helpers — check here before writing your own. | 📖 read only |
| `sdk/mkdocs/` | Plugin API docs, served at <http://localhost:3002>. | 📖 read only |
| `bin/`, `core/`, `plugins/installed/` | Prebuilt closed-source core + theme. | ❌ never |

## Dev workflow

1. `docker compose up --build` (from the devkit root). Admin UI: <http://localhost:3000>.
2. Create/edit your plugin under `data/plugins/devel/<your.package>/`.
3. Save — the watcher rebuilds and restarts the server automatically.
4. Verify the rebuild: `docker compose logs -f app` and wait for
   `Listening on port :3000`. Then hard-refresh the browser.
5. Inspect data in the SQLite browser at <http://localhost:3001>; read the API
   reference at <http://localhost:3002>.

**Build errors** surface in `docker compose logs -f app` (Go/templ/sqlc errors).
Fix the source and save again to trigger a fresh rebuild — no manual build step.

## Anatomy of a plugin

Minimum structure under `data/plugins/devel/<your.package>/`:

```
main.go        # func Init(api sdkapi.IPluginApi) error  — entry point
plugin.json    # name, package (must match the dir name), version, description
go.mod         # module <your.package>; go 1.21 + toolchain go1.21.13
resources/
  migrations/  # YYYYMMDD_NNNN_name.{up,down}.sql
  queries/     # *.sql → sqlc-generated into db/queries/ (do not edit generated)
  views/       # *.templ → compiled to *_templ.go (do not edit generated)
  assets/      # js, css, images (+ manifest.json)
  translations/
```

- The `package` in `plugin.json` and the `module` in `go.mod` **must equal the
  directory name** (e.g. `com.flarego.devkit-sample`).
- New plugins are picked up on startup; `flare fix-workspace` (run automatically)
  adds them to `go.work`. If a brand-new plugin isn't seen, restart the `app`
  service.
- Pin `go 1.21` / `toolchain go1.21.13` to match the devkit toolchain (see the
  sample's `go.mod`). The prebuilt `flare` compiles every plugin with the
  `dev devkit sqlite` build tags — you don't choose tags.

## Coding rules (carried from the Flarewifi conventions)

**Errors & data integrity**
- Handle **every** error — no `_ = fn()`, no silent log-and-continue in critical
  paths. For multi-step writes, **roll back** on partial failure.
- Add DB constraints (`UNIQUE`, `FOREIGN KEY`) for business rules; re-validate
  before acting, not just in the UI.

**Database (SQLite)**
- IDs are `int64`. Use **named params** (`@param`) in `.sql`; names must match the
  Go fields sqlc generates.
- Plugins create **their own tables** with foreign keys to core tables — never
  `ALTER` core tables.
- **Never** add a foreign key to `sessions.id`. Sessions may live only in the
  cloud with no stable local id — reference `session_uuid` (VARCHAR) instead.
- Store **all timestamps in UTC** (`time.Now().UTC()` / `CURRENT_TIMESTAMP`).
  Compute time bounds in Go and pass as params; don't use SQLite date functions.
  Convert UTC → local **only for display** (`sdkutil.UtcToLocalTime(t)`).

**Frontend**
- Templates are **templ**. Wrap URLs with `templ.SafeURL(...)` and build them via
  `api.Http().Helpers().UrlForRoute("route:name")` — never hardcode paths.
- **JavaScript is ES5 only** (`var`, `function() {}`, no template literals/arrow
  fns). Prefer **htmx** and **Alpine.js**; jQuery is already loaded.
- CSS frameworks differ by surface and must not be mixed: **Bootstrap 3** on the
  captive portal, **Bootstrap 5** on the admin dashboard.

**Translations (all user-facing text)**
- Use `api.Translate(type, text, pairs...)` — `type` ∈ `label|error|success|
  info|warning`. Never hardcode English strings.
- Interpolate with **paired params** and `<% .key %>` delimiters — **not** `{{ }}`
  (which prints literally), and **not** `fmt.Sprintf`:
  ```go
  api.Translate("success", "Saved <% .count %> vouchers", "count", n)
  ```

**Reuse**
- Check `sdk/utils/` before writing helpers (UUIDs, pointers, validators,
  pagination, formatters, retry, etc.). Only add your own if it's truly missing.

## Discovering the API

- Browse the docs site at <http://localhost:3002> (served from `sdk/mkdocs`).
- Read the interfaces in `sdk/api/` directly — e.g. `IPluginApi` is the root:
  `api.Http()`, `api.SqlDB()`, `api.Translate()`, `api.SessionsMgr()`,
  `api.Vouchers()`, `api.Logger()`, `api.Events()`, …
- Plugin logs go through `api.Logger()` (file + admin log viewer in the UI), not
  stdout — `docker compose logs` shows the **server/build** output, not your
  plugin's log lines.

## Verifying a change

- Confirm the rebuild finished (`Listening on port :3000` in the app logs).
- Open <http://localhost:3000>, exercise the plugin's admin page and/or portal.
- Check persisted rows in the SQLite browser (<http://localhost:3001>).

## Don't

- ❌ Edit core, SDK, prebuilt binaries, or generated files.
- ❌ Hardcode text or URLs.
- ❌ Use ES6+ JS, or mix Bootstrap 3 and 5 on one surface.
- ❌ Discard errors or leave multi-step writes without rollback.
- ❌ FK to `sessions.id`, store local-time timestamps, or `ALTER` core tables.
- ❌ Run `go build` / `templ` / `sqlc` manually — let the watcher do it.
