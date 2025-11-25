import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"

export default tool({
  description: "Update FlareHotspot translation files with language-specific translations",
  args: {
    filePath: tool.schema
      .string()
      .describe("Relative path from project root to translation file (e.g., core/resources/translations/es/label/Welcome.txt)"),
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
    const { filePath, content, createMissing = false } = args

    try {
      // Get current working directory (should be project root)
      const cwd = process.cwd()
      const fullPath = path.join(cwd, filePath)

      // Check if file exists
      const fileExists = fs.existsSync(fullPath)
      
      if (!fileExists && !createMissing) {
        return `Error: File does not exist: ${filePath}\nUse createMissing: true to create it.`
      }

      // Validate that this is a translation file
      if (!filePath.includes('/translations/') || !filePath.endsWith('.txt')) {
        return `Error: Invalid translation file path. Must be in /translations/ directory and end with .txt`
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

      // Extract language from path
      const langMatch = filePath.match(/\/translations\/([a-z]{2,3})\//)
      const language = langMatch ? langMatch[1] : 'unknown'
      
      // Write new content
      fs.writeFileSync(fullPath, content, "utf-8")

      const action = fileExists ? "Updated" : "Created"
      return `✅ ${action} translation file [${language.toUpperCase()}]: ${filePath}\n\nOld content:\n${oldContent || "(new file)"}\n\nNew content:\n${content}`
    } catch (error) {
      return `Error updating translation: ${error}`
    }
  },
})
