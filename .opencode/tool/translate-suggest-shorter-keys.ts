import { tool } from "@opencode-ai/plugin"

export default tool({
  description: "Scan FlareHotspot codebase for long translation keys (8+ words) and suggest shorter alternatives with context",
  args: {
    minWords: tool.schema
      .number()
      .optional()
      .default(8)
      .describe("Minimum word count to flag as too long (default: 8, warning at 8, truncation at 10)"),
    component: tool.schema
      .string()
      .optional()
      .describe("Filter by component name (e.g., 'core', 'com.flarego.wifi-hotspot')"),
    limit: tool.schema
      .number()
      .optional()
      .describe("Limit output to N entries (for pagination)"),
    suggestAlternatives: tool.schema
      .boolean()
      .optional()
      .default(true)
      .describe("Include AI-suggested shorter alternatives for each long key"),
  },
  async execute(args, context) {
    const { minWords = 8, component, limit, suggestAlternatives = true } = args

    try {
      // Run translator tool with validation to get long keys
      const command = `go run -tags="dev" $(pwd)/core/utils/translator --validate --json`
      const result = await Bun.$`sh -c ${command}`.text()

      // Parse the output to extract long keys
      const lines = result.split('\n')
      const longKeys: Array<{
        key: string
        wordCount: number
        filePath: string
        type: string
        truncated: boolean
      }> = []

      // Regex patterns to extract info from log lines
      const infoPattern = /ℹ️\s+INFO: Translation key is close to 10 word limit \((\d+) words\) in ([^:]+): "([^"]+)"/
      const warnPattern = /⚠️\s+WARNING: Translation key exceeds 10 word limit \((\d+) words\) in ([^:]+)\s+Original key: "([^"]+)"/

      for (const line of lines) {
        const infoMatch = line.match(infoPattern)
        const warnMatch = line.match(warnPattern)
        
        if (infoMatch) {
          const wordCount = parseInt(infoMatch[1])
          const filePath = infoMatch[2]
          const key = infoMatch[3]
          
          if (wordCount >= minWords && (!component || filePath.includes(component))) {
            // Extract type from file path or context
            const type = extractTypeFromContext(key, filePath)
            longKeys.push({ key, wordCount, filePath, type, truncated: false })
          }
        } else if (warnMatch) {
          const wordCount = parseInt(warnMatch[1])
          const filePath = warnMatch[2]
          const key = warnMatch[3]
          
          if (wordCount >= minWords && (!component || filePath.includes(component))) {
            const type = extractTypeFromContext(key, filePath)
            longKeys.push({ key, wordCount, filePath, type, truncated: true })
          }
        }
      }

      // Remove duplicates (same key may appear in multiple files)
      const uniqueKeys = new Map<string, typeof longKeys[0]>()
      for (const entry of longKeys) {
        if (!uniqueKeys.has(entry.key) || uniqueKeys.get(entry.key)!.wordCount < entry.wordCount) {
          uniqueKeys.set(entry.key, entry)
        }
      }

      let uniqueLongKeys = Array.from(uniqueKeys.values())
      
      // Sort by word count (longest first)
      uniqueLongKeys.sort((a, b) => b.wordCount - a.wordCount)

      // Apply limit if specified
      if (limit && limit > 0) {
        uniqueLongKeys = uniqueLongKeys.slice(0, limit)
      }

      if (uniqueLongKeys.length === 0) {
        return `✅ No long translation keys found (>=${minWords} words)\n\nAll translation keys are concise and follow best practices!`
      }

      // Build output
      let output = `📏 Long Translation Keys Report\n${'='.repeat(50)}\n\n`
      output += `Found ${uniqueLongKeys.length} keys with ${minWords}+ words\n`
      output += `(⚠️ Keys with 11+ words will be truncated to 10 words + "(truncated)")\n\n`

      for (let i = 0; i < uniqueLongKeys.length; i++) {
        const entry = uniqueLongKeys[i]
        const icon = entry.truncated ? '⚠️' : 'ℹ️'
        const status = entry.truncated ? 'WILL BE TRUNCATED' : 'Close to limit'
        
        output += `${icon} ${i + 1}. [${entry.wordCount} words] ${status}\n`
        output += `   Type: ${entry.type}\n`
        output += `   File: ${entry.filePath}\n`
        output += `   Original: "${entry.key}"\n`
        
        if (suggestAlternatives) {
          const suggestions = generateShorterSuggestions(entry.key, entry.type)
          if (suggestions.length > 0) {
            output += `   💡 Suggestions:\n`
            for (const suggestion of suggestions) {
              output += `      • "${suggestion}" (${countWords(suggestion)} words)\n`
            }
          }
        }
        
        output += `\n`
      }

      output += `\n📋 Next Steps:\n`
      output += `1. Review each long key and choose a shorter alternative\n`
      output += `2. Update the source code with the shorter key\n`
      output += `3. Update existing translation files if they already exist\n`
      output += `4. Run validation again to ensure all keys are under 10 words\n`
      output += `\n💡 TIP: Shorter keys are easier to maintain and translate consistently`
      output += `\n💡 TIP: Use variables for dynamic content instead of embedding it in keys`

      return output
    } catch (error) {
      return `❌ Error analyzing long translation keys: ${error}`
    }
  },
})

