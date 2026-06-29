package translations

import (
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"

	"core/utils/config"
	"core/utils/env"
	"core/utils/flaretmpl"
)

// useCache mirrors flaretmpl: production caches parsed catalogs/templates; dev
// re-reads on every call so edits to <lang>.json show live without a rebuild.
var useCache = env.GO_ENV != env.ENV_DEV

var (
	catalogCache sync.Map // "<dir>\x00<lang>" -> *loadEntry
	tmplCache    sync.Map // resolved string -> *texttemplate.Template (successes only)
)

// loadEntry guarantees a catalog is read from disk exactly once per (dir,lang)
// even under concurrent first-use: the winner of LoadOrStore runs once.Do; the
// inner Catalog maps are immutable after load, so reads need no further locking.
type loadEntry struct {
	once sync.Once
	cat  Catalog
}

// Translate resolves msgk for msgtype in the active language's catalog under
// translationsDir and interpolates paired params via the <% %> template engine.
// It is the single runtime entry behind every api.Translate / HttpHelpers.Translate
// call. There is NO legacy file fallback and NO write: a missing key (or missing
// <lang>.json) resolves to the English source text msgk.
func Translate(translationsDir, msgtype, msgk string, pairs ...any) string {
	if len(pairs)%2 != 0 {
		return "Invalid number of translation params."
	}

	lang := "en"
	if cfg, err := config.GetCachedAppConfig(); err == nil && cfg.Lang != "" {
		lang = cfg.Lang
	}

	resolved := msgk
	if m := getCatalog(translationsDir, lang)[msgtype]; m != nil {
		if v, ok := m[msgk]; ok && v != "" {
			resolved = v
		}
	}
	return interpolate(resolved, msgk, pairs)
}

// ClearCache drops all cached catalogs and templates. Called on a language change
// (the next lookup reloads the new language) and after catalog edits.
func ClearCache() {
	catalogCache.Range(func(k, _ any) bool { catalogCache.Delete(k); return true })
	tmplCache.Range(func(k, _ any) bool { tmplCache.Delete(k); return true })
}

// =============================================================================
// HELPERS
// =============================================================================

func getCatalog(dir, lang string) Catalog {
	if !useCache {
		return loadCatalog(dir, lang)
	}
	e, _ := catalogCache.LoadOrStore(dir+"\x00"+lang, &loadEntry{})
	entry := e.(*loadEntry)
	entry.once.Do(func() { entry.cat = loadCatalog(dir, lang) })
	return entry.cat
}

// loadCatalog reads <dir>/<lang>.json. A missing or invalid file yields an empty
// catalog so every lookup falls back to the English source text — never an error
// and never the old per-file tree.
func loadCatalog(dir, lang string) Catalog {
	cat, err := ReadCatalog(filepath.Join(dir, lang+".json"))
	if err != nil {
		return Catalog{}
	}
	return cat
}

// interpolate applies paired params to resolved. Fast path: a string with no "<%"
// is returned trimmed without parsing (the vast majority of messages). On any
// parse/exec error it falls back to the English source text.
func interpolate(resolved, fallback string, pairs []any) string {
	if !strings.Contains(resolved, "<%") {
		return strings.TrimSpace(resolved)
	}

	tmpl := getTemplate(resolved)
	if tmpl == nil {
		return fallback
	}

	vdata := make(map[any]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		vdata[pairs[i]] = pairs[i+1]
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, vdata); err != nil {
		return fallback
	}
	return strings.TrimSpace(sb.String())
}

func getTemplate(s string) *texttemplate.Template {
	if useCache {
		if v, ok := tmplCache.Load(s); ok {
			return v.(*texttemplate.Template)
		}
	}
	t, err := flaretmpl.ParseTextTemplate(s)
	if err != nil {
		return nil // failures are rare; don't cache, just re-parse next time
	}
	if useCache {
		tmplCache.Store(s, t)
	}
	return t
}
