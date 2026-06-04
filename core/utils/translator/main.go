package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"core/utils/config"
)

// TranslationRef represents a reference to a translation
type TranslationRef struct {
	MsgType     string // label, error, info, etc.
	MsgKey      string // the translation key (original)
	ModifiedKey string // the modified key (truncated if too long)
	FilePath    string // msgtype/filename (no extension)
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
	Validate           bool
	MinCoverage        int
	CriticalThreshold  int
	Strict             bool
	Color              bool
	MarkdownReport     string
	Limit              int
	Offset             int
	Component          string
	Summary            bool
	Compact            bool
	CreateMissing      bool
}

// ShouldSuppressLogs returns true if logs should be suppressed
// Automatically silences logs for JSON/report modes to avoid breaking parsers
func (sc *ScanConfig) ShouldSuppressLogs() bool {
	return sc.Silent || sc.JSON || sc.UntranslatedReport || sc.ListUntranslated || sc.Validate || sc.MarkdownReport != "" || sc.Summary
}

// IsReadOnly returns true if the tool should not modify files
func (sc *ScanConfig) IsReadOnly() bool {
	return sc.DryRun || sc.Validate || sc.MarkdownReport != ""
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
	Validation   *ValidationReport
}

// ValidationReport holds validation results per component and language
type ValidationReport struct {
	Components     []ComponentValidation
	TotalIssues    int
	CriticalIssues int
	HasFailures    bool
	HasWarnings    bool
}

// ComponentValidation holds validation results for a single component
type ComponentValidation struct {
	Name            string
	Path            string
	EnglishCount    int
	LanguageResults []LanguageValidation
}

