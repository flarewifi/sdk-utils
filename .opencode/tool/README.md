# FlareHotspot Translation Tools for OpenCode

Custom OpenCode tools for managing FlareHotspot translations.

## Available Tools

### 1. `translate-scan`
Scans the codebase for translation usage and untranslated content.

**Arguments:**
- `operation` (enum): `summary` | `list-untranslated` | `report` | `stats`
- `type` (string, optional): Filter by translation type (`label`, `error`, `success`, `info`, `warning`)
- `language` (string, optional): Filter by language code (`en`, `es`, `fr`, etc.)

**Examples:**
```
User: "Scan for untranslated content"
OpenCode: [Calls translate-scan with operation="summary"]

User: "Show me all untranslated Spanish labels"
OpenCode: [Calls translate-scan with operation="list-untranslated", type="label", language="es"]

User: "Generate a translation report for AI tools"
OpenCode: [Calls translate-scan with operation="report"]
```

### 2. `translate-update`
Updates a single translation file with language-specific content.

**Arguments:**
- `filePath` (string): Relative path from project root to translation file
- `content` (string): The translated content to write
- `createMissing` (boolean, optional): Create file if it doesn't exist (default: false)

**Examples:**
```
User: "Update the Spanish welcome message to 'Bienvenido a FlareHotspot'"
OpenCode: [Calls translate-update with filePath="core/resources/translations/es/label/Welcome.txt", content="Bienvenido a FlareHotspot"]
```

### 3. `translate-batch`
Batch updates multiple translation files at once.

**Arguments:**
- `updates` (array): Array of `{filePath: string, content: string}` objects
- `createMissing` (boolean, optional): Create files if they don't exist (default: false)

**Examples:**
```
User: "Translate these 10 error messages to French"
OpenCode: [Generates French translations, calls translate-batch with array of updates]
```

### 4. `translate-help`
Provides help and examples for the FlareHotspot translation system.

**Arguments:**
- `topic` (enum, optional): `overview` | `usage` | `variables` | `types` | `file-structure` | `best-practices`

**Examples:**
```
User: "How do I use translation variables?"
OpenCode: [Calls translate-help with topic="variables"]
```

## Workflow Examples

### Single Language Workflow

1. **Scan for untranslated content:**
   ```
   User: "Check for untranslated Spanish content"
   OpenCode uses translate-scan with language="es"
   ```

2. **Generate translations:**
   ```
   User: "Translate the untranslated Spanish labels"
   OpenCode generates Spanish translations using AI
   ```

3. **Apply translations:**
   ```
   OpenCode uses translate-batch to update all Spanish files
   ```

4. **Verify:**
   ```
   User: "Scan again to verify Spanish translations"
   OpenCode uses translate-scan with language="es" to confirm
   ```

### Parallel Multi-Language Workflow

**Perfect for dividing work across AI agents!**

1. **Agent 1** - Spanish translations:
   ```
   "Scan for untranslated Spanish content"
   "Translate all untranslated Spanish files"
   ```

2. **Agent 2** - French translations (running in parallel):
   ```
   "Scan for untranslated French content"
   "Translate all untranslated French files"
   ```

3. **Agent 3** - Arabic translations (running in parallel):
   ```
   "Scan for untranslated Arabic content"
   "Translate all untranslated Arabic files"
   ```

Each agent works independently on their assigned language, dramatically speeding up translation work!

## Translation File Structure

```
resources/translations/
├── en/                     # English (default)
│   ├── label/
│   ├── error/
│   ├── success/
│   ├── info/
│   └── warning/
├── es/                     # Spanish
├── fr/                     # French
├── am/                     # Amharic
├── ar/                     # Arabic
├── id/                     # Indonesian
├── prs/                    # Persian
├── ps/                     # Pashto
├── ru/                     # Russian
└── sw/                     # Swahili
```

## Translation API Reference

### In Go Code
```go
api.Translate("error", "Failed to save data")
api.Translate("label", "Username")
api.Translate("success", "Profile updated successfully")

// With variables
api.Translate("error", "Minimum length required", "min", 8)
```

### In Templ Templates
```templ
<h1>{ api.Translate("label", "Dashboard") }</h1>
<button>{ api.Translate("label", "Save") }</button>
```

