# Quick Start: Translation Tools

## Installation Complete ✅

All translation tools are now installed and ready to use!

## Quick Commands

### Check Translation Status
```
"Scan for untranslated content"
"Show me translation statistics"
"List all untranslated Spanish files"
```

### Get Help
```
"Show me translation help"
"How do I use translation variables?"
"What are the translation best practices?"
```

### Update Translations
```
"Translate this message to Spanish: [English text]"
"Update all French error messages"
"Batch translate these 10 files to German"
```

## How It Works

When you ask OpenCode about translations, it will automatically use these tools:

1. **translate-scan** - Scans your codebase for translation usage
2. **translate-update** - Updates single translation files
3. **translate-batch** - Updates multiple translation files at once
4. **translate-help** - Provides guidance on the translation system

## Example Workflow

```
You: "Are there any untranslated Spanish labels?"
AI: [Scans using translate-scan]
AI: "Yes, found 25 untranslated Spanish label files."

You: "Translate them all"
AI: [Generates Spanish translations]
AI: [Updates files using translate-batch]
AI: "Done! All 25 Spanish labels have been translated."

You: "Verify the translations"
AI: [Scans again]
AI: "All Spanish labels are now properly translated."
```

## Tool Details

| Tool | Purpose | Common Use Cases |
|------|---------|------------------|
| `translate-scan` | Scan for translations | Find untranslated content, get statistics |
| `translate-update` | Update one file | Fix a single translation, create new translation |
| `translate-batch` | Update many files | Mass translation, bulk updates |
| `translate-help` | Get guidance | Learn translation system, check best practices |

## Supported Languages

English (en), Spanish (es), French (fr), Amharic (am), Arabic (ar), Indonesian (id), Persian (prs), Pashto (ps), Russian (ru), Swahili (sw)

## Translation File Format

```
Location: resources/translations/{lang}/{type}/{key}.txt

Example:
  core/resources/translations/es/label/Welcome.txt
  
Content with variables:
  Welcome <% .username %> to FlareHotspot
```

## Need Help?

Just ask OpenCode:
- "Show me translation help"
- "How do translations work?"
- "What translation types are available?"

The AI will use the `translate-help` tool to provide detailed guidance.

## Per-Language Workflows 🚀

The tools support **per-language filtering** for parallel translation work:

### Single Language Focus
```
"Scan for untranslated Spanish content"
"Translate all Spanish error messages"
"Verify Spanish translations are complete"
```

### Parallel Multi-Language
Different AI agents can work on different languages simultaneously:

- **Agent 1**: Spanish translations
- **Agent 2**: French translations  
- **Agent 3**: Arabic translations

Each agent independently scans, translates, and verifies their assigned language!

## Next Steps

1. Try scanning your codebase: `"Scan for untranslated content"`
2. Check a specific language: `"Show me untranslated French files"`
3. Start translating: `"Translate these error messages to Spanish"`
4. **Advanced**: Divide languages across multiple agents for faster translation

Happy translating! 🌍
