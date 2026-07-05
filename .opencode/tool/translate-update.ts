import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"

export default tool({
  description: "Update a single FlareHotspot translation file for a specific language. Language parameter is REQUIRED.",
  args: {
    language: tool.schema
      .string()
      .describe("REQUIRED: Language code (en, es, fr, am, ar, id, in, prs, ps, ru, sw)"),
    filePath: tool.schema
      .string()
      .describe("Relative path from project root to translation file (e.g., core/resources/translations/es/label/Welcome)"),
    content: tool.schema
      .string()
      .describe("The translated content to write to the file"),
    createMissing: tool.schema
      .boolean()
      .optional()
      .default(false)
      .describe("Create the file if it doesn't exist"),
  },
  async execute(args, context) {
    const { language, filePath, content, createMissing = false } = args

    try {
     // Validate language parameter
     if (!language) {
       const supportedLanguages = await getSupportedLanguages()
       const langCodes = supportedLanguages.map(l => l.code).join(", ")
       return `❌ ERROR: The 'language' parameter is REQUIRED.
 
 Usage: translate-update({ language: "xx", filePath: "...", content: "..." })
 
 Supported languages: ${langCodes}
 
 💡 TIP: Use translate-scan with operation="list-languages" to see all supported languages.`
     }
       
     // Validate language code format
     if (!/^[a-z]{2,3}$/.test(language)) {
       const supportedLanguages = await getSupportedLanguages()
       const langCodes = supportedLanguages.map(l => l.code).join(", ")
       return `❌ ERROR: Invalid language code format: "${language}"
 Language codes must be 2-3 lowercase letters.
 
 Supported languages: ${langCodes}`
     }
       
     // Validate language is supported
     const supportedLanguages = await getSupportedLanguages()
     const supportedCodes = supportedLanguages.map(l => l.code)
     if (!supportedCodes.includes(language)) {
       const langCodes = supportedCodes.join(", ")
       return `❌ ERROR: Unsupported language: "${language}"
 
 Supported languages: ${langCodes}
 
 💡 TIP: Use translate-scan with operation="list-languages" to see all supported languages.`
     }
      
      // Get current working directory (should be project root)
      const cwd = process.cwd()

      // Validate that this is a translation file
      if (!filePath.includes('/translations/')) {
        return `❌ ERROR: Invalid translation file path. Must be in /translations/ directory

Provided path: ${filePath}`
      }

      // Parse filePath: {componentPath}/resources/translations/{lang}/{msgtype}/{key}
      const translationMatch = filePath.match(/^(.+?)\/resources\/translations\/([a-z]{2,3})\/([a-z]+)\/(.+)$/)
      if (!translationMatch) {
        return `❌ ERROR: Invalid translation file path format.

Expected: {component}/resources/translations/{lang}/{type}/{key}
Example: core/resources/translations/es/label/Welcome
Provided: ${filePath}`
      }

      const compPath = translationMatch[1]
      const lang = translationMatch[2]
      const msgtype = translationMatch[3]
      const key = translationMatch[4]

      // Validate language matches parameter
      if (lang !== language) {
        return `❌ ERROR: Language mismatch!

Language parameter: ${language}
Language in file path: ${lang}

The language parameter must match the language in the file path.`
      }

      const jsonPath = path.join(cwd, `${compPath}/resources/translations/${lang}.json`)
      const jsonDir = path.dirname(jsonPath)

      // Validate content encoding (check for common issues)
      if (content.includes('\0')) {
        return `❌ ERROR: Content contains null bytes (invalid UTF-8)

Translation content must be valid UTF-8 text.`
      }

      // Warn about potential issues
      const warnings: string[] = []
      if (content.length === 0) {
        warnings.push("⚠️  WARNING: Empty translation content")
      }
      if (content !== content.trim()) {
        warnings.push("⚠️  WARNING: Translation has leading/trailing whitespace (will be preserved)")
      }
      if (content.includes('  ')) {
        warnings.push("⚠️  WARNING: Translation contains double spaces")
      }
      if (content.match(/\w+_\w+/)) {
        warnings.push("⚠️  WARNING: Translation contains underscores (snake_case) - acceptable in content, but not in keys")
      }

      // Read existing catalog or create empty structure
      let catalog: Record<string, Record<string, string>> = {
        error: {}, info: {}, label: {}, success: {}, type: {}, warning: {},
      }
      const catalogExists = fs.existsSync(jsonPath)

      if (catalogExists) {
        try {
          const existing = JSON.parse(fs.readFileSync(jsonPath, 'utf-8'))
          catalog = { ...catalog, ...existing }
        } catch (e: any) {
          return `❌ ERROR: Could not parse existing catalog ${lang}.json: ${e.message}`
        }
      }

      if (!catalogExists && !createMissing) {
        return `❌ ERROR: Translation catalog does not exist: ${compPath}/resources/translations/${lang}.json

Use createMissing: true to create it.`
      }

      // Ensure msgtype section exists
      if (!catalog[msgtype]) {
        catalog[msgtype] = {}
      }

      // Read old value if it exists
      const oldContent = catalog[msgtype][key] || ""

      // Set the translation
      catalog[msgtype][key] = content

      // Write catalog back
      try {
        if (!fs.existsSync(jsonDir)) {
          fs.mkdirSync(jsonDir, { recursive: true })
        }

        // Sort keys for deterministic output
        const sorted: Record<string, Record<string, string>> = {}
        for (const [section, entries] of Object.entries(catalog)) {
          sorted[section] = {}
          for (const k of Object.keys(entries).sort()) {
            sorted[section][k] = entries[k]
          }
        }

        fs.writeFileSync(jsonPath, JSON.stringify(sorted, null, 2) + '\n', 'utf-8')
      } catch (writeError: any) {
        return `❌ ERROR: Failed to write ${compPath}/resources/translations/${lang}.json

${writeError.message}

Possible causes:
- Insufficient permissions
- Disk full
- File locked by another process`
      }

      const action = catalogExists ? "Updated" : "Created"
      let output = `✅ ${action} ${language.toUpperCase()} translation: ${compPath}/resources/translations/${lang}.json [${msgtype}.${key}]

Old value:
${oldContent || "(none)"}

New value:
${content}
`
      
      // Add warnings if any
      if (warnings.length > 0) {
        output += `\n${warnings.join('\n')}\n`
      }
      
      output += `\n💡 TIP: Use translate-batch to update multiple ${language.toUpperCase()} files at once`
      
       return output
     } catch (error) {
       return `❌ ERROR updating translation: ${error}`
     }
   },
})

