package main

// ⚠️  DEPRECATED: Please use update-translations.py instead
//
// This Go script is deprecated. Use the Python script for better features:
// - UTF-8 validation
// - Backup creation
// - Dry-run mode
// - Better error handling
// - Works from any directory
//
// Migration:
//   ./scripts/update-translations.py --file=translations.json
//
// The Python script is more portable and feature-rich.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Translation represents a translation entry
type Translation struct {
	Type         string            `json:"type"`
	Key          string            `json:"key"`
	Translations map[string]string `json:"translations"`
}

func main() {
	fmt.Println("⚠️  WARNING: This Go script is deprecated.")
	fmt.Println("   Please use: ./scripts/update-translations.py --file=<your-file>.json")
	fmt.Println()
	fmt.Println("Continuing with deprecated script...")
	fmt.Println()

	var translationsFile string
	flag.StringVar(&translationsFile, "file", "", "JSON file containing translations")
	flag.Parse()

	if translationsFile == "" {
		fmt.Println("Usage: go run scripts/batch-translate.go -file=translations.json")
		fmt.Println()
		fmt.Println("💡 Recommended: ./scripts/update-translations.py --file=translations.json")
		fmt.Println("\nExample JSON format:")
		fmt.Println(`{
  "type": "warning",
  "key": "The purchase has been cancelled",
  "translations": {
    "en": "The purchase has been cancelled",
    "es": "La compra ha sido cancelada",
    "fr": "L'achat a été annulé"
  }
}`)
		os.Exit(1)
	}

	// Read the JSON file
	data, err := os.ReadFile(translationsFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var translation Translation
	if err := json.Unmarshal(data, &translation); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Base directory for translations
	baseDir := "core/resources/translations"

	// Create translation files for each language
	for lang, text := range translation.Translations {
		// Create directory
		dir := filepath.Join(baseDir, lang, translation.Type)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			continue
		}

		// File path
		filePath := filepath.Join(dir, translation.Key+".txt")

		// Check if file exists and read it first (for compatibility with tools)
		if _, err := os.Stat(filePath); err == nil {
			existingContent, _ := os.ReadFile(filePath)
			fmt.Printf("Updating existing file: %s (was: %q)\n", filePath, string(existingContent))
		} else {
			fmt.Printf("Creating new file: %s\n", filePath)
		}

		// Write the translation
		if err := os.WriteFile(filePath, []byte(text), 0644); err != nil {
			fmt.Printf("Error writing file %s: %v\n", filePath, err)
			continue
		}
	}

	fmt.Println("\nTranslation files processed successfully!")
}
