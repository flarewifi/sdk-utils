import { tool } from "@opencode-ai/plugin"

export default tool({
  description: "Get help and examples for FlareHotspot translation system",
  args: {
    topic: tool.schema
      .enum(["overview", "usage", "variables", "types", "file-structure", "best-practices"])
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
- Location: resources/translations/{lang}/{type}/{key}.txt
- API: api.Translate(type, key, ...variables)
- Auto-generation: Missing files are created automatically
- Variables: Use <% .variableName %> syntax in .txt files

Supported Languages: en, es, fr, am, ar, id, prs, ps, ru, sw

Tools available:
- translate-scan: Scan for untranslated content
- translate-update: Update a single translation file
- translate-batch: Update multiple files at once
- translate-help: This help system

Example workflow:
1. Run translate-scan to find untranslated content
2. Use AI to generate translations
3. Apply with translate-batch
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

Translation file (resources/translations/en/label/paid_amount.txt):
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
  core/resources/translations/{lang}/{type}/{key}.txt

Plugin translations:
  data/plugins/local/{plugin}/resources/translations/{lang}/{type}/{key}.txt
  plugins/system/{plugin}/resources/translations/{lang}/{type}/{key}.txt

Example:
  core/resources/translations/
    ├── en/
    │   ├── label/
    │   │   └── Sessions.txt
    │   ├── error/
    │   │   └── Failed to create session.txt
    │   └── success/
    │       └── Session created successfully.txt
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

DON'T:
❌ Hardcode user-facing strings
❌ Concatenate variables in translation keys
❌ Use informal language or excessive punctuation
❌ Skip translating error messages
❌ Create language-specific keys
❌ Translate debug/console logs

Generic key example:
  // Good
  api.Translate("error", "Input value does not meet minimum", "label", "Password", "min", 8)
  
  // Bad - creates different keys for each field
  api.Translate("error", fieldLabel + " must be at least " + min + " characters")
`,
    }

    return helpContent[topic] || helpContent.overview
  },
})
