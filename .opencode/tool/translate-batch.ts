import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"

interface TranslationUpdate {
  filePath: string
  content: string
}

export default tool({
  description: "Batch update multiple FlareHotspot translation files for a SINGLE language. All files must be for the same language.",
  args: {
    language: tool.schema
      .string()
      .describe("REQUIRED: Language code for ALL updates (en, es, fr, am, ar, id, in, prs, ps, ru, sw)"),
    updates: tool.schema
      .array(
        tool.schema.object({
          filePath: tool.schema.string().describe("Relative path from project root to translation file"),
          content: tool.schema.string().describe("Translated content"),
        })
      )
      .describe("Array of translation updates to apply (all must be for the specified language)"),
    createMissing: tool.schema
      .boolean()
      .optional()
      .default(false)
      .describe("Create files if they don't exist"),
  },
  async execute(args, context) {
    const { language, updates, createMissing = false } = args
    
    // Validate language parameter
    if (!language) {
      return `❌ ERROR: The 'language' parameter is REQUIRED.

Usage: translate-batch({ 
  language: "xx", 
  updates: [{ filePath: "...", content: "..." }]
})

Supported languages: en, es, fr, am, ar, id, in, prs, ps, ru, sw

💡 TIP: Use translate-scan with operation="list-languages" to see all supported languages.`
    }
    
    // Validate language code format
    if (!/^[a-z]{2,3}$/.test(language)) {
      return `❌ ERROR: Invalid language code format: "${language}"
Language codes must be 2-3 lowercase letters.

Supported languages: en, es, fr, am, ar, id, in, prs, ps, ru, sw`
    }
    
    // Validate language is supported
    const supportedLanguages = ["en", "es", "fr", "am", "ar", "id", "in", "prs", "ps", "ru", "sw"]
    if (!supportedLanguages.includes(language)) {
      return `❌ ERROR: Unsupported language: "${language}"

Supported languages: ${supportedLanguages.join(", ")}

💡 TIP: Use translate-scan with operation="list-languages" to see all supported languages.`
    }
    
    // Get current working directory (should be project root)
    const cwd = process.cwd()
    
    const results: string[] = []
    let successCount = 0
    let errorCount = 0
    const languageUpper = language.toUpperCase()

    // First pass: validate all updates are for the correct language
    const validationErrors: string[] = []
    for (const update of updates) {
      // Validate translation file path
      if (!update.filePath.includes('/translations/') || !update.filePath.endsWith('.txt')) {
        validationErrors.push(`${update.filePath} - Not a valid translation file path`)
        continue
      }
      
      // Extract language from path
      const langMatch = update.filePath.match(/\/translations\/([a-z]{2,3})\//)
      if (!langMatch) {
        validationErrors.push(`${update.filePath} - Could not extract language from path`)
        continue
      }
      
      const pathLanguage = langMatch[1]
      if (pathLanguage !== language) {
        validationErrors.push(`${update.filePath} - Language mismatch (expected ${language}, found ${pathLanguage})`)
      }
    }
    
    // If there are validation errors, stop and report them
    if (validationErrors.length > 0) {
      return `❌ VALIDATION FAILED: Cannot process batch update

Language parameter: ${languageUpper}
Validation errors found: ${validationErrors.length}

${validationErrors.map(err => `  ❌ ${err}`).join('\n')}

💡 FIX: All files in a batch update must be for the SAME language (${language}).
💡 TIP: Use separate translate-batch calls for different languages.
💡 TIP: Different AI agents can work on different languages in parallel.`
    }

    // Second pass: apply updates
    for (const update of updates) {
      try {
        const fullPath = path.join(cwd, update.filePath)

        // Check if file exists
        const fileExists = fs.existsSync(fullPath)
        
        if (!fileExists && !createMissing) {
          results.push(`❌ SKIPPED: ${update.filePath} - File does not exist (use createMissing: true)`)
          errorCount++
          continue
        }

        // Create directory if needed
        const dir = path.dirname(fullPath)
        if (!fs.existsSync(dir)) {
          fs.mkdirSync(dir, { recursive: true })
        }

        // Write content
        fs.writeFileSync(fullPath, update.content, "utf-8")

        const action = fileExists ? "✅ UPDATED" : "✅ CREATED"
        results.push(`${action}: ${update.filePath}`)
        successCount++
      } catch (error) {
        results.push(`❌ ERROR: ${update.filePath} - ${error}`)
        errorCount++
      }
    }

    const summary = `
✅ Batch ${languageUpper} Translation Update Complete
${'='.repeat(50)}

Language: ${languageUpper}
Total files: ${updates.length}
Success: ${successCount}
Errors: ${errorCount}

Details:
${results.join('\n')}

💡 TIP: Different AI agents can work on different languages in parallel
💡 TIP: Use translate-scan with language="${language}" to verify results
`

    return summary
  },
})
