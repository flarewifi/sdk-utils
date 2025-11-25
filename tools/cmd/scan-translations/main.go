package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"tools/config"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// TranslationRef represents a reference to a translation
type TranslationRef struct {
	MsgType     string // label, error, info, etc.
	MsgKey      string // the translation key (original)
	ModifiedKey string // the modified key (truncated if too long)
	FilePath    string // msgtype/filename.txt
}

// UntranslatedEntry represents an untranslated translation entry
type UntranslatedEntry struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	DefaultText string `json:"default_text"`
	FilePath    string `json:"file_path"`
	Language    string `json:"language"`
}

// ScanConfig holds the configuration for the translation scanner
type ScanConfig struct {
	DryRun             bool
	Verbose            bool
	Silent             bool
	MaxConcurrency     int
	JSON               bool
	UntranslatedReport bool
	ListUntranslated   bool
	Language           string
	CorePath           string
	SystemPluginsPath  string
	LocalPluginsPath   string
}

// ShouldSuppressLogs returns true if logs should be suppressed
// Automatically silences logs for JSON/report modes to avoid breaking parsers
func (sc *ScanConfig) ShouldSuppressLogs() bool {
	return sc.Silent || sc.JSON || sc.UntranslatedReport || sc.ListUntranslated
}

// ScanError represents a structured error during scanning
type ScanError struct {
	Path  string
	Op    string
	Err   error
	Fatal bool
}

func (se *ScanError) Error() string {
	return fmt.Sprintf("%s failed for %s: %v", se.Op, se.Path, se.Err)
}

// FileOperation represents a file operation that was performed
type FileOperation struct {
	Operation string // "created", "removed", "synced", "fixed"
	Path      string
	Details   string
}

// TranslationStats holds usage statistics for translations
type TranslationStats struct {
	TypeUsage       map[string]int // Count by translation type (label, error, etc.)
	KeyUsage        map[string]int // Count by translation key
	FileUsage       map[string]int // Count by file path
	UnusedCount     int
	TotalReferences int
	mu              sync.RWMutex // Mutex for thread-safe access
}

// ScanReport holds the results of the scan
type ScanReport struct {
	TotalKeys    int
	UsedKeys     int
	Operations   []FileOperation
	Errors       []ScanError
	Warnings     []string
	Stats        *TranslationStats
	Untranslated []UntranslatedEntry
}

