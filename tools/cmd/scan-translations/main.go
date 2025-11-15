package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"tools/config"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// TranslationRef represents a reference to a translation
type TranslationRef struct {
	MsgType  string // label, error, info, etc.
	MsgKey   string // the translation key (original)
	FilePath string // msgtype/filename.txt
}

func main() {
	log.Println("Scanning for translation usage...")

	// Collect translation references from core
	coreUsed := make(map[string]*TranslationRef)

	// Scan core
	scanDirectory("core", coreUsed)

	log.Printf("Found %d unique translation references in core", len(coreUsed))

	// Get all supported languages from config
	supportedLanguages := config.SupportedLanguages
	var supportedLangCodes []string
	for _, lang := range supportedLanguages {
		supportedLangCodes = append(supportedLangCodes, lang.Code)
	}
	log.Printf("Checking translations for languages: %v", supportedLangCodes)

	// First, sync existing translation files across all languages for core
	syncExistingTranslations("core/resources/translations", supportedLangCodes)

	// Create missing translation files for all supported languages for core
	createMissingTranslations("core/resources/translations", coreUsed, supportedLangCodes)

	// Now scan translation files and remove unused ones for all supported languages for core
	removeUnusedTranslations("core/resources/translations", coreUsed, supportedLangCodes)

	// Remove unsupported language directories for core and plugins
	removeUnsupportedLanguages("core/resources/translations", supportedLangCodes)

	// Process system plugins
	processPlugins("plugins/system", supportedLangCodes)

	// Process local plugins
	processPlugins("data/plugins/local", supportedLangCodes)

	log.Println("Translation scan complete")
}

// processPlugins processes translations for all plugins in the given directory
func processPlugins(pluginsDir string, supportedLanguages []string) {
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		log.Printf("Error reading plugins directory %s: %v", pluginsDir, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(pluginsDir, entry.Name())
		pluginUsed := make(map[string]*TranslationRef)

		processPlugin(pluginPath, supportedLanguages, pluginUsed)

		// scanDirectory(pluginPath, pluginUsed)

		// log.Printf("Found %d unique translation references in plugin %s", len(pluginUsed), entry.Name())

		// translationsPath := filepath.Join(pluginPath, "resources", "translations")

		// syncExistingTranslations(translationsPath, supportedLanguages)
		// createMissingTranslations(translationsPath, pluginUsed, supportedLanguages)
		// removeUnusedTranslations(translationsPath, pluginUsed, supportedLanguages)
		// removeUnsupportedLanguages(translationsPath, supportedLanguages)
	}
}

func processPlugin(pluginPath string, supportedLanguages []string, pluginUsed map[string]*TranslationRef) {
	scanDirectory(pluginPath, pluginUsed)
	log.Printf("Found %d unique translation references in plugin %s", len(pluginUsed), filepath.Base(pluginPath))

	translationsPath := filepath.Join(pluginPath, "resources", "translations")

	syncExistingTranslations(translationsPath, supportedLanguages)
	createMissingTranslations(translationsPath, pluginUsed, supportedLanguages)
	removeUnusedTranslations(translationsPath, pluginUsed, supportedLanguages)
	removeUnsupportedLanguages(translationsPath, supportedLanguages)
}

// scanDirectory scans a directory recursively for .go and .templ files
func scanDirectory(dir string, usedTranslations map[string]*TranslationRef) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Process .go and .templ files
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".templ") {
			scanFile(path, usedTranslations)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error scanning directory %s: %v", dir, err)
	}
}

// scanFile scans a single file for Translate() calls
func scanFile(filePath string, usedTranslations map[string]*TranslationRef) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %s: %v", filePath, err)
		return
	}

	// Pattern to match Translate("type", "key")
	// Handles both .Translate and Translate
	pattern := regexp.MustCompile(`\.?Translate\(\s*"([^"]+)"\s*,\s*"([^"]+)"`)
	matches := pattern.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) >= 3 {
			msgType := match[1]
			msgKey := match[2]

			// Create the translation file path key
			filename := sdkutils.FilenameFromTranslationKey(msgKey)
			translationKey := filepath.Join(msgType, filename+".txt")

			// Store the translation reference with original key
			usedTranslations[translationKey] = &TranslationRef{
				MsgType:  msgType,
				MsgKey:   msgKey,
				FilePath: translationKey,
			}
		}
	}
}