// LanguageValidation holds validation results for a single language
type LanguageValidation struct {
	Language          string
	Translated        int
	Untranslated      int
	Missing           int
	Total             int
	Percentage        int
	Status            string // "complete", "warning", "critical", "missing_dir"
	MissingFiles      []string
	UntranslatedFiles []string
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
	flag.BoolVar(&scanConfig.Validate, "validate", false, "Validation mode - read-only check of translation coverage (for CI/CD)")
	flag.IntVar(&scanConfig.MinCoverage, "min-coverage", 80, "Minimum translation coverage percentage (used with --validate)")
	flag.IntVar(&scanConfig.CriticalThreshold, "critical-threshold", 0, "Critical failure threshold percentage (used with --validate)")
	flag.BoolVar(&scanConfig.Strict, "strict", false, "Strict mode - fail on any untranslated content (used with --validate)")
	flag.BoolVar(&scanConfig.Color, "color", true, "Enable colored output (auto-disabled for non-TTY)")
	flag.StringVar(&scanConfig.MarkdownReport, "markdown-report", "", "Generate markdown report to specified file")
	flag.IntVar(&scanConfig.Limit, "limit", 0, "Limit output to N entries (for pagination, 0 = no limit)")
	flag.IntVar(&scanConfig.Offset, "offset", 0, "Skip first N entries (for pagination)")
	flag.StringVar(&scanConfig.Component, "component", "", "Filter by component name (e.g., 'core', 'com.flarego.default-theme')")
	flag.BoolVar(&scanConfig.Summary, "summary", false, "Output summary only without detailed file lists (reduces token usage)")
	flag.BoolVar(&scanConfig.Compact, "compact", false, "Compact JSON output with minimal whitespace (reduces token usage)")
	flag.BoolVar(&scanConfig.CreateMissing, "create-missing", false, "Create missing translation files from code references (filename truncated if >10 words, content preserves original)")
	flag.Parse()



	report := &ScanReport{
		Stats: &TranslationStats{
			TypeUsage: make(map[string]int),
			KeyUsage:  make(map[string]int),
			FileUsage: make(map[string]int),
		},
		Validation: &ValidationReport{
			Components: []ComponentValidation{},
		},
	}

	// Get all supported languages from config
	supportedLanguages := config.SupportedLanguages
	var supportedLangCodes []string
	for _, lang := range supportedLanguages {
		supportedLangCodes = append(supportedLangCodes, lang.Code)
	}

	// Validation mode - read-only check
	if scanConfig.Validate || scanConfig.MarkdownReport != "" {

		// Validate core translations
		coreTranslationsPath := filepath.Join(scanConfig.CorePath, "resources", "translations")
		validateComponent(coreTranslationsPath, "Core Application", supportedLangCodes, scanConfig, report)

		// Validate system plugins
		validatePlugins(scanConfig.SystemPluginsPath, supportedLangCodes, scanConfig, report)

		// Validate local plugins
		validatePlugins(scanConfig.LocalPluginsPath, supportedLangCodes, scanConfig, report)

		// Apply filters before generating reports
		if scanConfig.Language != "" {
			filterReportByLanguage(report, scanConfig.Language)
		}
		if scanConfig.Component != "" {
			filterReportByComponent(report, scanConfig.Component)
		}

		// Generate markdown report if requested
		if scanConfig.MarkdownReport != "" {
			generateMarkdownReport(report, scanConfig)
			os.Exit(0)
		}

		// Print validation report
		printValidationReport(report, scanConfig)

		// Exit with appropriate code
		if report.Validation.HasFailures {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Normal mode - scan and sync translations

	// Collect translation references from core
	coreUsed := make(map[string]*TranslationRef)

	// Scan core
	if err := scanDirectory(scanConfig.CorePath, coreUsed, report.Stats); err != nil {
		report.Errors = append(report.Errors, *err)
	}

	report.TotalKeys = len(coreUsed)
	report.UsedKeys = len(coreUsed)

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
	}

	// Filter by component if specified
	if scanConfig.Component != "" {
		filterReportByComponent(report, scanConfig.Component)
	}

	// Apply pagination if specified
	if scanConfig.Limit > 0 || scanConfig.Offset > 0 {
		applyPagination(report, scanConfig)
	}

	// Print report
	printReport(report, scanConfig)
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

	// Also filter validation results
	if report.Validation != nil {
		for i := range report.Validation.Components {
			var filteredLangResults []LanguageValidation
			for _, langResult := range report.Validation.Components[i].LanguageResults {
				if langResult.Language == language {
					filteredLangResults = append(filteredLangResults, langResult)
				}
			}
			report.Validation.Components[i].LanguageResults = filteredLangResults
		}
	}
}

// filterReportByComponent filters the report to only include entries for a specific component
func filterReportByComponent(report *ScanReport, component string) {
	// Normalize component name
	componentLower := strings.ToLower(component)

	// Filter untranslated entries by file path
	var filtered []UntranslatedEntry
	for _, entry := range report.Untranslated {
		// Check if the file path contains the component name
		if componentLower == "core" && strings.Contains(entry.FilePath, "/core/resources/translations/") {
			filtered = append(filtered, entry)
		} else if strings.Contains(strings.ToLower(entry.FilePath), componentLower) {
			filtered = append(filtered, entry)
		}
	}
	report.Untranslated = filtered

	// Filter validation components
	if report.Validation != nil {
		var filteredComponents []ComponentValidation
		for _, comp := range report.Validation.Components {
			compNameLower := strings.ToLower(comp.Name)
			if componentLower == "core" && strings.Contains(compNameLower, "core") {
				filteredComponents = append(filteredComponents, comp)
			} else if strings.Contains(compNameLower, componentLower) {
				filteredComponents = append(filteredComponents, comp)
			}
		}
		report.Validation.Components = filteredComponents
	}
}

// applyPagination applies offset and limit to the report entries
func applyPagination(report *ScanReport, scanConfig *ScanConfig) {
	offset := scanConfig.Offset
	limit := scanConfig.Limit

	// Apply to untranslated entries
	total := len(report.Untranslated)
	if offset >= total {
		report.Untranslated = []UntranslatedEntry{}
		return
	}

	start := offset
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	report.Untranslated = report.Untranslated[start:end]
}

// processTranslations consolidates all translation file operations into a single pass
func processTranslations(translationsDir string, usedTranslations map[string]*TranslationRef, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	// Collect existing translations in a single pass
	existingTranslations := collectExistingTranslations(translationsDir, supportedLanguages, scanConfig, report)

	// Sync existing translations across languages
	syncExistingTranslations(translationsDir, existingTranslations, supportedLanguages, scanConfig, report)

	// Create missing translations if --create-missing flag is set
	// This scans code for .Translate() calls and creates files that don't exist
	// Filename uses ModifiedKey (truncated if >10 words), content uses MsgKey (original full text)
	if scanConfig.CreateMissing {
		createMissingTranslations(translationsDir, usedTranslations, supportedLanguages, scanConfig, report)
	}

	// Remove unused translations
	removeUnusedTranslations(translationsDir, usedTranslations, supportedLanguages, scanConfig, report)

	// Remove unsupported languages
	removeUnsupportedLanguages(translationsDir, supportedLanguages, scanConfig, report)
}

// collectExistingTranslations collects all existing translation files from English directory only
// English is the source of truth - all other languages sync from it
func collectExistingTranslations(translationsDir string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) map[string]string {
	existingTranslations := make(map[string]string)

	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return existingTranslations
	}

	// Only scan English directory - it's the source of truth
	enDir := filepath.Join(translationsDir, "en")
	if _, err := os.Stat(enDir); os.IsNotExist(err) {
		return existingTranslations
	}

	filepath.Walk(enDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(enDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err == nil {
			existingTranslations[relPath] = string(content)
		} else {
			existingTranslations[relPath] = filepath.Base(path)
		}
		return nil
	})

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

	if scanConfig.Summary {
		// Output summary statistics only
		printSummary(report, scanConfig)
		return
	}

	if scanConfig.UntranslatedReport {
		// Output only untranslated entries in JSON format for AI tools
		var jsonData []byte
		var err error
		if scanConfig.Compact {
			jsonData, err = json.Marshal(report.Untranslated)
		} else {
			jsonData, err = json.MarshalIndent(report.Untranslated, "", "  ")
		}
		if err != nil {
		}
		fmt.Println(string(jsonData))
		return
	}

	if scanConfig.JSON {
		var jsonData []byte
		var err error
		if scanConfig.Compact {
			jsonData, err = json.Marshal(report)
		} else {
			jsonData, err = json.MarshalIndent(report, "", "  ")
		}
		if err != nil {
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

// printSummary prints a compact summary of the translation status
func printSummary(report *ScanReport, scanConfig *ScanConfig) {
	summary := make(map[string]interface{})

	// Basic stats
	summary["total_keys"] = report.TotalKeys
	summary["used_keys"] = report.UsedKeys
	summary["total_references"] = report.Stats.TotalReferences
	summary["total_untranslated"] = len(report.Untranslated)
	summary["total_errors"] = len(report.Errors)
	summary["total_warnings"] = len(report.Warnings)
	summary["total_operations"] = len(report.Operations)

	// Group untranslated by language
	byLanguage := make(map[string]int)
	for _, entry := range report.Untranslated {
		byLanguage[entry.Language]++
	}
	summary["untranslated_by_language"] = byLanguage

	// Validation summary if available
	if report.Validation != nil && len(report.Validation.Components) > 0 {
		validationSummary := make(map[string]interface{})
		validationSummary["total_components"] = len(report.Validation.Components)
		validationSummary["total_issues"] = report.Validation.TotalIssues
		validationSummary["critical_issues"] = report.Validation.CriticalIssues
		validationSummary["has_failures"] = report.Validation.HasFailures
		validationSummary["has_warnings"] = report.Validation.HasWarnings

		// Component summaries
		componentSummaries := make([]map[string]interface{}, 0)
		for _, comp := range report.Validation.Components {
			compSummary := make(map[string]interface{})
			compSummary["name"] = comp.Name
			compSummary["english_count"] = comp.EnglishCount
			compSummary["languages"] = len(comp.LanguageResults)

			// Language status counts
			statusCounts := make(map[string]int)
			for _, langResult := range comp.LanguageResults {
				statusCounts[langResult.Status]++
			}
			compSummary["status_counts"] = statusCounts

			componentSummaries = append(componentSummaries, compSummary)
		}
		validationSummary["components"] = componentSummaries
		summary["validation"] = validationSummary
	}

	// Output as JSON
	var jsonData []byte
	var err error
	if scanConfig.Compact {
		jsonData, err = json.Marshal(summary)
	} else {
		jsonData, err = json.MarshalIndent(summary, "", "  ")
	}
	if err != nil {
	}
	fmt.Println(string(jsonData))
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
	pattern := regexp.MustCompile(`\.?Translate\(\s*"([^"]+?)"\s*,\s*"([^"]+?)"`)
	matches := pattern.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) >= 3 {
			msgType := match[1]
			msgKey := match[2]

			// Validate translation key
			modifiedKey := validateTranslationKey(msgKey, filePath)

			// Skip if validation failed (empty string returned)
			if modifiedKey == "" {
				continue
			}

			// Create the translation file path key
			// Use the key directly as filename (no URL encoding)
			// Translation files are stored with actual characters for readability
			translationKey := filepath.Join(msgType, modifiedKey)

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

	return nil
}

// validateTranslationKey validates and modifies the translation key according to guidelines
func validateTranslationKey(key, filePath string) string {
	// Check for snake_case (underscores)
	if strings.Contains(key, "_") {
		return "" // Return empty to signal invalid key
	}

	// Check character count, limit to 120 characters
	const maxLength = 120
	const warnThreshold = 100
	const suffix = " (truncated)"
	charCount := len(key)

	// WARNING if exceeds 120 characters - key will be truncated
	if charCount > maxLength {
		// Find last space before limit to avoid cutting mid-word
		truncateAt := maxLength
		for i := maxLength - 1; i > 0; i-- {
			if key[i] == ' ' {
				truncateAt = i
				break
			}
		}
		modifiedKey := strings.TrimSpace(key[:truncateAt]) + suffix
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

				// Create the file with the ORIGINAL key as default content (for translator context)
				// The filename uses ModifiedKey (truncated if needed), but content preserves original text
				if scanConfig.DryRun {
					report.Operations = append(report.Operations, FileOperation{
						Operation: "created",
						Path:      translationFilePath,
						Details:   fmt.Sprintf("default: %s", ref.MsgKey),
					})
				} else {
					if err := os.WriteFile(translationFilePath, []byte(ref.MsgKey), 0644); err != nil {
						report.Errors = append(report.Errors, ScanError{Path: translationFilePath, Op: "creating translation file", Err: err, Fatal: false})
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
			continue
		}

		// Walk through translation directories for this language
		filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Get relative path from language directory
			relPath, err := filepath.Rel(langDir, path)
			if err != nil {
				return err
			}

			// Create key as "msgtype/filename"
			translationKey := relPath

			// Also check with truncated key for files with long names
			// Parse msgType and key from the path
			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) >= 2 {
				msgType := parts[0]
				filename := parts[len(parts)-1]
				key := filename

				// Apply same truncation logic as validateTranslationKey
				fields := strings.Fields(key)
				if len(fields) > 10 {
					truncatedKey := strings.Join(fields[:10], " ") + " (truncated)"
					truncatedFilename := truncatedKey
					truncatedPath := filepath.Join(msgType, truncatedFilename)

					// Check both original and truncated keys
					if usedTranslations[translationKey] != nil || usedTranslations[truncatedPath] != nil {
						// File is used (either with original or truncated key)
						return nil
					}
				}
			}

			// Check if this translation is used
			if usedTranslations[translationKey] == nil {
				if scanConfig.DryRun {
					report.Operations = append(report.Operations, FileOperation{
						Operation: "removed",
						Path:      path,
						Details:   "unused translation",
					})
				} else {
					if err := os.Remove(path); err != nil {
						report.Errors = append(report.Errors, ScanError{Path: path, Op: "removing unused translation", Err: err, Fatal: false})
					}
				}
			}

			return nil
		})

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
			if err != nil || info.IsDir() {
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
			key := filename

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
				if err := os.RemoveAll(langDir); err != nil {
					report.Errors = append(report.Errors, ScanError{Path: langDir, Op: "removing unsupported language directory", Err: err, Fatal: false})
				}
			}
		}
	}
}

