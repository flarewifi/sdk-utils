import { tool } from "@opencode-ai/plugin"

export default tool({
  description: "Get help and examples for FlareHotspot translation system",
  args: {
    topic: tool.schema
      .enum(["overview", "usage", "variables", "types", "file-structure", "best-practices", "languages"])
      .optional()
      .default("overview")
      .describe("Help topic to display"),
  },
  async execute(args) {
    const { topic } = args

    const helpContent: Record<string, string> = {
      overview: `
FlareHotspot Translation System Overview
========================================

The translation system uses file-based translations with:
- Location: resources/translations/{lang}/{type}/{key}
- API: api.Translate(type, key, ...variables)
- Source of Truth: English (/en) directory - all other languages sync from it
- Auto-sync: Files in /en are automatically synced to other languages
- Variables: Use <% .variableName %> syntax in translation files

⚠️ CRITICAL: Per-Language Operations
- ALL translation tools require a language parameter
- Exception: translate-scan with operation="list-languages"
- Each tool operation works on ONE language at a time
- Different AI agents can work on different languages in parallel

⚠️ IMPORTANT: English-First Workflow
- All translation files MUST be created in /en first
- The scanner syncs /en files to other languages automatically
- Auto-creation available: Use --create-missing flag to create from code references
- Manual creation option: Create English files manually if preferred

Tools available:
- translate-scan: Scan for untranslated content (REQUIRES language param)
- translate-update: Update a single translation file (REQUIRES language param)
- translate-batch: Batch update files for ONE language (REQUIRES language param)
- translate-help: This help system

Command-line tools:
- make translations-check: Validate translation coverage
- make translation-report: Generate detailed markdown report
- make find-missing LANG=xx: Find missing translations for a language

Example workflow (automatic):
1. List supported languages: translate-scan({ operation: "list-languages" })
2. Add Translate() call in code
3. Auto-create files: go run -tags="dev" ./tools/translator --create-missing
4. Scan for untranslated: translate-scan({ operation: "list-untranslated", language: "es" })
5. Use AI to generate translations
6. Apply with translate-batch({ language: "es", updates: [...] })
7. Validate: translate-scan({ operation: "validate", language: "es" })

Example workflow (manual):
1. List supported languages: translate-scan({ operation: "list-languages" })
2. Add Translate() call in code
3. Manually create English file in /en directory
4. Run Go scanner to sync: go run -tags="dev" ./tools/translator
5. Scan for untranslated: translate-scan({ operation: "list-untranslated", language: "es" })
6. Use AI to generate translations
7. Apply with translate-batch({ language: "es", updates: [...] })
8. Validate: translate-scan({ operation: "validate", language: "es" })
`,

      usage: `
Translation API Usage
=====================

In Go code:
  api.Translate("error", "Failed to save data")
  api.Translate("label", "Username")
  api.Translate("success", "Profile updated successfully")

With variables:
  api.Translate("error", "Minimum length required", "min", 8)
  
In Templ templates:
  <h1>{ api.Translate("label", "Dashboard") }</h1>
  <button>{ api.Translate("label", "Save") }</button>

In JavaScript (ES5):
  // Pass from backend
  var msg = '{ api.Translate("error", "Invalid input") }';
  alert(msg);
`,

      variables: `
Translation Variables
=====================

Template syntax: <% .variableName %>

Go code:
  api.Translate("label", "paid_amount", "currency", "USD", "amount", 100)

Translation file (resources/translations/en/label/paid_amount):
  You paid <% .currency %> <% .amount %>

Result: "You paid USD 100"

Best practices:
- Use generic keys, not concatenated strings
- Pass variables as key-value pairs
- Use descriptive variable names
- Test with different variable values
`,

      types: `
Translation Types
=================

label    - UI labels, buttons, navigation, form fields, page titles
error    - Error messages shown to users
success  - Success confirmation messages
info     - Informational messages
warning  - Warning messages
custom   - Plugin-specific types

Examples:
  api.Translate("label", "Sessions")
  api.Translate("error", "Failed to create session")
  api.Translate("success", "Session created successfully")
  api.Translate("info", "Please check your email")
  api.Translate("warning", "Session will expire soon")
`,

      "file-structure": `
Translation File Structure
==========================

Core translations:
  core/resources/translations/{lang}/{type}/{key}

Plugin translations:
  data/plugins/local/{plugin}/resources/translations/{lang}/{type}/{key}
  plugins/system/{plugin}/resources/translations/{lang}/{type}/{key}

Example:
  core/resources/translations/
    ├── en/
    │   ├── label/
    │   │   └── Sessions
    │   ├── error/
    │   │   └── Failed to create session
    │   └── success/
    │       └── Session created successfully
    ├── es/
    │   ├── label/
    │   ├── error/
    │   └── success/
    └── fr/
        ├── label/
        ├── error/
        └── success/
`,

      "best-practices": `
Translation Best Practices
==========================

DO:
✅ Use generic, reusable keys
✅ Pass variables as parameters, not in keys
✅ Use professional language (no "...", "!!", casual terms)
✅ Translate ALL user-facing text
✅ Use appropriate translation types
✅ Test with different languages
✅ Always specify language parameter in translation tools
✅ Process one language at a time

DON'T:
❌ Hardcode user-facing strings
❌ Concatenate variables in translation keys
❌ Use informal language or excessive punctuation
❌ Skip translating error messages
❌ Create language-specific keys
❌ Translate debug/console logs
❌ Mix different languages in a single batch update

Generic key example:
  // Good
  api.Translate("error", "Input value does not meet minimum", "label", "Password", "min", 8)
  
  // Bad - creates different keys for each field
  api.Translate("error", fieldLabel + " must be at least " + min + " characters")

Tool usage examples:
  // Good - specify language
  translate-scan({ operation: "summary", language: "es" })
  translate-batch({ language: "fr", updates: [...] })
  
  // Bad - missing language parameter
  translate-scan({ operation: "summary" }) // ❌ ERROR
  translate-batch({ updates: [...] }) // ❌ ERROR
`,

       languages: `
Supported Languages
===================

To see the current list of supported languages with full names:
  translate-scan({ operation: "list-languages" })

The list of supported languages is defined in:
  core/utils/config/application.go (SupportedLanguages variable)

Currently supported language codes are dynamically loaded from the config file.
Use translate-scan({ operation: "list-languages" }) to see the current list.

Per-Language Workflow:
======================

1. List languages:
   translate-scan({ operation: "list-languages" })

2. Scan for untranslated (REQUIRES language):
   translate-scan({ operation: "list-untranslated", language: "es" })

3. Update single file (REQUIRES language):
    translate-update({
      language: "es",
      filePath: "core/resources/translations/es/label/Welcome",
      content: "Bienvenido"
    })

4. Batch update (REQUIRES language, all files must match):
    translate-batch({
      language: "es",
      updates: [
        { filePath: "core/.../es/label/Welcome", content: "Bienvenido" },
        { filePath: "core/.../es/error/Failed", content: "Falló" }
      ]
    })

5. Validate (REQUIRES language):
   translate-scan({ operation: "validate", language: "es" })

Parallel Processing:
===================

Different AI agents can work on different languages simultaneously:

Agent 1: Spanish
  translate-scan({ operation: "summary", language: "es" })
  translate-batch({ language: "es", updates: [...] })

Agent 2: French
  translate-scan({ operation: "summary", language: "fr" })
  translate-batch({ language: "fr", updates: [...] })

Agent 3: German
  translate-scan({ operation: "summary", language: "de" })
  translate-batch({ language: "de", updates: [...] })

This approach:
- Prevents language mixing errors
- Enables parallel processing
- Reduces token usage per operation
- Makes delegation clearer
- Improves audit trails
`,
    }

    return helpContent[topic] || helpContent.overview
  },
})
