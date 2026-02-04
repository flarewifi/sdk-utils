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
      const fullPath = path.join(cwd, filePath)

      // Validate that this is a translation file
      if (!filePath.includes('/translations/')) {
        return `❌ ERROR: Invalid translation file path. Must be in /translations/ directory

Provided path: ${filePath}`
      }

      // Extract language from path and validate it matches the language parameter
      const langMatch = filePath.match(/\/translations\/([a-z]{2,3})\//)
      if (!langMatch) {
        return `❌ ERROR: Could not extract language code from file path.

File path must contain /translations/{language}/ pattern.
Provided path: ${filePath}
Expected language: ${language}`
      }
      
      const pathLanguage = langMatch[1]
      if (pathLanguage !== language) {
        return `❌ ERROR: Language mismatch!

Language parameter: ${language}
Language in file path: ${pathLanguage}

The language parameter must match the language in the file path.
Either change the language parameter to "${pathLanguage}" or update the file path to use /translations/${language}/`
      }

      // Check if file exists
      const fileExists = fs.existsSync(fullPath)
      
      if (!fileExists && !createMissing) {
        return `❌ ERROR: File does not exist: ${filePath}

Use createMissing: true to create it.`
      }

      // Create directory if it doesn't exist
      const dir = path.dirname(fullPath)
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true })
      }

      // Read current content if file exists
      let oldContent = ""
      if (fileExists) {
        oldContent = fs.readFileSync(fullPath, "utf-8")
      }
      
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
      
      // Check for snake_case in content
      if (content.match(/\w+_\w+/)) {
        warnings.push("⚠️  WARNING: Translation contains underscores (snake_case) - acceptable in content, but not in keys")
      }
      
      // Write new content
      try {
        fs.writeFileSync(fullPath, content, "utf-8")
      } catch (writeError: any) {
        return `❌ ERROR: Failed to write file: ${filePath}

${writeError.message}

Possible causes:
- Insufficient permissions
- Disk full
- File locked by another process

💡 TIP: Check file permissions and disk space`
      }

      const action = fileExists ? "Updated" : "Created"
      let output = `✅ ${action} ${language.toUpperCase()} translation: ${filePath}

Old content:
${oldContent || "(new file)"}

New content:
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