func main() {
	scanConfig := &ScanConfig{}
	flag.BoolVar(&scanConfig.DryRun, "dry-run", false, "Show what would be done without making changes")
	flag.BoolVar(&scanConfig.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&scanConfig.Silent, "silent", false, "Only output if there are errors, warnings, or untranslated content")
	flag.IntVar(&scanConfig.MaxConcurrency, "concurrency", 5, "Maximum concurrent plugin processing")
	flag.BoolVar(&scanConfig.JSON, "json", false, "Output results in JSON format")
	flag.BoolVar(&scanConfig.UntranslatedReport, "untranslated-report", false, "Generate report of untranslated content for AI translation tools")
	flag.BoolVar(&scanConfig.ListUntranslated, "list-untranslated", false, "List all files with untranslated content (content equals key)")
	flag.StringVar(&scanConfig.Language, "language", "", "Filter results by language code (e.g., en, es, fr) - useful for parallel AI agent workflows")
	flag.StringVar(&scanConfig.CorePath, "core-path", "core", "Path to core directory")
	flag.StringVar(&scanConfig.SystemPluginsPath, "system-plugins-path", "plugins/system", "Path to system plugins directory")
	flag.StringVar(&scanConfig.LocalPluginsPath, "local-plugins-path", "data/plugins/local", "Path to local plugins directory")
	flag.Parse()

	if scanConfig.DryRun && !scanConfig.Silent {
		log.Println("DRY RUN MODE - No files will be modified")
	}

	report := &ScanReport{
		Stats: &TranslationStats{
			TypeUsage: make(map[string]int),
			KeyUsage:  make(map[string]int),
			FileUsage: make(map[string]int),
		},
	}

	if !scanConfig.ShouldSuppressLogs() {
		log.Println("Scanning for translation usage...")
	}

	// Collect translation references from core
	coreUsed := make(map[string]*TranslationRef)

	// Scan core
	if err := scanDirectory(scanConfig.CorePath, coreUsed, report.Stats); err != nil {
		report.Errors = append(report.Errors, *err)
	}

	report.TotalKeys = len(coreUsed)
	report.UsedKeys = len(coreUsed)

	if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
		log.Printf("Found %d unique translation references in core", len(coreUsed))
	}

	// Get all supported languages from config
	supportedLanguages := config.SupportedLanguages
	var supportedLangCodes []string
	for _, lang := range supportedLanguages {
		supportedLangCodes = append(supportedLangCodes, lang.Code)
	}
	if !scanConfig.ShouldSuppressLogs() {
		log.Printf("Checking translations for languages: %v", supportedLangCodes)
	}

	// Process core translations
	coreTranslationsPath := filepath.Join(scanConfig.CorePath, "resources", "translations")
	processTranslations(coreTranslationsPath, coreUsed, supportedLangCodes, scanConfig, report)

	// Process system plugins
	processPlugins(scanConfig.SystemPluginsPath, supportedLangCodes, scanConfig, report)

	// Process local plugins
	processPlugins(scanConfig.LocalPluginsPath, supportedLangCodes, scanConfig, report)

	// Calculate unused translations count
	calculateUnusedStats(report, supportedLangCodes, scanConfig)

	// Always collect untranslated entries for the report
	collectUntranslatedEntries(report, supportedLangCodes, scanConfig)

	// Filter by language if specified
	if scanConfig.Language != "" {
		filterReportByLanguage(report, scanConfig.Language)
		if !scanConfig.ShouldSuppressLogs() {
			log.Printf("Filtering results for language: %s", scanConfig.Language)
		}
	}

	// Print report
	printReport(report, scanConfig)

	if !scanConfig.ShouldSuppressLogs() {
		log.Println("Translation scan complete")
	}
}

// filterReportByLanguage filters the report to only include entries for a specific language
func filterReportByLanguage(report *ScanReport, language string) {
	var filtered []UntranslatedEntry
	for _, entry := range report.Untranslated {
		if entry.Language == language {
			filtered = append(filtered, entry)
		}
	}
	report.Untranslated = filtered
}

// processTranslations consolidates all translation file operations into a single pass
func processTranslations(translationsDir string, usedTranslations map[string]*TranslationRef, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	// Collect existing translations in a single pass
	existingTranslations := collectExistingTranslations(translationsDir, supportedLanguages, scanConfig, report)

	// Sync existing translations across languages
	syncExistingTranslations(translationsDir, existingTranslations, supportedLanguages, scanConfig, report)

	// Create missing translations
	createMissingTranslations(translationsDir, usedTranslations, supportedLanguages, scanConfig, report)

	// Remove unused translations
	removeUnusedTranslations(translationsDir, usedTranslations, supportedLanguages, scanConfig, report)

	// Remove unsupported languages
	removeUnsupportedLanguages(translationsDir, supportedLanguages, scanConfig, report)
}

// collectExistingTranslations collects all existing translation files in a single pass
func collectExistingTranslations(translationsDir string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) map[string]string {
	existingTranslations := make(map[string]string)

	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return existingTranslations
	}

	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".txt") {
				return err
			}

			relPath, err := filepath.Rel(langDir, path)
			if err != nil {
				return err
			}

			if _, exists := existingTranslations[relPath]; !exists {
				content, err := os.ReadFile(path)
				if err == nil {
					existingTranslations[relPath] = string(content)
				} else {
					existingTranslations[relPath] = strings.TrimSuffix(filepath.Base(path), ".txt")
				}
			}
			return nil
		})
	}

	if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
		log.Printf("Found %d existing translation keys to sync across languages", len(existingTranslations))
	}

	return existingTranslations
}

