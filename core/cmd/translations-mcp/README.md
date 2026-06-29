# translations-mcp

An [MCP](https://modelcontextprotocol.io) server that lets any AI tool **summarize,
query, and work on** the Flarewifi translation catalogs
(`resources/translations/<lang>.json`).

Translations are per-language JSON catalogs keyed by the English source text:

```json
// es.json
{ "error": { "Device is blocked": "El dispositivo está bloqueado" }, "label": { } }
```

`en.json` is the registry (value == key). A key absent from a `<lang>.json` falls
back to the English source at runtime, so non-English catalogs stay sparse.

## Build

```sh
go build -tags dev -o .tmp/translations-mcp ./core/cmd/translations-mcp
```

## Run

```sh
translations-mcp              # MCP server: JSON-RPC 2.0, newline-delimited, over stdio
translations-mcp -root .      # working directory to scan (default ".")
translations-mcp -check       # CI gate: exit 1 if any code key is missing from en.json
translations-mcp -check -min 80   # also require >= 80% coverage per language
```

`make translate-check` runs the CI gate.

## Register with an AI tool (example: Claude Code)

```json
{
  "mcpServers": {
    "flarewifi-translations": {
      "command": "/abs/path/to/translations-mcp",
      "args": ["-root", "/abs/path/to/core/flarewifi"]
    }
  }
}
```

## Tools

| Tool | Purpose |
|------|---------|
| `list_components` | List components (core + plugins), their languages and en-key counts |
| `summarize` | Per-component, per-language coverage (translated / untranslated / percent) |
| `list_keys` | Browse entries (msgtype/key/value/translated) with search + paging |
| `get_translation` | One key's value, English source, and translated flag |
| `find_untranslated` | The work queue: English keys not yet translated in a language |
| `set_translation` | Set one translation (key must exist in en.json; en is read-only) |
| `set_translations` | Bulk-set many translations for one component+language |
| `sync` | Scan Go/templ source for `Translate("type","text")` and add missing keys to en.json |
| `check` | Report code keys missing from en.json + per-language coverage (read-only) |

### Typical workflows

- **Translate a language**: `find_untranslated` → `set_translations` (bulk) → `summarize`.
- **After adding new UI strings in code**: `sync` (registers them in en.json), then translate.
- **CI**: `check` / `translations-mcp -check` fails the build if `sync` wasn't run.

Components are discovered from `core` and each plugin source under
`data/plugins/devel/*` and `data/plugins/local/*` (build-output copies under
`plugins/installed/*` are intentionally excluded — edits belong in the sources).
