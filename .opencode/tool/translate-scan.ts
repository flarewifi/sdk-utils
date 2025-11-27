import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"

export default tool({
  description: "Scan FlareHotspot codebase for translation usage and untranslated content. All operations require a language parameter except 'list-languages'.",
  args: {
    operation: tool.schema
      .enum(["list-languages", "summary", "list-untranslated", "report", "stats", "validate"])
      .describe("Operation: list-languages (show supported languages), summary (overview), list-untranslated (files needing translation), report (JSON for AI), stats (detailed statistics), validate (check coverage)"),
    language: tool.schema
      .string()
      .optional()
      .describe("REQUIRED for all operations except 'list-languages'. Language code (en, es, fr, am, ar, id, in, prs, ps, ru, sw)"),
    type: tool.schema
      .string()
      .optional()
      .describe("Filter by translation type (label, error, success, info, warning)"),
    component: tool.schema
      .string()
      .optional()
      .describe("Filter by component name (core, plugin name like 'paystack' or 'wifi-hotspot')"),
    limit: tool.schema
      .number()
      .optional()
      .describe("Limit output to N entries (for pagination, reduces token usage)"),
    offset: tool.schema
      .number()
      .optional()
      .describe("Skip first N entries (for pagination)"),
  },
  async execute(args, context) {
    const { operation = "summary", language, type, component, limit, offset } = args
    
    // Handle list-languages operation (doesn't require language parameter)
    if (operation === "list-languages") {
      return await listSupportedLanguages()
    }
    
    // All other operations REQUIRE language parameter
    if (!language) {
      return `❌ ERROR: The 'language' parameter is REQUIRED for operation '${operation}'.

Usage: translate-scan({ operation: "${operation}", language: "xx" })

Supported languages: en, es, fr, am, ar, id, in, prs, ps, ru, sw

💡 TIP: Use operation="list-languages" to see all supported languages with their names.`
    }
    
    // Validate language code format
    if (!/^[a-z]{2,3}$/.test(language)) {
      return `❌ ERROR: Invalid language code format: "${language}"
Language codes must be 2-3 lowercase letters.

Supported languages: en, es, fr, am, ar, id, in, prs, ps, ru, sw

💡 TIP: Use operation="list-languages" to see all supported languages.`
    }
    
    // Validate language is supported
    const supportedLanguages = ["en", "es", "fr", "am", "ar", "id", "in", "prs", "ps", "ru", "sw"]
    if (!supportedLanguages.includes(language)) {
      return `❌ ERROR: Unsupported language: "${language}"

Supported languages: ${supportedLanguages.join(", ")}

💡 TIP: Use operation="list-languages" to see all supported languages with their full names.`
    }

    try {
      // Use portable command that works from any directory
      let flags = ""
      
      switch (operation) {
        case "list-untranslated":
          flags = " --list-untranslated"
          break
        case "report":
          flags = " --untranslated-report --compact"
          break
        case "stats":
          flags = " --json --compact"
          break
        case "validate":
          flags = " --validate"
          break
        case "summary":
        default:
          flags = " --summary --compact"
          break
      }

      // Language is always specified for non-list-languages operations
      flags += ` --language=${language}`
      
      if (component) {
        flags += ` --component=${component}`
      }
      
      if (limit !== undefined && limit > 0) {
        flags += ` --limit=${limit}`
      }
      
      if (offset !== undefined && offset > 0) {
        flags += ` --offset=${offset}`
      }
      
      // Use the renamed Go tool with proper tags
      const command = `go run -tags="dev" $(pwd)/tools/translator${flags}`
      const result = await Bun.$`sh -c ${command}`.text()

      // Parse and format the output based on operation
      if (operation === "summary") {
        // Parse summary JSON and format nicely
        try {
          const jsonData = JSON.parse(result)
          
          let output = "📊 Translation Summary\n\n"
          output += `Total Keys: ${jsonData.total_keys || 0}\n`
          output += `Total Untranslated: ${jsonData.total_untranslated || 0}\n`
          
          if (jsonData.untranslated_by_language) {
            output += "\nUntranslated by Language:\n"
            Object.entries(jsonData.untranslated_by_language).forEach(([lang, count]) => {
              output += `  ${lang.toUpperCase()}: ${count}\n`
            })
          }
          
          if (jsonData.validation) {
            output += `\nValidation:\n`
            output += `  Components: ${jsonData.validation.total_components || 0}\n`
            output += `  Total Issues: ${jsonData.validation.total_issues || 0}\n`
            output += `  Critical Issues: ${jsonData.validation.critical_issues || 0}\n`
            
            if (jsonData.validation.components && jsonData.validation.components.length > 0) {
              output += `\nComponents:\n`
              jsonData.validation.components.forEach((comp: any) => {
                output += `  ${comp.name}: ${comp.english_count} files\n`
                if (comp.status_counts) {
                  Object.entries(comp.status_counts).forEach(([status, count]) => {
                    output += `    ${status}: ${count}\n`
                  })
                }
              })
            }
          }
          
          // Add pagination info if applicable
          if (limit || offset) {
            output += `\n📄 Pagination: offset=${offset || 0}, limit=${limit || 'all'}\n`
          }
          
          // Add filter info
          const filters = []
          filters.push(`language=${language}`)
          if (component) filters.push(`component=${component}`)
          output += `🔍 Filters: ${filters.join(', ')}\n`
          
          output += "\n💡 TIP: Use 'report' operation for detailed JSON output"
          output += "\n💡 TIP: Use 'limit' and 'offset' for pagination"
          output += "\n💡 TIP: Use 'component' to filter by core or plugin"
          output += "\n💡 TIP: Different AI agents can work on different languages in parallel"
          
          return output
        } catch (e) {
          return result
        }
      } else if (operation === "report" || operation === "stats") {
        // Try to parse as JSON
        try {
          const jsonData = JSON.parse(result)
          
          // Apply type filter if specified (language/component filtering is done by Go tool)
          let filtered = jsonData
          if (Array.isArray(jsonData) && type) {
            filtered = jsonData.filter((entry: any) => entry.type === type)
          }
          
          // Add metadata about filters and pagination
          let output = ""
          if (Array.isArray(filtered)) {
            output += `Found ${filtered.length} entries`
            if (limit || offset) {
              output += ` (offset=${offset || 0}, limit=${limit || 'all'})`
            }
            output += "\n\n"
          }
          
          output += JSON.stringify(filtered, null, 2)
          
          // Add helpful tips
          if (Array.isArray(filtered) && filtered.length > 0) {
            output += `\n\n💡 Working on: ${language.toUpperCase()} translations`
            output += "\n💡 TIP: Use 'limit' to reduce output size"
            output += "\n💡 TIP: Use 'offset' to get next batch"
            if (!component) {
              output += "\n💡 TIP: Use 'component' to filter by core or plugin"
            }
            output += "\n💡 TIP: Process different languages in parallel with separate agents"
          }
          
          return output
        } catch (e) {
          return result
        }
      } else if (operation === "list-untranslated") {
        const lines = result.split('\n').filter(line => line.trim())
        
        // Apply type filter
        let filtered = lines
        if (type) {
          filtered = filtered.filter(line => line.includes(`/${type}/`))
        }
        
        // Format output
        let output = `Found ${filtered.length} untranslated ${language.toUpperCase()} files`
        
        const filters = [`language=${language.toUpperCase()}`]
        if (component) filters.push(`component=${component}`)
        if (type) filters.push(`type=${type}`)
        if (limit) filters.push(`limit=${limit}`)
        if (offset) filters.push(`offset=${offset}`)
        
        output += ` (${filters.join(', ')})`
        output += '\n\n'
        
        // Show flat list for the specific language
        output += filtered.join('\n')
        
        if (filtered.length > 0) {
          output += `\n\n💡 TIP: Use translate-batch to update multiple ${language.toUpperCase()} files at once`
          if (!limit) {
            output += `\n💡 TIP: Use 'limit' to process in smaller batches (e.g., limit=20)`
          }
          if (limit && filtered.length >= limit) {
            output += `\n💡 TIP: Use 'offset=${offset ? offset + limit : limit}' to get the next batch`
          }
          output += `\n💡 TIP: Work on ${language.toUpperCase()} while other agents handle different languages in parallel`
        } else {
          output += `\n✅ All ${language.toUpperCase()} translations are complete!`
        }
        
        return output
      }

      return result
    } catch (error) {
      return `Error scanning translations: ${error}`
    }
  },
})

