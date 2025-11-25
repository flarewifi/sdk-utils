package sdkutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureLanguageAvailable ensures a language is available in uncompressed form
// If compressed, it decompresses it; if already uncompressed, does nothing
// In dev mode (when no .tar.gz files exist), this just verifies the directory exists
func EnsureLanguageAvailable(langDir string) error {
	parentDir := filepath.Dir(langDir)
	langName := filepath.Base(langDir)
	tarGzPath := filepath.Join(parentDir, langName+".tar.gz")

	// Check if uncompressed directory exists
	if _, err := os.Stat(langDir); err == nil {
		// Directory exists, nothing to do
		return nil
	}

	// Check if compressed file exists
	if _, err := os.Stat(tarGzPath); err != nil {
		// In dev mode, there are no .tar.gz files
		// If the directory doesn't exist and there's no compressed file, it's an error
		return fmt.Errorf("translation directory not found: %s", langName)
	}

	// Decompress the language (only happens in production mode)
	return decompressLanguage(tarGzPath)
}

// SwitchLanguage handles switching from one language to another
// Compresses the old language and decompresses the new one
func SwitchLanguage(oldLang, newLang string, translationsDir string) error {
	oldLangDir := filepath.Join(translationsDir, oldLang)
	newLangDir := filepath.Join(translationsDir, newLang)

	// Ensure new language is available
	if err := EnsureLanguageAvailable(newLangDir); err != nil {
		return fmt.Errorf("failed to ensure new language is available: %w", err)
	}

	// Compress old language if it exists as directory
	if _, err := os.Stat(oldLangDir); err == nil {
		if err := compressLanguage(oldLangDir); err != nil {
			return fmt.Errorf("failed to compress old language: %w", err)
		}
	}

	return nil
}

// forEachPluginTranslationsDir iterates over all plugin translation directories
// and calls the provided function for each one
func forEachPluginDir(rootDir string, fn func(pluginDir string) error) error {
	installDir := filepath.Join(rootDir, "plugins", "installed")
	entries, err := os.ReadDir(installDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(installDir, entry.Name())
		_, err := GetPluginInfoFromPath(pluginPath)
		if err != nil {
			continue // Not a valid plugin, skip
		}

		if err := fn(pluginPath); err != nil {
			return err
		}
	}

	return nil
}

// CompressAllTranslations compresses all translation directories for deployment
// Scans core/resources/translations/ and all installed plugins' translation directories
// Only used in production builds (//go:build !dev && !staging)
func CompressAllTranslations(rootDir string) error {
	// Compress core translations
	coreDir := filepath.Join(rootDir, "core")
	if err := CompressPluginTranslations(coreDir); err != nil {
		return fmt.Errorf("failed to compress core translations: %w", err)
	}

	// Compress plugin translations
	return forEachPluginDir(rootDir, CompressPluginTranslations)
}

// CompressAllUnusedLanguages compresses all languages except current one for core and all plugins
// Used in production to optimize space by keeping only active language decompressed
// Only used in production builds (//go:build !dev && !staging)
func CompressAllUnusedLanguages(rootDir, currentLang string) error {
	// Compress unused core translations
	coreTranslationsDir := filepath.Join(rootDir, "core", "resources", "translations")
	if err := CompressUnusedLanguages(coreTranslationsDir, currentLang); err != nil {
		return fmt.Errorf("failed to compress unused core translations: %w", err)
	}

	// Compress unused plugin translations
	return forEachPluginDir(rootDir, func(pluginDir string) error {
		translationsDir := filepath.Join(pluginDir, "resources", "translations")
		return CompressUnusedLanguages(translationsDir, currentLang)
	})
}

