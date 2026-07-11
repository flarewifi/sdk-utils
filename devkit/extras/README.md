# Flarewifi Plugin Devkit

A self-contained environment for building Flarewifi plugins. It runs the
closed-source Flarewifi core (shipped as prebuilt binaries) in Docker and
hot-rebuilds your plugins as you edit them — no Flarewifi source required.

## Requirements

- **Docker** and the **Docker Compose** plugin.
- **Windows:** Docker Desktop with the **WSL2** backend. For fast file-watching
  and correct permissions, keep this devkit folder **inside** the WSL2 filesystem
  (e.g. under your Linux home), not on a Windows drive (`/mnt/c/...`).

The devkit ships native binaries for both `linux/amd64` and `linux/arm64` and
selects the matching set at startup, so it runs as-is on Apple Silicon, Windows
(x86 and ARM via WSL2), and Linux (x86 and ARM) — no configuration needed.

## Quick start

```sh
docker compose up --build
```

Then open the admin dashboard at <http://localhost:3000>. Put your plugin sources
under `data/plugins/devel/<your.package>/` — they are compiled on startup and
rebuilt automatically on change.

## Services

| Service | URL | Purpose |
|---------|-----|---------|
| Admin & captive portal | <http://localhost:3000> | The Flarewifi machine UI where your plugin renders (admin dashboard + captive portal). |
| Admin & portal (HTTPS) | <https://localhost:443> | Same as above over TLS (self-signed cert generated on first boot). |
| Live reload | `ws://localhost:8000` | Drives automatic browser refresh when a plugin rebuild completes. Used by the dev loop; not browsed directly. |
| Plugin API docs | <http://localhost:3002> | The Flarewifi plugin SDK / API documentation site (mkdocs), live-reloaded from `sdk/mkdocs`. |
| SQLite browser | <http://localhost:3001> | Browse and query the devkit's SQLite database (`data/db/database.sqlite`). |

To start only a subset, name the services — e.g. `docker compose up app docs`.

## Your plugin's README

If you add a `README.md` under your plugin's own folder, keep in mind it becomes
the plugin's **store listing description** when published — the copy an operator
reads while deciding whether to buy/install it, not technical documentation for
other developers. Keep it short and plain-language: what the plugin does for
their hotspot business, not how it's built. Leave API details, database/config
internals, and setup walkthroughs out of it.

## Layout

| Path | What it is |
|------|------------|
| `data/plugins/devel/` | **Your plugins** — edit here; rebuilt on change. |
| `sdk/` | The public plugin SDK you compile against (includes `sdk/mkdocs`, the docs served on `:3002`). |
| `bin/`, `core/` | Prebuilt core binaries (`bin/<arch>/…`, `core/plugin.<arch>.so`); resolved per-architecture at startup. |
| `data/db/` | The SQLite database (browsable on `:3001`). |
