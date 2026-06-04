package api

import (
	"path/filepath"
	"strings"

	"core/utils/config"
	"core/utils/flaretmpl"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// truncateTranslationKey truncates translation keys that exceed 120 characters
// This matches the logic used by the translation scanner
func truncateTranslationKey(key string) string {
	const maxLength = 120
	const suffix = " (truncated)"

	if len(key) <= maxLength {
		return key
	}

	// Find last space before limit to avoid cutting mid-word
	truncateAt := maxLength
	for i := maxLength - 1; i > 0; i-- {
		if key[i] == ' ' {
			truncateAt = i
			break
		}
	}

	return strings.TrimSpace(key[:truncateAt]) + suffix
}

// TranslateMessage is the unified translation function used by all APIs
// It converts translation keys to filesystem-safe filenames and auto-creates missing translations
func TranslateMessage(translationsDir string, msgtype string, msgk string, pairs ...any) string {
	if len(pairs)%2 != 0 {
		return "Invalid number of translation params."
	}

	appcfg, _ := config.ReadApplicationConfig()

	truncatedKey := truncateTranslationKey(msgk)

	f := filepath.Join(translationsDir, appcfg.Lang, msgtype, truncatedKey)

	tmpl, err := flaretmpl.GetTextTemplate(f)
	if err != nil {
		sdkutils.FsWriteFile(f, []byte(msgk))
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
		return msgk
	}

	s := output.String()
	return strings.TrimSpace(s)
}
