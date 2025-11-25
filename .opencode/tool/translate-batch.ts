import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"

interface TranslationUpdate {
  filePath: string
  content: string
}

export default tool({
  description: "Batch update multiple FlareHotspot translation files at once",
  args: {
    updates: tool.schema
      .array(
        tool.schema.object({
          filePath: tool.schema.string().describe("Relative path from project root to translation file"),
          content: tool.schema.string().describe("Translated content"),
        })
      )
      .describe("Array of translation updates to apply"),
    createMissing: tool.schema
      .boolean()
      .optional()
      .default(false)
      .describe("Create files if they don't exist"),
  },
  async execute(args, context) {
    const { updates, createMissing = false } = args
    
    // Get current working directory (should be project root)
    const cwd = process.cwd()
    
    const results: string[] = []
    let successCount = 0
    let errorCount = 0
    
    // Group by language for reporting
    const byLanguage: Record<string, { success: number; error: number }> = {}

    for (const update of updates) {
      try {
        const fullPath = path.join(cwd, update.filePath)

        // Validate translation file path
        if (!update.filePath.includes('/translations/') || !update.filePath.endsWith('.txt')) {
          results.push(`❌ SKIPPED: ${update.filePath} - Not a valid translation file path`)
          errorCount++
          continue
        }
        
        // Extract language from path
        const langMatch = update.filePath.match(/\/translations\/([a-z]{2,3})\//)
        const language = langMatch ? langMatch[1].toUpperCase() : 'UNKNOWN'
        
        // Initialize language counter
        if (!byLanguage[language]) {
          byLanguage[language] = { success: 0, error: 0 }
        }

        // Check if file exists
        const fileExists = fs.existsSync(fullPath)
        
        if (!fileExists && !createMissing) {
          results.push(`❌ SKIPPED [${language}]: ${update.filePath} - File does not exist (use createMissing: true)`)
          byLanguage[language].error++
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
        results.push(`${action} [${language}]: ${update.filePath}`)
        byLanguage[language].success++
        successCount++
      } catch (error) {
        const langMatch = update.filePath.match(/\/translations\/([a-z]{2,3})\//)
        const language = langMatch ? langMatch[1].toUpperCase() : 'UNKNOWN'
        if (!byLanguage[language]) {
          byLanguage[language] = { success: 0, error: 0 }
        }
        results.push(`❌ ERROR [${language}]: ${update.filePath} - ${error}`)
        byLanguage[language].error++
        errorCount++
      }
    }
    
    // Build language summary
    let languageSummary = '\nPer-Language Summary:\n'
    const languages = Object.keys(byLanguage).sort()
    languages.forEach(lang => {
      const stats = byLanguage[lang]
      languageSummary += `  ${lang}: ${stats.success} succeeded, ${stats.error} failed\n`
    })

    const summary = `
Batch Translation Update Complete
==================================
Total files: ${updates.length}
Success: ${successCount}
Errors: ${errorCount}
${languageSummary}
Details:
${results.join('\n')}

💡 TIP: Process different languages in parallel for faster translation updates
`

    return summary
  },
})