// CompressPluginTranslations compresses all language subdirectories in a translations directory
func CompressPluginTranslations(pluginDir string) error {
	translationsDir := filepath.Join(pluginDir, "resources", "translations")
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		// Directory doesn't exist, skip
		return nil
	}

	entries, err := os.ReadDir(translationsDir)
	if err != nil {
		return fmt.Errorf("failed to read translations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		langDir := filepath.Join(translationsDir, entry.Name())
		if err := compressLanguage(langDir); err != nil {
			return fmt.Errorf("failed to compress language %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// CompressUnusedLanguages compresses all languages except the current one
// Used in production to optimize space by keeping only active language decompressed
// Only used in production builds (//go:build !dev && !staging)
func CompressUnusedLanguages(translationsDir, currentLang string) error {
	entries, err := os.ReadDir(translationsDir)
	if err != nil {
		return fmt.Errorf("failed to read translations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		langName := entry.Name()
		// Skip the current language
		if langName == currentLang {
			continue
		}

		langDir := filepath.Join(translationsDir, langName)
		if err := compressLanguage(langDir); err != nil {
			return fmt.Errorf("failed to compress unused language %s: %w", langName, err)
		}
	}

	return nil
}

// EnsureTranslations ensures the current language translations are available for a directory (core or plugin)
// If compressed, it decompresses them; if already uncompressed, does nothing
// The dir parameter should be the core or plugin root directory containing "resources/translations"
func EnsureTranslations(dir, currentLang string) error {
	translationsDir := filepath.Join(dir, "resources", "translations")

	// Check if translations directory exists
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		// No translations directory, skip
		return nil
	}

	// Ensure the current language is available
	langDir := filepath.Join(translationsDir, currentLang)
	return EnsureLanguageAvailable(langDir)
}

// SwitchAllLanguages switches languages for core and all installed plugins
// Compresses the old language and decompresses the new one for each
func SwitchAllLanguages(oldLang, newLang string) error {
	// Switch core translations
	coreTranslationsDir := filepath.Join(PathCoreDir, "resources", "translations")
	if err := SwitchLanguage(oldLang, newLang, coreTranslationsDir); err != nil {
		return fmt.Errorf("failed to switch core translations: %w", err)
	}

	// Switch plugin translations
	return forEachPluginDir(PathAppDir, func(pluginDir string) error {
		translationsDir := filepath.Join(pluginDir, "resources", "translations")
		// Check if translations directory exists
		if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
			// No translations for this plugin, skip
			return nil
		}

		// Switch language for this plugin
		if err := SwitchLanguage(oldLang, newLang, translationsDir); err != nil {
			// Log error but don't fail - some plugins might not have all languages
			return nil
		}
		return nil
	})
}

// compressLanguage compresses a language directory to a .tar.gz archive
// Only used in production builds (//go:build !dev && !staging)
func compressLanguage(langDir string) error {
	// Get the parent directory and language name
	parentDir := filepath.Dir(langDir)
	langName := filepath.Base(langDir)

	// Create the tar.gz file path
	tarGzPath := filepath.Join(parentDir, langName+".tar.gz")

	// Remove existing compressed file if it exists
	if _, err := os.Stat(tarGzPath); err == nil {
		if err := os.Remove(tarGzPath); err != nil {
			return fmt.Errorf("failed to remove existing compressed file: %w", err)
		}
	}

	// Compress the directory using native Go tar utilities
	if err := CompressTar(langDir, tarGzPath); err != nil {
		return fmt.Errorf("failed to compress language directory: %w", err)
	}

	// Remove the uncompressed directory
	if err := os.RemoveAll(langDir); err != nil {
		return fmt.Errorf("failed to remove uncompressed directory: %w", err)
	}

	return nil
}

// decompressLanguage extracts a .tar.gz archive to a language directory
// Only used in production builds (//go:build !dev && !staging)
func decompressLanguage(tarGzPath string) error {
	lang := strings.TrimSuffix(filepath.Base(tarGzPath), ".tar.gz")
	// Get the parent directory
	parentDir := filepath.Dir(tarGzPath)
	extractPath := filepath.Join(parentDir, lang)

	// Extract the archive using native Go tar utilities
	if err := Untar(tarGzPath, extractPath); err != nil {
		return fmt.Errorf("failed to decompress language archive: %w", err)
	}

	// Remove the compressed file
	if err := os.Remove(tarGzPath); err != nil {
		return fmt.Errorf("failed to remove compressed file: %w", err)
	}

	return nil
}