func printReport(report *ScanReport, scanConfig *ScanConfig) {
	if scanConfig.ListUntranslated {
		// Output just the untranslated file paths, one per line (unique)
		fileSet := make(map[string]bool)
		for _, entry := range report.Untranslated {
			fileSet[entry.FilePath] = true
		}

		// Sort for consistent output
		var files []string
		for filePath := range fileSet {
			files = append(files, filePath)
		}
		slices.Sort(files)

		for _, filePath := range files {
			fmt.Println(filePath)
		}
		return
	}

	if scanConfig.UntranslatedReport {
		// Output only untranslated entries in JSON format for AI tools
		jsonData, err := json.MarshalIndent(report.Untranslated, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal untranslated report to JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return
	}

	if scanConfig.JSON {
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal report to JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return
	}

	// Silent mode - only print if there are issues
	if scanConfig.Silent {
		hasIssues := len(report.Operations) > 0 ||
			len(report.Errors) > 0 ||
			len(report.Warnings) > 0 ||
			len(report.Untranslated) > 0

		if !hasIssues {
			return // No output - everything is OK
		}
	}

	fmt.Printf("\n=== Translation Scan Report ===\n")
	fmt.Printf("Total translation keys found: %d\n", report.TotalKeys)
	fmt.Printf("Used translation keys: %d\n", report.UsedKeys)
	fmt.Printf("Total translation references: %d\n", report.Stats.TotalReferences)

	if len(report.Stats.TypeUsage) > 0 {
		fmt.Printf("\nTranslation Types Usage:\n")
		for msgType, count := range report.Stats.TypeUsage {
			fmt.Printf("  %s: %d references\n", msgType, count)
		}
	}

	if len(report.Operations) > 0 {
		fmt.Printf("\nFile Operations:\n")
		for _, op := range report.Operations {
			fmt.Printf("  %s: %s", op.Operation, op.Path)
			if op.Details != "" {
				fmt.Printf(" (%s)", op.Details)
			}
			fmt.Printf("\n")
		}
	}

	if len(report.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(report.Errors))
		for _, err := range report.Errors {
			fmt.Printf("  %s\n", err.Error())
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(report.Warnings))
		for _, warning := range report.Warnings {
			fmt.Printf("  %s\n", warning)
		}
	}

	if len(report.Untranslated) > 0 {
		fmt.Printf("\nUntranslated Files (%d):\n", len(report.Untranslated))
		// Group by language for better organization
		byLanguage := make(map[string][]string)
		for _, entry := range report.Untranslated {
			byLanguage[entry.Language] = append(byLanguage[entry.Language], entry.FilePath)
		}

		for lang, files := range byLanguage {
			fmt.Printf("  %s (%d files):\n", lang, len(files))
			for _, file := range files {
				fmt.Printf("    %s\n", file)
			}
		}
	}
}

// processPlugins processes translations for all plugins in the given directory
func processPlugins(pluginsDir string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		report.Errors = append(report.Errors, ScanError{Path: pluginsDir, Op: "reading plugins directory", Err: err, Fatal: false})
		return
	}

	semaphore := make(chan struct{}, scanConfig.MaxConcurrency)
	var wg sync.WaitGroup

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wg.Add(1)
		go func(pluginName string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pluginPath := filepath.Join(pluginsDir, pluginName)
			processPlugin(pluginPath, supportedLanguages, scanConfig, report)
		}(entry.Name())
	}

	wg.Wait()
}

func processPlugin(pluginPath string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	pluginUsed := make(map[string]*TranslationRef)

	if err := scanDirectory(pluginPath, pluginUsed, report.Stats); err != nil {
		report.Errors = append(report.Errors, *err)
		return
	}

	if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
		log.Printf("Found %d unique translation references in plugin %s", len(pluginUsed), filepath.Base(pluginPath))
	}

	translationsPath := filepath.Join(pluginPath, "resources", "translations")
	processTranslations(translationsPath, pluginUsed, supportedLanguages, scanConfig, report)
}