// syncExistingTranslations ensures that all existing translation files exist in all supported languages
func syncExistingTranslations(translationsDir string, supportedLanguages []string) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return
	}

	// Collect all existing translation file paths across all languages
	existingTranslations := make(map[string]string) // map[filepath]defaultContent

	// Scan all language directories to find existing translations
	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			continue
		}

		// Walk through this language directory
		filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}

			// Only process .txt files
			if !strings.HasSuffix(path, ".txt") {
				return nil
			}

			// Get relative path from language directory
			relPath, err := filepath.Rel(langDir, path)
			if err != nil {
				return err
			}

			// Store this translation key if we haven't seen it yet
			if _, exists := existingTranslations[relPath]; !exists {
				// Read the content to use as default for other languages
				content, err := os.ReadFile(path)
				if err == nil {
					existingTranslations[relPath] = string(content)
				} else {
					// Use the filename as fallback
					existingTranslations[relPath] = strings.TrimSuffix(filepath.Base(path), ".txt")
				}
			}

			return nil
		})
	}

	log.Printf("Found %d existing translation keys to sync across languages", len(existingTranslations))

	// Now ensure all translation keys exist in all languages
	for translationKey, defaultContent := range existingTranslations {
		for _, lang := range supportedLanguages {
			langDir := filepath.Join(translationsDir, lang)

			// Create language directory if it doesn't exist
			if _, err := os.Stat(langDir); os.IsNotExist(err) {
				log.Printf("Creating language directory: %s", langDir)
				if err := os.MkdirAll(langDir, 0755); err != nil {
					log.Printf("Error creating language directory %s: %v", langDir, err)
					continue
				}
			}

			translationFilePath := filepath.Join(langDir, translationKey)

			// Check if file exists
			if _, err := os.Stat(translationFilePath); os.IsNotExist(err) {
				// Create the directory if it doesn't exist
				dir := filepath.Dir(translationFilePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					log.Printf("Error creating directory %s: %v", dir, err)
					continue
				}

				// Create the file with default content
				if err := os.WriteFile(translationFilePath, []byte(defaultContent), 0644); err != nil {
					log.Printf("Error syncing translation file %s: %v", translationFilePath, err)
				} else {
					log.Printf("Synced translation [%s]: %s", lang, translationFilePath)
				}
			}
		}
	}
}

// createMissingTranslations creates translation files that are referenced in code but don't exist
// Creates files for ALL supported languages, creating language directories if needed
func createMissingTranslations(translationsDir string, usedTranslations map[string]*TranslationRef, supportedLanguages []string) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		// Create the base translations directory if it doesn't exist
		if err := os.MkdirAll(translationsDir, 0755); err != nil {
			log.Printf("Error creating translations directory %s: %v", translationsDir, err)
			return
		}
	}

	// Process each supported language
	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)

		// Create language directory if it doesn't exist
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			log.Printf("Creating language directory: %s", langDir)
			if err := os.MkdirAll(langDir, 0755); err != nil {
				log.Printf("Error creating language directory %s: %v", langDir, err)
				continue
			}
		}

		// Check each used translation
		for _, ref := range usedTranslations {
			translationFilePath := filepath.Join(langDir, ref.FilePath)

			// Check if file exists
			if _, err := os.Stat(translationFilePath); os.IsNotExist(err) {
				// Create the directory if it doesn't exist
				dir := filepath.Dir(translationFilePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					log.Printf("Error creating directory %s: %v", dir, err)
					continue
				}

				// Create the file with the original key as default content
				if err := os.WriteFile(translationFilePath, []byte(ref.MsgKey), 0644); err != nil {
					log.Printf("Error creating translation file %s: %v", translationFilePath, err)
				} else {
					log.Printf("Created missing translation [%s]: %s (default: %s)", lang, translationFilePath, ref.MsgKey)
				}
			}
		}
	}
}

// removeUnusedTranslations removes translation files that are not referenced in code
// for all supported languages
func removeUnusedTranslations(translationsDir string, usedTranslations map[string]*TranslationRef, supportedLanguages []string) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return
	}

	// Process each supported language
	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			log.Printf("Language directory not found: %s", langDir)
			continue
		}

		// Walk through translation directories for this language
		err := filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Only process .txt files
			if !strings.HasSuffix(path, ".txt") {
				return nil
			}

			// Get relative path from language directory
			relPath, err := filepath.Rel(langDir, path)
			if err != nil {
				return err
			}

			// Create key as "msgtype/filename.txt"
			translationKey := relPath

			// Check if this translation is used
			if usedTranslations[translationKey] == nil {
				log.Printf("Removing unused translation [%s]: %s", lang, path)
				if err := os.Remove(path); err != nil {
					log.Printf("Error removing file %s: %v", path, err)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error walking translations directory %s: %v", langDir, err)
		}
	}
}

// removeUnsupportedLanguages removes language directories that are not in the supported languages list
func removeUnsupportedLanguages(translationsDir string, supportedLanguages []string) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(translationsDir)
	if err != nil {
		log.Printf("Error reading translations directory %s: %v", translationsDir, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		lang := entry.Name()
		isSupported := slices.Contains(supportedLanguages, lang)

		if !isSupported {
			langDir := filepath.Join(translationsDir, lang)
			log.Printf("Removing unsupported language directory: %s", langDir)
			if err := os.RemoveAll(langDir); err != nil {
				log.Printf("Error removing unsupported language directory %s: %v", langDir, err)
			}
		}
	}
}