// validateComponent validates translations for a single component
func validateComponent(translationsDir, componentName string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return
	}

	enDir := filepath.Join(translationsDir, "en")
	if _, err := os.Stat(enDir); os.IsNotExist(err) {
		return
	}

	// Count English files
	enCount := 0
	filepath.Walk(enDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			enCount++
		}
		return nil
	})

	if enCount == 0 {
		return
	}

	component := ComponentValidation{
		Name:            componentName,
		Path:            translationsDir,
		EnglishCount:    enCount,
		LanguageResults: []LanguageValidation{},
	}

	// Check each supported language (skip English as it's the source)
	for _, lang := range supportedLanguages {
		if lang == "en" {
			continue
		}
		langResult := validateLanguage(translationsDir, lang, enCount, scanConfig)
		component.LanguageResults = append(component.LanguageResults, langResult)

		// Update report counters
		if langResult.Status == "critical" || langResult.Status == "missing_dir" {
			report.Validation.CriticalIssues++
			report.Validation.HasFailures = true
		}
		if langResult.Untranslated > 0 || langResult.Missing > 0 {
			report.Validation.TotalIssues++
			if langResult.Percentage < scanConfig.MinCoverage {
				report.Validation.HasWarnings = true
			}
		}
	}

	report.Validation.Components = append(report.Validation.Components, component)
}

