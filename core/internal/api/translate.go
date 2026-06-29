package api

import "core/utils/translations"

// TranslateMessage is the unified translation entry used by all APIs. It resolves
// the message against the per-language JSON catalog under translationsDir and
// interpolates paired params. The catalog loading, caching, English-source
// fallback, and <% %> interpolation all live in core/utils/translations — this is
// a thin seam so PluginUtils/HttpHelpers keep their signatures.
func TranslateMessage(translationsDir string, msgtype string, msgk string, pairs ...any) string {
	return translations.Translate(translationsDir, msgtype, msgk, pairs...)
}
