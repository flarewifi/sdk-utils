package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"core/utils/translations"
)

// parseArgs decodes tool arguments into dst, tolerating absent/partial args
// (every tool validates the fields it actually requires).
func parseArgs(args json.RawMessage, dst any) {
	if len(args) > 0 {
		_ = json.Unmarshal(args, dst)
	}
}

// =============================================================================
// Tool schemas (advertised via tools/list)
// =============================================================================

func toolDefs() []map[string]any {
	str := map[string]any{"type": "string"}
	obj := func(props map[string]any, required ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}
	return []map[string]any{
		{
			"name":        "list_components",
			"description": "List every translatable component (core + plugins) with its language codes and English key count.",
			"inputSchema": obj(map[string]any{}),
		},
		{
			"name":        "summarize",
			"description": "Translation coverage summary: per component and language, the total English keys, how many are translated, and the percent. Optionally filter to one component.",
			"inputSchema": obj(map[string]any{"component": str}),
		},
		{
			"name":        "list_keys",
			"description": "List translation entries for a component. Returns msgtype, English key, the translation in the given language (or the English source if untranslated), and a translated flag. Supports msgtype filter, case-insensitive substring search, and limit/offset paging.",
			"inputSchema": obj(map[string]any{
				"component": str,
				"lang":      str,
				"msgtype":   str,
				"search":    str,
				"limit":     map[string]any{"type": "integer"},
				"offset":    map[string]any{"type": "integer"},
			}, "component"),
		},
		{
			"name":        "get_translation",
			"description": "Get one translation: the value in the requested language, the English source, and whether it is translated.",
			"inputSchema": obj(map[string]any{"component": str, "msgtype": str, "key": str, "lang": str}, "component", "msgtype", "key", "lang"),
		},
		{
			"name":        "find_untranslated",
			"description": "List English keys that are not yet translated in the given language (the work queue for that language). Returns msgtype, key (English source), optional limit.",
			"inputSchema": obj(map[string]any{"component": str, "lang": str, "limit": map[string]any{"type": "integer"}}, "component", "lang"),
		},
		{
			"name":        "set_translation",
			"description": "Set the translation for one key in one language. The key must already exist in en.json (run sync first for new code strings). Writes the pretty-printed catalog.",
			"inputSchema": obj(map[string]any{"component": str, "lang": str, "msgtype": str, "key": str, "value": str}, "component", "lang", "msgtype", "key", "value"),
		},
		{
			"name":        "set_translations",
			"description": "Set many translations for one component+language at once. items is an array of {msgtype, key, value}. Each key must exist in en.json.",
			"inputSchema": obj(map[string]any{
				"component": str,
				"lang":      str,
				"items": map[string]any{"type": "array", "items": obj(map[string]any{"msgtype": str, "key": str, "value": str}, "msgtype", "key", "value")},
			}, "component", "lang", "items"),
		},
		{
			"name":        "sync",
			"description": "Scan a component's Go/templ source for Translate(\"type\",\"text\") calls and add any missing keys to en.json (value == key). Optionally pass component; otherwise syncs all. Reports keys added.",
			"inputSchema": obj(map[string]any{"component": str}),
		},
		{
			"name":        "check",
			"description": "Report keys used in code but missing from en.json (i.e. sync needed), plus per-language coverage. Read-only.",
			"inputSchema": obj(map[string]any{"component": str}),
		},
	}
}

// =============================================================================
// Tool implementations
// =============================================================================

func (s *server) toolListComponents() (any, error) {
	comps := translations.DiscoverComponents(s.root)
	out := make([]map[string]any, 0, len(comps))
	for _, c := range comps {
		en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
		out = append(out, map[string]any{
			"component": c.ID,
			"name":      c.Name,
			"languages": translations.Languages(c.Dir),
			"en_keys":   countKeys(en),
		})
	}
	return map[string]any{"components": out}, nil
}

