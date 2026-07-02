# SQLC Generation

`scripts/sqlc-gen.sh` generates sqlc query code for the core and for plugins. It is
a single self-contained script — SQLite is the only engine (Postgres support was
removed), so there is no driver argument and no per-engine config files.

## Usage

```bash
./scripts/sqlc-gen.sh <source_dir>
```

`<source_dir>` is either the core (`./core`) or a plugin
(`./data/plugins/local/<pkg>`).

```bash
# Regenerate core queries
./scripts/sqlc-gen.sh ./core

# Regenerate a plugin's queries
./scripts/sqlc-gen.sh ./data/plugins/local/com.flarego.wifi-hotspot
```

> In dev you normally never run this by hand — reflex runs it on `.sql` changes,
> and `core/utils/plugins/build-queries.go` runs it during plugin builds.

## What it does

sqlc runs in a throwaway temp dir assembled from:

1. **`core/sqlc.yml`** — the sqlc config (`engine: sqlite`).
2. **Core migrations** — always included, so plugin queries can `JOIN` core tables.
3. **The source's own migrations** — nothing extra when generating for core itself.
4. **The source's own queries** — the only queries Go is generated for.

The generated Go is copied back to `<source_dir>/db/queries`. The temp dir is
removed on exit (even on failure).

Every plugin uses the core sqlc config as-is; there is no per-plugin sqlc
configuration. Plugins customize types via sqlc's normal in-query mechanisms
(e.g. `sqlc.embed`, column comments) rather than a separate config file.