function extractTypeFromContext(key: string, filePath: string): string {
  // Try to infer type from key content or context
  const lowerKey = key.toLowerCase()
  
  if (lowerKey.includes('error') || lowerKey.includes('failed') || lowerKey.includes('invalid') || lowerKey.includes('unable')) {
    return 'error'
  } else if (lowerKey.includes('success') || lowerKey.includes('created') || lowerKey.includes('updated') || lowerKey.includes('deleted')) {
    return 'success'
  } else if (lowerKey.includes('warning') || lowerKey.includes('caution') || lowerKey.includes('careful')) {
    return 'warning'
  } else if (lowerKey.includes('info') || lowerKey.includes('please') || lowerKey.includes('wait')) {
    return 'info'
  }
  
  // Default to label for UI text
  return 'label'
}

function countWords(text: string): number {
  return text.trim().split(/\s+/).length
}

function generateShorterSuggestions(key: string, type: string): string[] {
  const suggestions: string[] = []
  const wordCount = countWords(key)
  
  // Don't suggest if already short enough
  if (wordCount < 8) {
    return []
  }
  
  // Strategy 1: Remove redundant words
  let shorter = key
    .replace(/\s+(will|shall)\s+/gi, ' ')
    .replace(/\s+please\s+/gi, ' ')
    .replace(/\s+the\s+/gi, ' ')
    .replace(/\s+a\s+/gi, ' ')
    .replace(/\s+an\s+/gi, ' ')
    .replace(/\s{2,}/g, ' ')
    .trim()
  
  if (shorter !== key && countWords(shorter) < wordCount && countWords(shorter) <= 10) {
    suggestions.push(shorter)
  }
  
  // Strategy 2: Use action-focused phrasing for errors/success
  if (type === 'error') {
    // Extract the core action/problem
    if (key.match(/invalid|not valid/i)) {
      const subject = key.match(/invalid\s+(.+?)(?:\s+format|\s+type|\.|$)/i)?.[1]
      if (subject) {
        suggestions.push(`Invalid ${subject}`)
      }
    } else if (key.match(/unable to|cannot|failed to/i)) {
      const action = key.match(/(?:unable to|cannot|failed to)\s+(.+?)(?:\.|$)/i)?.[1]
      if (action) {
        suggestions.push(`Could not ${action}`)
        suggestions.push(`Failed to ${action}`)
      }
    }
  } else if (type === 'success') {
    const action = key.match(/^(.+?)\s+(?:successfully|success)/i)?.[1]
    if (action) {
      suggestions.push(`${action} successful`)
    }
  } else if (type === 'info' || type === 'warning') {
    // For informational messages, focus on the key information
    if (key.match(/system|device/i) && key.match(/reboot|restart/i)) {
      suggestions.push('System rebooting')
      suggestions.push('Reboot in progress')
    } else if (key.match(/update|updating/i)) {
      suggestions.push('Update in progress')
      suggestions.push('Updating software')
    }
  }
  
  // Strategy 3: Break into multiple smaller keys with variables
  if (key.includes('.') || key.includes(',')) {
    const parts = key.split(/[.,]\s*/)
    if (parts.length >= 2) {
      suggestions.push(parts[0].trim() + ' (consider splitting into separate messages)')
    }
  }
  
  // Remove duplicates and filter out suggestions that are still too long
  return Array.from(new Set(suggestions))
    .filter(s => countWords(s) <= 10 && s.length > 0)
    .slice(0, 3) // Limit to top 3 suggestions
}