func (s *server) toolSummarize(args json.RawMessage) (any, error) {
	var a struct {
		Component string `json:"component"`
	}
	parseArgs(args, &a)

	var out []map[string]any
	for _, c := range s.components(a.Component) {
		en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
		total := countKeys(en)
		var langs []map[string]any
		for _, lang := range translations.Languages(c.Dir) {
			if lang == "en" {
				continue
			}
			cat, _ := translations.ReadCatalog(filepath.Join(c.Dir, lang+".json"))
			tr := translatedCount(en, cat)
			langs = append(langs, map[string]any{
				"lang": lang, "translated": tr, "untranslated": total - tr,
				"percent": pct(tr, total),
			})
		}
		out = append(out, map[string]any{"component": c.ID, "en_keys": total, "languages": langs})
	}
	return map[string]any{"summary": out}, nil
}

func (s *server) toolListKeys(args json.RawMessage) (any, error) {
	var a struct {
		Component, Lang, Msgtype, Search string
		Limit, Offset                    int
	}
	parseArgs(args, &a)
	c, err := s.component(a.Component)
	if err != nil {
		return nil, err
	}
	lang := a.Lang
	if lang == "" {
		lang = "en"
	}
	en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
	cat, _ := translations.ReadCatalog(filepath.Join(c.Dir, lang+".json"))
	search := strings.ToLower(a.Search)

	type entry struct {
		Msgtype, Key, Value string
		Translated          bool
	}
	var entries []entry
	for _, mt := range sortedTypes(en) {
		if a.Msgtype != "" && a.Msgtype != mt {
			continue
		}
		for _, key := range sortedKeys(en[mt]) {
			val, translated := resolve(cat, mt, key)
			if search != "" && !strings.Contains(strings.ToLower(key), search) && !strings.Contains(strings.ToLower(val), search) {
				continue
			}
			entries = append(entries, entry{mt, key, val, translated})
		}
	}

	total := len(entries)
	entries = page(entries, a.Offset, a.Limit)
	items := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		items = append(items, map[string]any{"msgtype": e.Msgtype, "key": e.Key, "value": e.Value, "translated": e.Translated})
	}
	return map[string]any{"component": c.ID, "lang": lang, "total": total, "returned": len(items), "entries": items}, nil
}

func (s *server) toolGetTranslation(args json.RawMessage) (any, error) {
	var a struct{ Component, Msgtype, Key, Lang string }
	parseArgs(args, &a)
	c, err := s.component(a.Component)
	if err != nil {
		return nil, err
	}
	en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
	if _, ok := en[a.Msgtype][a.Key]; !ok {
		return nil, fmt.Errorf("key not found in en.json for msgtype %q: %q", a.Msgtype, a.Key)
	}
	cat, _ := translations.ReadCatalog(filepath.Join(c.Dir, a.Lang+".json"))
	val, translated := resolve(cat, a.Msgtype, a.Key)
	return map[string]any{"component": c.ID, "lang": a.Lang, "msgtype": a.Msgtype, "key": a.Key, "source": a.Key, "value": val, "translated": translated}, nil
}

func (s *server) toolFindUntranslated(args json.RawMessage) (any, error) {
	var a struct {
		Component, Lang string
		Limit           int
	}
	parseArgs(args, &a)
	c, err := s.component(a.Component)
	if err != nil {
		return nil, err
	}
	en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
	cat, _ := translations.ReadCatalog(filepath.Join(c.Dir, a.Lang+".json"))
	var missing []map[string]any
	for _, mt := range sortedTypes(en) {
		for _, key := range sortedKeys(en[mt]) {
			if _, translated := resolve(cat, mt, key); !translated {
				missing = append(missing, map[string]any{"msgtype": mt, "key": key})
			}
		}
	}
	total := len(missing)
	if a.Limit > 0 && a.Limit < total {
		missing = missing[:a.Limit]
	}
	return map[string]any{"component": c.ID, "lang": a.Lang, "untranslated_total": total, "entries": missing}, nil
}

func (s *server) toolSetTranslation(args json.RawMessage) (any, error) {
	var a struct{ Component, Lang, Msgtype, Key, Value string }
	parseArgs(args, &a)
	if err := s.applyOne(a.Component, a.Lang, a.Msgtype, a.Key, a.Value); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "component": a.Component, "lang": a.Lang, "msgtype": a.Msgtype, "key": a.Key}, nil
}