// Helper function to read supported languages from Go config
async function getSupportedLanguages(): Promise<Array<{ code: string; name: string }>> {
  try {
    const cwd = process.cwd()
    const configPath = path.join(cwd, "core/utils/config/application.go")

    if (!fs.existsSync(configPath)) {
      throw new Error("Could not find core/utils/config/application.go")
    }

    const content = fs.readFileSync(configPath, "utf-8")

    // Parse SupportedLanguages array
    const langPattern = /var SupportedLanguages = \[\]sdkapi\.SupportedLanguage\{([\s\S]*?)\n\}/
    const match = content.match(langPattern)

    if (!match) {
      throw new Error("Could not parse SupportedLanguages from core/utils/config/application.go")
    }

    // Extract language entries
    const languagesBlock = match[1]
    const entryPattern = /\{Code:\s*"([^"]+)",\s*Name:\s*"([^"]+)"\}/g
    const languages: Array<{ code: string; name: string }> = []

    let entryMatch
    while ((entryMatch = entryPattern.exec(languagesBlock)) !== null) {
      languages.push({
        code: entryMatch[1],
        name: entryMatch[2]
      })
    }

    if (languages.length === 0) {
      throw new Error("No languages found in core/utils/config/application.go")
    }

    return languages
  } catch (error) {
    throw new Error(`Failed to read supported languages: ${error}`)
  }
}
