import { tool } from "@opencode-ai/plugin"

export default tool({
  description: "Scan FlareHotspot codebase for translation usage and untranslated content",
  args: {
    operation: tool.schema
      .enum(["summary", "list-untranslated", "report", "stats"])
      .describe("Operation: summary (overview), list-untranslated (files needing translation), report (JSON for AI), stats (detailed statistics)"),
    type: tool.schema
      .string()
      .optional()
      .describe("Filter by translation type (label, error, success, info, warning)"),
    language: tool.schema
      .string()
      .optional()
      .describe("Filter by language code (en, es, fr, etc.)"),
  },
  async execute(args, context) {
    const { operation = "summary", type, language } = args

    try {
      // Use portable command that works from any directory
      let flags = ""
      
      switch (operation) {
        case "list-untranslated":
          flags = " --list-untranslated"
          break
        case "report":
          flags = " --untranslated-report"
          break
        case "stats":
          flags = " --json"
          break
        case "summary":
        default:
          // Default output is summary
          break
      }

      // Add language filter if specified
      if (language) {
        flags += ` --language=${language}`
      }
      
      // Portable command: cd to project root, then run with $(pwd)
      const command = `go run $(pwd)/tools/cmd/scan-translations/main.go${flags}`
      const result = await Bun.$`sh -c ${command}`.text()

      // Parse and format the output based on operation
      if (operation === "report" || operation === "stats") {
        // Try to parse as JSON
        try {
          const jsonData = JSON.parse(result)
          
          // Apply type filter if specified (language filtering is done by Go tool)
          let filtered = jsonData
          if (Array.isArray(jsonData) && type) {
            filtered = jsonData.filter((entry: any) => entry.type === type)
          }
          
          return JSON.stringify(filtered, null, 2)
        } catch (e) {
          return result
        }
      } else if (operation === "list-untranslated") {
        const lines = result.split('\n').filter(line => line.trim())
        
        // Apply type filter (language filtering is done by Go tool)
        let filtered = lines
        if (type) {
          filtered = filtered.filter(line => line.includes(`/${type}/`))
        }
        
        // Group by language for better organization
        const byLanguage: Record<string, string[]> = {}
        filtered.forEach(line => {
          const match = line.match(/\/translations\/([a-z]{2,3})\//)
          if (match) {
            const lang = match[1]
            if (!byLanguage[lang]) byLanguage[lang] = []
            byLanguage[lang].push(line)
          }
        })
        
        // Format output with language grouping
        let output = `Found ${filtered.length} untranslated files`
        
        if (language) {
          output += ` for language: ${language.toUpperCase()}`
        }
        
        output += '\n\n'
        
        // If specific language requested, show flat list
        if (language) {
          output += filtered.join('\n')
          if (filtered.length > 0) {
            output += `\n\n💡 TIP: Use translate-batch to update multiple ${language.toUpperCase()} files at once`
          }
        } else {
          // Show grouped by language for easier delegation
          const languages = Object.keys(byLanguage).sort()
          languages.forEach(lang => {
            output += `\n${lang.toUpperCase()} (${byLanguage[lang].length} files):\n`
            output += byLanguage[lang].slice(0, 5).join('\n')
            if (byLanguage[lang].length > 5) {
              output += `\n... and ${byLanguage[lang].length - 5} more ${lang} files`
            }
            output += '\n'
          })
          
          output += '\n💡 TIP: Use language="xx" parameter to see all files for a specific language'
          output += '\n💡 TIP: Different AI agents can work on different languages in parallel'
          output += '\n💡 EXAMPLE: "Scan for untranslated Spanish files" → filters to language="es"'
        }
        
        return output
      }

      return result
    } catch (error) {
      return `Error scanning translations: ${error}`
    }
  },
})
