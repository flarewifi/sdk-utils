# FlareHotspot Translation Tool

Go-based tool for managing FlareHotspot translations. English (`/en`) is the source of truth - all translation files must be created in `/en` first, then the tool syncs them to other languages.

## Quick Start

```bash
# Sync translations from /en to all languages
go run -tags="dev" ./tools/translator

# Validate translations (read-only)
go run -tags="dev" ./tools/translator --validate

# Get untranslated entries (AI-friendly)
go run -tags="dev" ./tools/translator --untranslated-report --compact
```

## Workflow

### Manual Workflow (Traditional)
1. Add translation in code: `api.Translate("label", "Dashboard")`
2. Create English file: `echo "Dashboard" > core/resources/translations/en/label/Dashboard.txt`
3. Run tool to sync: `go run -tags="dev" ./tools/translator`
4. Tool creates files in all languages with English content
5. Translate manually or with AI

### Automatic Workflow (with --create-missing)
1. Add translation in code: `api.Translate("label", "Dashboard")`
2. Run tool with flag: `go run -tags="dev" ./tools/translator --create-missing`
3. Tool scans code, creates files in all languages automatically
4. Translate manually or with AI

**Note:** Files with keys >10 words will have truncated filenames but preserve full text content for translators.

## Common Commands

```bash
# Default: Scan & sync translations
go run -tags="dev" ./tools/translator

# Create missing translation files from code references
go run -tags="dev" ./tools/translator --create-missing

# Preview what --create-missing would do (dry-run)
go run -tags="dev" ./tools/translator --create-missing --dry-run

# Validation (read-only)
go run -tags="dev" ./tools/translator --validate

# Summary (overview)
go run -tags="dev" ./tools/translator --summary --compact

# Untranslated entries (AI-friendly)
go run -tags="dev" ./tools/translator --untranslated-report --compact

# Filter by language
go run -tags="dev" ./tools/translator --untranslated-report --language es --compact

# Filter by component
go run -tags="dev" ./tools/translator --untranslated-report --component core --compact

# Pagination (large datasets)
go run -tags="dev" ./tools/translator --untranslated-report --limit 20 --offset 0 --compact

# Markdown report
go run -tags="dev" ./tools/translator --markdown-report report.md
```

## Key Flags

**Creation & Sync:**
- `--create-missing` - Auto-create translation files from code references
  - Scans `.go` and `.templ` files for `.Translate()` calls
  - Creates files in all supported languages
  - Filename: truncated if >10 words (e.g., "Long key name (truncated).txt")
  - Content: preserves original full text for translator context

**Output Modes:**
- `--untranslated-report` - Untranslated entries (JSON)
- `--summary` - Summary statistics only
- `--compact` - Compact JSON (minimal whitespace)
- `--markdown-report <file>` - Markdown report

**Filtering:**
- `--language <code>` - Filter by language (e.g., es, fr)
- `--component <name>` - Filter by component (core, plugin name)
- `--limit <n>` - Limit entries
- `--offset <n>` - Skip entries (pagination)

**Validation:**
- `--validate` - Read-only validation
- `--strict` - Fail on any untranslated content

**General:**
- `--dry-run` - Preview changes (works with --create-missing)
- `--verbose` - Detailed logging
- `--silent` - Only show issues

## AI Translation Workflows

**Process by language (stay within token limits):**
```bash
# Get summary
go run -tags="dev" ./tools/translator --untranslated-report --language es --summary --compact

# Get batches
go run -tags="dev" ./tools/translator --untranslated-report --language es --limit 50 --compact
go run -tags="dev" ./tools/translator --untranslated-report --language es --offset 50 --limit 50 --compact
```

**Process by component:**
```bash
go run -tags="dev" ./tools/translator --untranslated-report --component core --compact
go run -tags="dev" ./tools/translator --untranslated-report --component paystack --compact
```

**Parallel processing (multiple AI agents):**
```bash
# Agent 1: Spanish core
go run -tags="dev" ./tools/translator --untranslated-report --language es --component core --compact

# Agent 2: French core
go run -tags="dev" ./tools/translator --untranslated-report --language fr --component core --compact
```

## Output Examples

**Untranslated Report:**
```json
[{"key":"Dashboard","type":"label","default_text":"Dashboard","file_path":"core/resources/translations/es/label/Dashboard.txt","language":"es"}]
```

**Summary:**
```json
{"total_keys":158,"total_untranslated":30,"untranslated_by_language":{"es":15,"fr":15}}
```

## Tips

- Use `--compact` to minimize token usage
- Use `--summary` for overview without file lists
- Use `--language` + `--limit` for batches
- Use `--validate` in CI/CD pipelines
- Combine filters: `--language es --component core --limit 20`
