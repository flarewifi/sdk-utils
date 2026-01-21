package api

import (
	"log"
	"path/filepath"
	"strings"

	"core/utils/config"
	"core/utils/flaretmpl"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// truncateTranslationKey truncates translation keys that exceed 10 words
// This matches the logic used by the translation scanner
func truncateTranslationKey(key string) string {
	fields := strings.Fields(key)
	wordCount := len(fields)

	// Truncate to 10 words if exceeds limit
	if wordCount > 10 {
		return strings.Join(fields[:10], " ") + " (truncated)"
	}

	return key
}

// TranslateMessage is the unified translation function used by all APIs
// It converts translation keys to filesystem-safe filenames and auto-creates missing translations
func TranslateMessage(translationsDir string, msgtype string, msgk string, pairs ...any) string {
	if len(pairs)%2 != 0 {
		log.Printf("Translate pairs: %+v", pairs)
		return "Invalid number of translation params."
	}

	appcfg, _ := config.ReadApplicationConfig()

	// Apply the same truncation logic as the translation scanner
	truncatedKey := truncateTranslationKey(msgk)

	// Convert translation key to filesystem-safe filename
	filename := sdkutils.FilenameFromTranslationKey(truncatedKey)
	f := filepath.Join(translationsDir, appcfg.Lang, msgtype, filename)

	tmpl, err := flaretmpl.GetTextTemplate(f)
	if err != nil {
		// Auto-create missing translation file with key as default content
		createErr := sdkutils.FsWriteFile(f, []byte(msgk))
		if createErr != nil {
			log.Printf("Warning: Translation file not found and could not create: %s, error: %v", f, createErr)
		} else {
			log.Printf("Created missing translation: %s with default content: %s", f, msgk)
		}
		return msgk
	}

	vdata := map[any]any{}
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		value := pairs[i+1]
		vdata[key] = value
	}

	var output strings.Builder
	err = tmpl.Execute(&output, vdata)
	if err != nil {
		log.Println("Error executing translation template "+f, err)
		return msgk
	}

	s := output.String()
	return strings.TrimSpace(s)
}