// scanDirectory scans a directory recursively for .go and .templ files
func scanDirectory(dir string, usedTranslations map[string]*TranslationRef, stats *TranslationStats) *ScanError {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Process .go and .templ files
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".templ") {
			if scanErr := scanFile(path, usedTranslations, stats); scanErr != nil {
				return scanErr.Err
			}
		}

		return nil
	})

	if err != nil {
		return &ScanError{Path: dir, Op: "scanning directory", Err: err, Fatal: false}
	}
	return nil
}

// scanFile scans a single file for Translate() calls
func scanFile(filePath string, usedTranslations map[string]*TranslationRef, stats *TranslationStats) *ScanError {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &ScanError{Path: filePath, Op: "reading file", Err: err, Fatal: false}
	}

	// Pattern to match Translate("type", "key")
	// Handles both .Translate and Translate
	pattern := regexp.MustCompile(`\.?Translate\(\s*"([^"]+)"\s*,\s*"([^"]+)"`)
	matches := pattern.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) >= 3 {
			msgType := match[1]
			msgKey := match[2]

			// Validate translation key
			modifiedKey := validateTranslationKey(msgKey, filePath)

			// Create the translation file path key
			filename := sdkutils.FilenameFromTranslationKey(modifiedKey)
			translationKey := filepath.Join(msgType, filename+".txt")

			// Store the translation reference with original and modified keys
			usedTranslations[translationKey] = &TranslationRef{
				MsgType:     msgType,
				MsgKey:      msgKey,
				ModifiedKey: modifiedKey,
				FilePath:    translationKey,
			}

			// Collect statistics (thread-safe)
			stats.mu.Lock()
			stats.TypeUsage[msgType]++
			stats.KeyUsage[translationKey]++
			stats.FileUsage[filePath]++
			stats.TotalReferences++
			stats.mu.Unlock()
		}
	}

	// Check for concatenated translation keys
	badPattern := regexp.MustCompile(`\.?Translate\(\s*"([^"]+)"\s*,\s*([^"]+)\)`)
	badMatches := badPattern.FindAllStringSubmatch(string(content), -1)

	for _, match := range badMatches {
		if len(match) >= 3 {
			msgType := match[1]
			secondArg := match[2]

			// Panic if the second argument contains concatenation (+)
			if strings.Contains(secondArg, "+") {
				log.Panicf("Concatenated translation key detected in %s: Translate(%q, %s)", filePath, msgType, secondArg)
			}
		}
	}
	return nil
}

// validateTranslationKey validates and modifies the translation key according to guidelines
func validateTranslationKey(key, filePath string) string {
	// Check for snake_case (underscores)
	if strings.Contains(key, "_") {
		log.Panicf("Snake_case translation key detected in %s: %q. Use Title Case instead (e.g., 'Used Voucher')", filePath, key)
	}

	// Check word count, limit to 10 words
	fields := strings.Fields(key)
	if len(fields) > 10 {
		modifiedKey := strings.Join(fields[:10], " ") + " (truncated)"
		return modifiedKey
	}

	return key
}