### Variables in Translation Files
Use `<% .variableName %>` syntax in `.txt` files:

```
File: resources/translations/en/label/paid_amount.txt
Content: You paid <% .currency %> <% .amount %>

Code: api.Translate("label", "paid_amount", "currency", "USD", "amount", 100)
Result: "You paid USD 100"
```

## Best Practices

✅ **DO:**
- Use generic, reusable keys
- Pass variables as parameters, not in keys
- Use professional language
- Translate ALL user-facing text
- Use appropriate translation types

❌ **DON'T:**
- Hardcode user-facing strings
- Concatenate variables in translation keys
- Use informal language or excessive punctuation
- Skip translating error messages
- Translate debug/console logs

## Supported Languages

- `en` - English
- `es` - Spanish
- `fr` - French
- `am` - Amharic
- `ar` - Arabic
- `id` - Indonesian
- `prs` - Persian
- `ps` - Pashto
- `ru` - Russian
- `sw` - Swahili

## Technical Details

### Portable Commands
All tools use portable commands that work from any project directory:
- Uses `process.cwd()` to get the current working directory
- Uses `$(pwd)` in shell commands for dynamic path resolution
- Accepts relative paths from project root

### Auto-Generation
When a translation key is used but the file doesn't exist, the system automatically:
1. Creates the file with the key as default content
2. Creates files for all supported languages
3. Requires manual translation to proper language-specific text

### Scanning Tool
The `translate-scan` tool wraps the Go scanner at `tools/cmd/scan-translations/main.go`:
- Scans all `.go` and `.templ` files for `Translate()` calls
- Validates translation keys
- Manages translation files across languages
- Identifies untranslated content (where file content equals the key)

## Per-Language Workflows

### Why Per-Language Processing?

The tools are specifically designed to support **per-language workflows**, allowing multiple AI agents to work on different languages in parallel. This provides:

- **Faster translation**: Divide and conquer across languages
- **Better context**: Each agent focuses on one language's nuances
- **Easier management**: Track progress per language
- **Scalability**: Add more languages without bottlenecks

### How to Use Per-Language Filtering

All tools support the `language` parameter:

```
# Scan only Spanish files
translate-scan(operation="list-untranslated", language="es")

# Get Spanish-only report
translate-scan(operation="report", language="es")

# Update only French translations
translate-batch(updates=[...french files...])
```

### Parallel Agent Example

Assign different languages to different agents:

**Agent 1 (Spanish):**
```
1. Scan: translate-scan(language="es")
2. Translate: Generate Spanish translations
3. Apply: translate-batch(updates=[...spanish...])
4. Verify: translate-scan(language="es")
```

**Agent 2 (French)** - Running simultaneously:
```
1. Scan: translate-scan(language="fr")
2. Translate: Generate French translations
3. Apply: translate-batch(updates=[...french...])
4. Verify: translate-scan(language="fr")
```

**Result**: Both languages translated in parallel!

## Integration with OpenCode

OpenCode automatically loads these tools on startup. The AI can use them naturally in conversation:

```
User: "Are there any untranslated error messages in French?"
AI: [Uses translate-scan with language="fr"]
AI: "Yes, I found 15 untranslated French error messages. Would you like me to translate them?"

User: "Yes, please"
AI: [Generates French translations using context and AI]
AI: [Uses translate-batch to apply all French translations]
AI: "Done! All 15 French error messages have been translated."
```

## Troubleshooting

### Tool not found
If OpenCode doesn't recognize the tools, restart OpenCode to reload the tool definitions.

### Permission errors
Ensure the `.opencode/tool/` directory and files have proper read/write permissions.

### Go scanner fails
Ensure you're in the FlareHotspot project root directory and that Go is properly installed.

### Translation files not created
Check that the file path follows the pattern: `{module}/resources/translations/{lang}/{type}/{key}.txt`

## Contributing

To add new translation-related tools:

1. Create a new `.ts` file in `.opencode/tool/`
2. Use the `tool()` helper from `@opencode-ai/plugin`
3. Follow the existing tool patterns
4. Update this README

## Related Documentation

- [Custom Tools Documentation](https://opencode.ai/docs/custom-tools/)
- [FlareHotspot Translation Agent](./.opencode/agent/translations.md)
- [FlareHotspot AGENTS.md](../AGENTS.md)