// validateLanguage validates translations for a single language
func validateLanguage(translationsDir, lang string, enCount int, scanConfig *ScanConfig) LanguageValidation {
	result := LanguageValidation{
		Language:          lang,
		Total:             enCount,
		MissingFiles:      []string{},
		UntranslatedFiles: []string{},
	}

	langDir := filepath.Join(translationsDir, lang)
	if _, err := os.Stat(langDir); os.IsNotExist(err) {
		result.Status = "missing_dir"
		result.Missing = enCount
		result.Percentage = 0
		return result
	}

	enDir := filepath.Join(translationsDir, "en")

	// Check each English file
	filepath.Walk(enDir, func(enPath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(enDir, enPath)
		if err != nil {
			return nil
		}

		langPath := filepath.Join(langDir, relPath)

		// Check if file exists
		if _, err := os.Stat(langPath); os.IsNotExist(err) {
			result.Missing++
			result.Untranslated++
			result.MissingFiles = append(result.MissingFiles, relPath)
			return nil
		}

		// Check if file is identical to English (untranslated)
		enContent, err1 := os.ReadFile(enPath)
		langContent, err2 := os.ReadFile(langPath)
		if err1 == nil && err2 == nil && string(enContent) == string(langContent) {
			result.Untranslated++
			result.UntranslatedFiles = append(result.UntranslatedFiles, relPath)
		}

		return nil
	})

	result.Translated = enCount - result.Untranslated
	if enCount > 0 {
		result.Percentage = (result.Translated * 100) / enCount
	}

	// Determine status
	if result.Percentage >= 100 {
		result.Status = "complete"
	} else if result.Percentage < scanConfig.CriticalThreshold {
		result.Status = "critical"
	} else if result.Percentage < scanConfig.MinCoverage {
		result.Status = "warning"
	} else if result.Untranslated > 0 {
		result.Status = "warning"
	} else {
		result.Status = "complete"
	}

	return result
}