// syncExistingTranslations ensures that all existing translation files exist in all supported languages
func syncExistingTranslations(translationsDir string, existingTranslations map[string]string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {

	// Now ensure all translation keys exist in all languages
	for translationKey, defaultContent := range existingTranslations {
		for _, lang := range supportedLanguages {
			langDir := filepath.Join(translationsDir, lang)

			// Create language directory if it doesn't exist
			if _, err := os.Stat(langDir); os.IsNotExist(err) {
				if scanConfig.DryRun {
					report.Operations = append(report.Operations, FileOperation{
						Operation: "create_dir",
						Path:      langDir,
						Details:   "language directory",
					})
				} else {
					if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
						log.Printf("Creating language directory: %s", langDir)
					}
					if err := os.MkdirAll(langDir, 0755); err != nil {
						report.Errors = append(report.Errors, ScanError{Path: langDir, Op: "creating language directory", Err: err, Fatal: false})
						continue
					}
				}
			}

			translationFilePath := filepath.Join(langDir, translationKey)

			// Check if file exists
			if _, err := os.Stat(translationFilePath); os.IsNotExist(err) {
				// Create the directory if it doesn't exist
				dir := filepath.Dir(translationFilePath)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					if scanConfig.DryRun {
						report.Operations = append(report.Operations, FileOperation{
							Operation: "create_dir",
							Path:      dir,
							Details:   "translation type directory",
						})
					} else {
						if err := os.MkdirAll(dir, 0755); err != nil {
							report.Errors = append(report.Errors, ScanError{Path: dir, Op: "creating directory", Err: err, Fatal: false})
							continue
						}
					}
				}

				// Create the file with default content
				if scanConfig.DryRun {
					report.Operations = append(report.Operations, FileOperation{
						Operation: "synced",
						Path:      translationFilePath,
						Details:   fmt.Sprintf("default: %s", defaultContent),
					})
				} else {
					if err := os.WriteFile(translationFilePath, []byte(defaultContent), 0644); err != nil {
						report.Errors = append(report.Errors, ScanError{Path: translationFilePath, Op: "syncing translation file", Err: err, Fatal: false})
					} else {
						if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
							log.Printf("Synced translation [%s]: %s", lang, translationFilePath)
						}
					}
				}
			}
		}
	}
}

// createMissingTranslations creates translation files that are referenced in code but don't exist
// Creates files for ALL supported languages, creating language directories if needed
func createMissingTranslations(translationsDir string, usedTranslations map[string]*TranslationRef, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		// Create the base translations directory if it doesn't exist
		if scanConfig.DryRun {
			report.Operations = append(report.Operations, FileOperation{
				Operation: "create_dir",
				Path:      translationsDir,
				Details:   "base translations directory",
			})
		} else {
			if err := os.MkdirAll(translationsDir, 0755); err != nil {
				report.Errors = append(report.Errors, ScanError{Path: translationsDir, Op: "creating translations directory", Err: err, Fatal: false})
				return
			}
		}
	}

	// Process each supported language
	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)

		// Create language directory if it doesn't exist
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			if scanConfig.DryRun {
				report.Operations = append(report.Operations, FileOperation{
					Operation: "create_dir",
					Path:      langDir,
					Details:   "language directory",
				})
			} else {
				if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
					log.Printf("Creating language directory: %s", langDir)
				}
				if err := os.MkdirAll(langDir, 0755); err != nil {
					report.Errors = append(report.Errors, ScanError{Path: langDir, Op: "creating language directory", Err: err, Fatal: false})
					continue
				}
			}
		}

		// Check each used translation
		for _, ref := range usedTranslations {
			translationFilePath := filepath.Join(langDir, ref.FilePath)

			// Check if file exists
			if _, err := os.Stat(translationFilePath); os.IsNotExist(err) {
				// Create the directory if it doesn't exist
				dir := filepath.Dir(translationFilePath)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					if scanConfig.DryRun {
						report.Operations = append(report.Operations, FileOperation{
							Operation: "create_dir",
							Path:      dir,
							Details:   "translation type directory",
						})
					} else {
						if err := os.MkdirAll(dir, 0755); err != nil {
							report.Errors = append(report.Errors, ScanError{Path: dir, Op: "creating directory", Err: err, Fatal: false})
							continue
						}
					}
				}

				// Create the file with the modified key as default content
				if scanConfig.DryRun {
					report.Operations = append(report.Operations, FileOperation{
						Operation: "created",
						Path:      translationFilePath,
						Details:   fmt.Sprintf("default: %s", ref.ModifiedKey),
					})
				} else {
					if err := os.WriteFile(translationFilePath, []byte(ref.ModifiedKey), 0644); err != nil {
						report.Errors = append(report.Errors, ScanError{Path: translationFilePath, Op: "creating translation file", Err: err, Fatal: false})
					} else {
						if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
							log.Printf("Created missing translation [%s]: %s (default: %s)", lang, translationFilePath, ref.ModifiedKey)
						}
					}
				}
			}
		}
	}
}