// Helper function to list supported languages from Go config
async function listSupportedLanguages(): Promise<string> {
  try {
    // Read the Go config file to get supported languages
    const cwd = process.cwd()
    const configPath = path.join(cwd, "tools/config/application.go")
    
    if (!fs.existsSync(configPath)) {
      return "❌ ERROR: Could not find tools/config/application.go"
    }
    
    const content = fs.readFileSync(configPath, "utf-8")
    
    // Parse SupportedLanguages array
    const langPattern = /var SupportedLanguages = \[\]sdkapi\.SupportedLanguage\{([\s\S]*?)\}/
    const match = content.match(langPattern)
    
    if (!match) {
      return "❌ ERROR: Could not parse SupportedLanguages from tools/config/application.go"
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
      return "❌ ERROR: No languages found in tools/config/application.go"
    }
    
    // Format output
    let output = "📋 Supported Languages\n"
    output += "=====================\n\n"
    
    const defaultLang = languages.find(l => l.code === "en")
    if (defaultLang) {
      output += `${defaultLang.code.toUpperCase()} - ${defaultLang.name} (default)\n`
    }
    
    languages.forEach(lang => {
      if (lang.code !== "en") {
        output += `${lang.code.toUpperCase()} - ${lang.name}\n`
      }
    })
    
    output += `\nTotal: ${languages.length} languages\n`
    output += "\n💡 TIP: Use the language code (lowercase) in other translation operations"
    output += "\n💡 EXAMPLE: translate-scan({ operation: \"summary\", language: \"es\" })"
    output += "\n💡 NOTE: All operations except 'list-languages' require the language parameter"
    
    return output
  } catch (error) {
    return `❌ ERROR reading supported languages: ${error}`
  }
}
