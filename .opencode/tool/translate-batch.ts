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
       const supportedLanguages = await getSupportedLanguages()
       const langCodes = supportedLanguages.map(l => l.code).join(", ")
       return `❌ ERROR: The 'language' parameter is REQUIRED.
 
 Usage: translate-batch({ 
   language: "xx", 
   updates: [{ filePath: "...", content: "..." }]
 })
 
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
    
    const results: string[] = []
    let successCount = 0
    let errorCount = 0
    const languageUpper = language.toUpperCase()

    // First pass: validate all updates are for the correct language
    const validationErrors: string[] = []
    for (const update of updates) {
      // Validate translation file path
      if (!update.filePath.includes('/translations/')) {
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

    // Process updates and track results
    for (let i = 0; i < updates.length; i++) {
      const update = updates[i]

      try {
        const fullPath = path.join(cwd, update.filePath)

        // Check if file exists
        const fileExists = fs.existsSync(fullPath)

        if (!fileExists && !createMissing) {
          results.push(`❌ SKIPPED: ${update.filePath} - File does not exist (use createMissing: true)`)
          errorCount++
          continue
        }

        // Validate content
        if (update.content.includes('\0')) {
          results.push(`❌ SKIPPED: ${update.filePath} - Invalid UTF-8 (contains null bytes)`)
          errorCount++
          continue
        }

        // Create directory if needed
        const dir = path.dirname(fullPath)
        if (!fs.existsSync(dir)) {
          try {
            fs.mkdirSync(dir, { recursive: true })
          } catch (mkdirError: any) {
            results.push(`❌ ERROR: ${update.filePath} - Failed to create directory: ${mkdirError.message}`)
            errorCount++
            continue
          }
        }

        // Write content with retry logic
        let retries = 3
        let written = false
        let lastError: any = null

        while (retries > 0 && !written) {
          try {
            fs.writeFileSync(fullPath, update.content, "utf-8")
            written = true
          } catch (writeError: any) {
            lastError = writeError
            retries--
            if (retries > 0) {
              // Wait a bit before retrying (file might be locked)
              await new Promise(resolve => setTimeout(resolve, 100))
            }
          }
        }

        if (!written) {
          results.push(`❌ ERROR: ${update.filePath} - Failed after 3 retries: ${lastError?.message || 'Unknown error'}`)
          errorCount++
          continue
        }

        const action = fileExists ? "✅ UPDATED" : "✅ CREATED"
        results.push(`${action}: ${update.filePath}`)
        successCount++
      } catch (error: any) {
        results.push(`❌ ERROR: ${update.filePath} - ${error.message || error}`)
        errorCount++
      }
    }

    // Prepare summary with status indicator
    const statusIcon = errorCount === 0 ? '✅' : errorCount < updates.length ? '⚠️' : '❌'
    const statusText = errorCount === 0 ? 'Complete' : errorCount < updates.length ? 'Partial Success' : 'Failed'

    const summary = `
${statusIcon} Batch ${languageUpper} Translation Update ${statusText}
${'='.repeat(50)}

Language: ${languageUpper}
Total files: ${updates.length}
Success: ${successCount}
Errors: ${errorCount}

Details:
${results.join('\n')}

${errorCount > 0 ? `\n⚠️  Some updates failed. Review errors above and retry failed items.\n` : ''}
💡 TIP: Different AI agents can work on different languages in parallel
💡 TIP: Use translate-scan with language="${language}" to verify results
${errorCount > 0 ? `💡 TIP: You can retry just the failed files by running translate-batch again with only those files\n` : ''}
`

     return summary
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