// removeUnusedTranslations removes translation files that are not referenced in code
// for all supported languages
func removeUnusedTranslations(translationsDir string, usedTranslations map[string]*TranslationRef, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return
	}

	// Process each supported language
	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
				log.Printf("Language directory not found: %s", langDir)
			}
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
				if scanConfig.DryRun {
					report.Operations = append(report.Operations, FileOperation{
						Operation: "removed",
						Path:      path,
						Details:   "unused translation",
					})
				} else {
					if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
						log.Printf("Removing unused translation [%s]: %s", lang, path)
					}
					if err := os.Remove(path); err != nil {
						report.Errors = append(report.Errors, ScanError{Path: path, Op: "removing unused translation", Err: err, Fatal: false})
					}
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error walking translations directory %s: %v", langDir, err)
		}
	}
}

// calculateUnusedStats calculates statistics about unused translations
func calculateUnusedStats(report *ScanReport, supportedLanguages []string, scanConfig *ScanConfig) {
	// For now, unused count is calculated during the removeUnusedTranslations phase
	// and tracked in the operations. This could be enhanced to provide more detailed stats.
	report.Stats.UnusedCount = 0 // Placeholder - would need more complex logic to calculate properly
}

// collectUntranslatedEntries scans for translation files that still have default content
func collectUntranslatedEntries(report *ScanReport, supportedLanguages []string, scanConfig *ScanConfig) {
	// Process core translations
	coreTranslationsPath := filepath.Join(scanConfig.CorePath, "resources", "translations")
	collectUntranslatedFromDir(coreTranslationsPath, supportedLanguages, report)

	// Process system plugins
	entries, err := os.ReadDir(scanConfig.SystemPluginsPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				pluginPath := filepath.Join(scanConfig.SystemPluginsPath, entry.Name(), "resources", "translations")
				collectUntranslatedFromDir(pluginPath, supportedLanguages, report)
			}
		}
	}

	// Process local plugins
	entries, err = os.ReadDir(scanConfig.LocalPluginsPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				pluginPath := filepath.Join(scanConfig.LocalPluginsPath, entry.Name(), "resources", "translations")
				collectUntranslatedFromDir(pluginPath, supportedLanguages, report)
			}
		}
	}
}

// collectUntranslatedFromDir collects untranslated entries from a specific translations directory
func collectUntranslatedFromDir(translationsDir string, supportedLanguages []string, report *ScanReport) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return
	}

	for _, lang := range supportedLanguages {
		langDir := filepath.Join(translationsDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".txt") {
				return err
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			contentStr := strings.TrimSpace(string(content))

			// Get relative path from language directory
			relPath, err := filepath.Rel(langDir, path)
			if err != nil {
				return err
			}

			// Parse the translation type and key from the path
			parts := strings.Split(relPath, "/")
			if len(parts) != 2 {
				return nil
			}

			msgType := parts[0]
			filename := parts[1]
			key := strings.TrimSuffix(filename, ".txt")

			// Check if this appears to be untranslated
			// We consider it untranslated if the content equals the key
			// (which is how we create default content)
			// Skip English language as it's the source language
			if contentStr == key && lang != "en" {
				entry := UntranslatedEntry{
					Key:         key,
					Type:        msgType,
					DefaultText: contentStr,
					FilePath:    path,
					Language:    lang,
				}
				report.Untranslated = append(report.Untranslated, entry)
			}

			return nil
		})
	}
}

// removeUnsupportedLanguages removes language directories that are not in the supported languages list
func removeUnsupportedLanguages(translationsDir string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
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
			if scanConfig.DryRun {
				report.Operations = append(report.Operations, FileOperation{
					Operation: "removed",
					Path:      langDir,
					Details:   "unsupported language directory",
				})
			} else {
				if scanConfig.Verbose && !scanConfig.ShouldSuppressLogs() {
					log.Printf("Removing unsupported language directory: %s", langDir)
				}
				if err := os.RemoveAll(langDir); err != nil {
					report.Errors = append(report.Errors, ScanError{Path: langDir, Op: "removing unsupported language directory", Err: err, Fatal: false})
				}
			}
		}
	}
}