func (s *server) toolSetTranslations(args json.RawMessage) (any, error) {
	var a struct {
		Component, Lang string
		Items           []struct{ Msgtype, Key, Value string }
	}
	parseArgs(args, &a)
	c, err := s.component(a.Component)
	if err != nil {
		return nil, err
	}
	en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
	path := filepath.Join(c.Dir, a.Lang+".json")
	cat, err := translations.ReadCatalog(path)
	if err != nil {
		cat = translations.NewCatalog()
	}
	var applied, skipped int
	var skippedKeys []string
	for _, it := range a.Items {
		if _, ok := en[it.Msgtype][it.Key]; !ok {
			skipped++
			skippedKeys = append(skippedKeys, it.Msgtype+"/"+it.Key)
			continue
		}
		cat.Set(it.Msgtype, it.Key, it.Value)
		applied++
	}
	if a.Lang == "en" {
		return nil, fmt.Errorf("refusing to overwrite en.json (the source registry) via set_translations")
	}
	if err := translations.WriteCatalog(path, cat, true); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "component": c.ID, "lang": a.Lang, "applied": applied, "skipped": skipped, "skipped_keys": skippedKeys}, nil
}

func (s *server) toolSync(args json.RawMessage) (any, error) {
	var a struct {
		Component string `json:"component"`
	}
	parseArgs(args, &a)
	var report []map[string]any
	for _, c := range s.components(a.Component) {
		added, err := syncComponent(c)
		if err != nil {
			return nil, err
		}
		report = append(report, map[string]any{"component": c.ID, "added": len(added), "keys": added})
	}
	return map[string]any{"synced": report}, nil
}

func (s *server) toolCheck(args json.RawMessage) (any, error) {
	var a struct {
		Component string `json:"component"`
	}
	parseArgs(args, &a)
	var report []map[string]any
	for _, c := range s.components(a.Component) {
		report = append(report, s.checkComponent(c))
	}
	return map[string]any{"check": report}, nil
}

// =============================================================================
// Shared logic
// =============================================================================

func (s *server) components(id string) []translations.Component {
	all := translations.DiscoverComponents(s.root)
	if id == "" {
		return all
	}
	for _, c := range all {
		if c.ID == id || c.Name == id {
			return []translations.Component{c}
		}
	}
	return nil
}

func (s *server) component(id string) (translations.Component, error) {
	cs := s.components(id)
	if len(cs) == 0 {
		return translations.Component{}, fmt.Errorf("component not found: %q (use list_components)", id)
	}
	return cs[0], nil
}

func (s *server) applyOne(id, lang, msgtype, key, value string) error {
	if lang == "en" {
		return fmt.Errorf("refusing to overwrite en.json (the source registry)")
	}
	c, err := s.component(id)
	if err != nil {
		return err
	}
	en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
	if _, ok := en[msgtype][key]; !ok {
		return fmt.Errorf("key not in en.json for msgtype %q: %q (run sync for new code strings)", msgtype, key)
	}
	path := filepath.Join(c.Dir, lang+".json")
	cat, err := translations.ReadCatalog(path)
	if err != nil {
		cat = translations.NewCatalog()
	}
	cat.Set(msgtype, key, value)
	return translations.WriteCatalog(path, cat, true)
}

func (s *server) checkComponent(c translations.Component) map[string]any {
	en, _ := translations.ReadCatalog(filepath.Join(c.Dir, "en.json"))
	codeKeys := scanSource(componentBase(c))
	var missing []string
	for k := range codeKeys {
		parts := strings.SplitN(k, "\x00", 2)
		if _, ok := en[parts[0]][parts[1]]; !ok {
			missing = append(missing, parts[0]+"/"+parts[1])
		}
	}
	sort.Strings(missing)

	total := countKeys(en)
	var cov []map[string]any
	for _, lang := range translations.Languages(c.Dir) {
		if lang == "en" {
			continue
		}
		cat, _ := translations.ReadCatalog(filepath.Join(c.Dir, lang+".json"))
		tr := translatedCount(en, cat)
		cov = append(cov, map[string]any{"lang": lang, "percent": pct(tr, total)})
	}
	return map[string]any{"component": c.ID, "missing_in_en": missing, "coverage": cov}
}

