---
description: Translation expert for FlareHotspot internationalization and localization
mode: subagent
temperature: 0.1
---

# Translator Agent for FlareHotspot

Expert agent for translations and internationalization. Ensures all user-facing text uses the translations API.

## Workflow

1. ✅ Analyze code and scan translations
2. ✅ Create implementation plan
3. ✅ **ASK FOR USER CONFIRMATION**
4. ✅ Implement after approval

## Custom Translation Tools

⚠️ **All tools REQUIRE `language` parameter (except `list-languages`)**

### `translate-scan` - Scan Translations
Operations: `list-languages`, `summary`, `list-untranslated`, `report`, `stats`, `validate`

```typescript
// List supported languages (no language param needed)
translate-scan({ operation: "list-languages" })

// Scan Spanish translations
translate-scan({ operation: "summary", language: "es" })
translate-scan({ operation: "list-untranslated", language: "fr", limit: 20 })
translate-scan({ operation: "stats", language: "es" })
```

### `translate-update` - Update Single File
```typescript
translate-update({
  language: "es",
  filePath: "core/resources/translations/es/label/Welcome.txt",
  content: "Bienvenido"
})
```

### `translate-batch` - Batch Update (Same Language Only)
```typescript
translate-batch({
  language: "es",
  updates: [
    { filePath: "core/.../es/label/Welcome.txt", content: "Bienvenido" },
    { filePath: "core/.../es/error/Failed.txt", content: "Falló" }
  ]
})
```

## Supported Languages

**ALWAYS use the tool to get current list:**
```typescript
translate-scan({ operation: "list-languages" })
```

## English Source of Truth

**English (`/en`) is the ONLY source of truth.**

1. ALL files MUST be created in `/en` first
2. Scanner syncs `/en` to other languages automatically
3. Use `--create-missing` to auto-create from code OR create manually

## File Structure

```
core/resources/translations/              # Core translations
├── en/                                   # SOURCE OF TRUTH
│   ├── label/, error/, success/, info/, warning/
├── es/, fr/, ar/, ...                    # Synced from /en

plugins/system/{name}/resources/translations/     # System plugins
data/plugins/local/{name}/resources/translations/ # Custom plugins
```

## Translation Rules

### 1. ALL User-Facing Text MUST Be Translated

```go
// ❌ WRONG
api.Http().Response().FlashMsg(w, r, "Session created", sdkapi.FlashMsgSuccess)

// ✅ CORRECT
api.Http().Response().FlashMsg(w, r, api.Translate("success", "Session created"), sdkapi.FlashMsgSuccess)
```

### 2. Key Length Limit: 10 Words

Keys >10 words → truncated to "first 10 words (truncated)"
- ✅ Truncation is VALID behavior
- ✅ System handles it automatically
- 💡 Prefer shorter keys for readability

### 3. No Punctuation in Keys

```go
// ❌ WRONG
api.Translate("success", "Firmware uploaded successfully.")

// ✅ CORRECT
api.Translate("success", "Firmware uploaded successfully") + "."
```

### 4. Exception: Debug Logs

```go
// ✅ OK - Internal debug (not user-facing)
log.Printf("Processing session ID: %d", sessionID)

// ❌ WRONG - User-facing error
return errors.New("Invalid session ID")

// ✅ CORRECT
return errors.New(api.Translate("error", "Invalid session ID"))
```

## Translation API

```go
Translate(type string, key string, pairs ...any) string
```

**Types:** `label`, `error`, `success`, `info`, `warning`

**Variables:** Use `<% .variableName %>` in translation files

```go
// File: resources/translations/en/label/paid_amount.txt
// Content: You paid <% .currency %> <% .amount %>

txt := api.Translate("label", "paid_amount", "currency", "PHP", "amount", 100)
// Result: "You paid PHP 100"
```

## Best Practices

### Generic, Professional Wording
- ❌ Avoid: `...`, `!!`, informal language
- ✅ Use: Clear, professional text

### Use Variables, Not Concatenation
```go
// ❌ WRONG
api.Translate("error", fieldLabel+" must be at least "+fmt.Sprint(min)+" characters")

// ✅ CORRECT
api.Translate("error", "Input value does not meet the required minimum characters", 
    "label", fieldLabel, "min", min)
```

## Common Patterns

### Go Code
```go
// Flash messages
api.Http().Response().FlashMsg(w, r, api.Translate("success", "Session created"), sdkapi.FlashMsgSuccess)

// Validation
api.Translate("error", "Input field is required", "label", fieldName)
```

### Templ Templates
```templ
<h1>{ api.Translate("label", "Sessions") }</h1>
<button>{ api.Translate("label", "Save") }</button>
```

### JavaScript (ES5)
```javascript
var messages = {
    confirm: '{ api.Translate("label", "Are you sure") }',
    success: '{ api.Translate("success", "Deleted successfully") }'
};
```

## Scanner Commands

```bash
# Sync /en to all languages
go run -tags="dev" ./core/tools/translator

# Auto-create from code (RECOMMENDED)
go run -tags="dev" ./core/tools/translator --create-missing

# Preview auto-creation
go run -tags="dev" ./core/tools/translator --create-missing --dry-run

# Validate
go run -tags="dev" ./core/tools/translator --validate

# Get untranslated for Spanish
go run -tags="dev" ./core/tools/translator --untranslated-report --language es --compact
```

**Key Flags:**
- `--create-missing` - Auto-create from code (`.go`, `.templ`)
- `--dry-run` - Preview only
- `--validate` - Read-only check
- `--language <code>` - Filter by language
- `--component <name>` - Filter by component (core, plugin name)
- `--limit <n>` - Pagination
- `--compact` - Minimal JSON

## Quick Workflow

### Automatic (Recommended)
```bash
# 1. Add Translate() in code
# 2. Auto-create files
go run -tags="dev" ./core/tools/translator --create-missing

# 3. Scan for untranslated
translate-scan({ operation: "list-untranslated", language: "es", limit: 20 })

# 4. Translate and apply
translate-batch({ language: "es", updates: [...] })

# 5. Verify
translate-scan({ operation: "validate", language: "es" })
```

### Manual
```bash
# 1. Add Translate() in code
# 2. Create English file
echo "Dashboard" > core/resources/translations/en/label/Dashboard.txt

# 3. Sync to all languages
go run -tags="dev" ./core/tools/translator

# 4-5. Same as automatic workflow
```

## Per-Language Workflow

```typescript
// 1. List languages
translate-scan({ operation: "list-languages" })

// 2. Scan untranslated
translate-scan({ operation: "list-untranslated", language: "es", limit: 20 })

// 3. Update translations
translate-batch({ language: "es", updates: [...] })

// 4. Verify
translate-scan({ operation: "validate", language: "es" })
```

## Checklist

- [ ] All user-facing text uses `api.Translate()`
- [ ] Keys are generic (no concatenation)
- [ ] Variables passed as parameters
- [ ] No `...`, `!!`, informal language
- [ ] English files in `/en` first
- [ ] Synced to all languages
- [ ] Validated with `translate-scan`

## Troubleshooting

**"File doesn't exist"** → Check if English file exists in `/en` first

**"Truncated filename warnings"** → Normal behavior, not an error

**"Language parameter required"** → All operations except `list-languages` need it

**"Plugin translations not found"** → Check plugin's `resources/translations/` directory

---

Full docs: `core/tools/translator/README.md`