// validatePlugins validates translations for all plugins in a directory
func validatePlugins(pluginsDir string, supportedLanguages []string, scanConfig *ScanConfig, report *ScanReport) {
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(pluginsDir, entry.Name())
		translationsPath := filepath.Join(pluginPath, "resources", "translations")
		componentName := "Plugin: " + entry.Name()

		validateComponent(translationsPath, componentName, supportedLanguages, scanConfig, report)
	}
}

// printValidationReport prints the validation report with colors
func printValidationReport(report *ScanReport, scanConfig *ScanConfig) {
	// If summary mode, use the summary printer
	if scanConfig.Summary {
		printSummary(report, scanConfig)
		return
	}

	// Color codes
	red := "\033[0;31m"
	yellow := "\033[1;33m"
	green := "\033[0;32m"
	blue := "\033[0;34m"
	nc := "\033[0m"

	if !scanConfig.Color {
		red, yellow, green, blue, nc = "", "", "", "", ""
	}

	fmt.Printf("%s=== FlareHotspot Translation Validation ===%s\n\n", blue, nc)

	for _, component := range report.Validation.Components {
		fmt.Printf("%sChecking: %s%s\n", blue, component.Name, nc)
		fmt.Printf("  English files: %d\n\n", component.EnglishCount)

		for _, langResult := range component.LanguageResults {
			statusIcon := "✓"
			statusColor := green

			switch langResult.Status {
			case "missing_dir":
				statusIcon = "✗"
				statusColor = red
				fmt.Printf("  %s%s %s: Missing translation directory%s\n", statusColor, statusIcon, langResult.Language, nc)
				continue
			case "critical":
				statusIcon = "✗"
				statusColor = red
			case "warning":
				statusIcon = "⚠"
				statusColor = yellow
			}

			fmt.Printf("  %s%s %s: %d/%d translated (%d%%)%s\n",
				statusColor, statusIcon, langResult.Language,
				langResult.Translated, langResult.Total, langResult.Percentage, nc)

			if langResult.Missing > 0 {
				fmt.Printf("    %sMissing files: %d%s\n", red, langResult.Missing, nc)
			}
			if langResult.Untranslated > 0 && langResult.Missing < langResult.Untranslated {
				fmt.Printf("    %sIdentical to EN: %d%s\n", yellow, langResult.Untranslated-langResult.Missing, nc)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("%s=== Summary ===%s\n", blue, nc)
	fmt.Printf("Total issues found: %d\n", report.Validation.TotalIssues)
	fmt.Printf("Critical issues (< %d%% translated): %d\n\n", scanConfig.CriticalThreshold, report.Validation.CriticalIssues)

	// Exit message
	if scanConfig.Strict {
		if report.Validation.TotalIssues > 0 {
			fmt.Printf("%s❌ Translation validation FAILED (strict mode)%s\n", red, nc)
			fmt.Printf("%sHint: Set --strict=false to allow commits with warnings%s\n", yellow, nc)
			report.Validation.HasFailures = true
		}
	} else {
		if report.Validation.CriticalIssues > 0 {
			fmt.Printf("%s❌ Translation validation FAILED%s\n", red, nc)
			fmt.Printf("%sCritical: Some languages have %d%% or less translations (essentially empty)%s\n", red, scanConfig.CriticalThreshold, nc)
			fmt.Printf("%sFix critical issues or use 'git commit --no-verify' to bypass%s\n", yellow, nc)
			report.Validation.HasFailures = true
		} else if report.Validation.TotalIssues > 0 {
			fmt.Printf("%s⚠️  Translation validation PASSED with warnings%s\n", yellow, nc)
			fmt.Printf("%sFound %d translation issues - please address when possible:%s\n", yellow, report.Validation.TotalIssues, nc)
			fmt.Printf("%s  - Languages below %d%% completion should be prioritized%s\n", yellow, scanConfig.MinCoverage, nc)
			fmt.Printf("%s  - Run 'make translation-report' for detailed breakdown%s\n", yellow, nc)
			fmt.Printf("%s✅ Commit allowed%s\n", green, nc)
		} else {
			fmt.Printf("%s✅ Translation validation PASSED%s\n", green, nc)
			fmt.Printf("%sAll translations are complete!%s\n", green, nc)
		}
	}
}

// generateMarkdownReport generates a markdown report file
func generateMarkdownReport(report *ScanReport, scanConfig *ScanConfig) error {
	var sb strings.Builder

	sb.WriteString("# Translation Status Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Add filter information if applied
	if scanConfig.Language != "" || scanConfig.Component != "" || scanConfig.Limit > 0 || scanConfig.Offset > 0 {
		sb.WriteString("## Filters Applied\n\n")
		if scanConfig.Language != "" {
			sb.WriteString(fmt.Sprintf("- **Language**: %s\n", scanConfig.Language))
		}
		if scanConfig.Component != "" {
			sb.WriteString(fmt.Sprintf("- **Component**: %s\n", scanConfig.Component))
		}
		if scanConfig.Offset > 0 {
			sb.WriteString(fmt.Sprintf("- **Offset**: %d\n", scanConfig.Offset))
		}
		if scanConfig.Limit > 0 {
			sb.WriteString(fmt.Sprintf("- **Limit**: %d\n", scanConfig.Limit))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Overview\n\n")
	sb.WriteString("This report lists all untranslated files across all supported languages.\n\n")

	for _, component := range report.Validation.Components {
		sb.WriteString("\n---\n\n")
		sb.WriteString(fmt.Sprintf("## %s\n\n", component.Name))
		sb.WriteString(fmt.Sprintf("Total English files: %d\n\n", component.EnglishCount))

		for _, langResult := range component.LanguageResults {
			if langResult.Status == "missing_dir" {
				sb.WriteString(fmt.Sprintf("\n### ❌ %s - MISSING DIRECTORY\n\n", langResult.Language))
				sb.WriteString(fmt.Sprintf("**Action Required:** Create translation directory at `%s/%s`\n\n", component.Path, langResult.Language))
				continue
			}

			statusIcon := "✅"
			if langResult.Status == "critical" {
				statusIcon = "❌"
			} else if langResult.Status == "warning" {
				statusIcon = "⚠️"
			}

			if langResult.Translated == langResult.Total {
				sb.WriteString(fmt.Sprintf("\n### %s %s - Fully Translated (%d%%)\n\n", statusIcon, langResult.Language, langResult.Percentage))
				sb.WriteString(fmt.Sprintf("All %d files are translated.\n\n", langResult.Total))
			} else {
				sb.WriteString(fmt.Sprintf("\n### %s %s - %d/%d translated (%d%%)\n\n", statusIcon, langResult.Language, langResult.Translated, langResult.Total, langResult.Percentage))

				if len(langResult.MissingFiles) > 0 {
					sb.WriteString(fmt.Sprintf("\n#### Missing Files (%d)\n\n", len(langResult.MissingFiles)))
					sb.WriteString("These files need to be created:\n\n")
					for _, file := range langResult.MissingFiles {
						sb.WriteString(fmt.Sprintf("- `%s`\n", file))
					}
					sb.WriteString("\n")
				}

				if len(langResult.UntranslatedFiles) > 0 {
					sb.WriteString(fmt.Sprintf("\n#### Untranslated Files (%d)\n\n", len(langResult.UntranslatedFiles)))
					sb.WriteString("These files exist but contain English text:\n\n")
					for _, file := range langResult.UntranslatedFiles {
						sb.WriteString(fmt.Sprintf("- `%s`\n", file))
					}
					sb.WriteString("\n")
				}
			}
		}
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("## Action Items\n\n")
	sb.WriteString("### Priority 1: Critical (< 50% translated)\n\n")
	sb.WriteString("Languages with less than 50% translation coverage need immediate attention.\n\n")
	sb.WriteString("### Priority 2: Incomplete (50-80% translated)\n\n")
	sb.WriteString("Languages with 50-80% coverage should be completed soon.\n\n")
	sb.WriteString("### Priority 3: Nearly Complete (80-99% translated)\n\n")
	sb.WriteString("Languages with 80-99% coverage need final touches.\n\n")
	sb.WriteString("## Translation Workflow\n\n")
	sb.WriteString("1. **Find untranslated files** in this report\n")
	sb.WriteString("2. **Create/update translation files** with proper translations\n")
	sb.WriteString("3. **Run validation**: `make translations-check`\n")
	sb.WriteString("4. **Commit changes** once validation passes\n\n")
	sb.WriteString("## Notes\n\n")
	sb.WriteString("- Translation files should contain ONLY the translated text\n")
	sb.WriteString("- Do NOT copy English text into other language files\n")
	sb.WriteString("- Use native speakers or professional translation services\n")
	sb.WriteString("- Maintain consistent terminology across all translations\n")

	return os.WriteFile(scanConfig.MarkdownReport, []byte(sb.String()), 0644)
}