// syncComponent scans a component's source and appends any code keys missing from
// en.json (value == key). Other language files are left untouched (sparse).
func syncComponent(c translations.Component) ([]string, error) {
	enPath := filepath.Join(c.Dir, "en.json")
	en, err := translations.ReadCatalog(enPath)
	if err != nil {
		en = translations.NewCatalog()
	}
	var added []string
	for k := range scanSource(componentBase(c)) {
		parts := strings.SplitN(k, "\x00", 2)
		mt, key := parts[0], parts[1]
		if _, ok := en[mt][key]; ok {
			continue
		}
		en.Set(mt, key, key)
		added = append(added, mt+"/"+key)
	}
	sort.Strings(added)
	if len(added) > 0 {
		if err := translations.WriteCatalog(enPath, en, true); err != nil {
			return nil, err
		}
	}
	return added, nil
}

func (s *server) runCheck(minCoverage float64) int {
	comps := translations.DiscoverComponents(s.root)
	failed := false
	for _, c := range comps {
		r := s.checkComponent(c)
		missing, _ := r["missing_in_en"].([]string)
		if len(missing) > 0 {
			failed = true
			fmt.Fprintf(os.Stderr, "%s: %d code key(s) missing from en.json (run sync): %s\n",
				c.ID, len(missing), strings.Join(missing, ", "))
		}
		if minCoverage > 0 {
			for _, cv := range r["coverage"].([]map[string]any) {
				if p, _ := cv["percent"].(float64); p < minCoverage {
					failed = true
					fmt.Fprintf(os.Stderr, "%s: %s coverage %.1f%% < %.1f%%\n", c.ID, cv["lang"], p, minCoverage)
				}
			}
		}
	}
	if failed {
		return 1
	}
	fmt.Printf("translations OK: %d component(s) checked\n", len(comps))
	return 0
}

// =============================================================================
// Source scanning + small helpers
// =============================================================================

// scanSource walks base for .go/.templ files and returns the set of translation
// keys used in code, encoded as "<msgtype>\x00<text>". The resources/translations
// dir and common vendored/build dirs are skipped.
func scanSource(base string) map[string]bool {
	keys := map[string]bool{}
	validType := map[string]bool{}
	for _, t := range translations.MsgTypes {
		validType[t] = true
	}
	filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case "translations", "node_modules", ".git", "db":
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".templ" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range translateCall.FindAllStringSubmatch(string(data), -1) {
			if validType[m[1]] {
				keys[m[1]+"\x00"+m[2]] = true
			}
		}
		return nil
	})
	return keys
}

// componentBase is the plugin/core root (parent of resources/translations).
func componentBase(c translations.Component) string {
	return filepath.Dir(filepath.Dir(c.Dir))
}

// resolve returns the translation for (msgtype,key) and whether it is genuinely
// translated (present, non-empty, and different from the English source key).
func resolve(cat translations.Catalog, msgtype, key string) (string, bool) {
	if v, ok := cat[msgtype][key]; ok && v != "" && v != key {
		return v, true
	}
	return key, false
}

func translatedCount(en, lang translations.Catalog) int {
	n := 0
	for mt, keys := range en {
		for key := range keys {
			if _, translated := resolve(lang, mt, key); translated {
				n++
			}
		}
	}
	return n
}

func countKeys(c translations.Catalog) int {
	n := 0
	for _, m := range c {
		n += len(m)
	}
	return n
}

func sortedTypes(c translations.Catalog) []string {
	var ts []string
	for t := range c {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	return ts
}

func sortedKeys(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func pct(n, total int) float64 {
	if total == 0 {
		return 100
	}
	return float64(n) * 100 / float64(total)
}

func page[T any](items []T, offset, limit int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return nil
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}
