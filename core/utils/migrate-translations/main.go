// Command migrate-translations folds the legacy one-file-per-string translation
// tree into per-language JSON catalogs (resources/translations/<lang>.json).
//
// For each language file, the English KEY is the CONTENT of its sibling file at
// the same relative path under en/ (filenames are identical across languages), and
// the file's own content is the translation VALUE. en.json stores every key
// (value == key, the registry); other languages store only real translations
// (untranslated files, where content == English, are omitted — runtime falls back
// to the English source). The conversion is verified losslessly in memory against
// runtime resolution semantics BEFORE any legacy directory is deleted.
//
// It is idempotent (skips a component whose en.json already exists and whose en/
// directory is gone) and runs per-repo, because each component is self-contained.
//
// Usage:
//
//	go run -C core ./utils/migrate-translations               # core + plugins
//	go run -C core ./utils/migrate-translations -dry-run -v
//	go run -C core ./utils/migrate-translations -base-dir .    # one plugin repo root
//	go run -C core ./utils/migrate-translations -keep-legacy   # write JSON, keep dirs
//
// A "base dir" contains resources/translations (e.g. "core", "data/plugins/local/<pkg>").
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"core/utils/translations"
)

// hashSuffix matches the "~<16 hex>" tail of a canonical (hashed) translation
// filename, used to recover the human-readable English key from a legacy file
// whose tree was only partially migrated to the hashed scheme.
var hashSuffix = regexp.MustCompile(`~[0-9a-f]{16}$`)

type stats struct {
	components int
	catalogs   int // <lang>.json files written
	keys       int // en.json entries (the registry size)
	orphans    int // language files with no en sibling (unrecoverable key)
	skipped    int // components already migrated
}

func main() {
	var baseDirsCSV string
	var dryRun, verbose, keepLegacy bool
	flag.StringVar(&baseDirsCSV, "base-dir", "", "Comma-separated dirs containing resources/translations (default: core + plugins/installed/* + data/plugins/local/* + data/plugins/system/*)")
	flag.BoolVar(&dryRun, "dry-run", false, "Report what would change without writing or deleting")
	flag.BoolVar(&verbose, "v", false, "Verbose: print per-component detail")
	flag.BoolVar(&keepLegacy, "keep-legacy", false, "Write <lang>.json but do NOT delete the legacy per-language directories")
	flag.Parse()

	bases := resolveBaseDirs(baseDirsCSV)
	if len(bases) == 0 {
		fmt.Fprintln(os.Stderr, "no base dirs with resources/translations found")
		os.Exit(1)
	}

	var total stats
	failed := false
	for _, base := range bases {
		s, err := migrateBase(base, dryRun, verbose, keepLegacy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", base, err)
			failed = true
			continue
		}
		total.components += s.components
		total.catalogs += s.catalogs
		total.keys += s.keys
		total.orphans += s.orphans
		total.skipped += s.skipped
		fmt.Printf("%-50s catalogs=%d keys=%d orphans=%d skipped=%t\n",
			base, s.catalogs, s.keys, s.orphans, s.skipped > 0)
	}

	mode := "migrated"
	if dryRun {
		mode = "would migrate"
	}
	fmt.Printf("\nTotal: %s %d component(s), %d catalog file(s), %d key(s), %d orphan(s)\n",
		mode, total.components, total.catalogs, total.keys, total.orphans)
	if failed {
		os.Exit(2)
	}
}

// =============================================================================
// HELPERS
// =============================================================================

func resolveBaseDirs(csv string) []string {
	var candidates []string
	if csv != "" {
		for _, b := range strings.Split(csv, ",") {
			if b = strings.TrimSpace(b); b != "" {
				candidates = append(candidates, b)
			}
		}
	} else {
		candidates = append(candidates, "core")
		for _, glob := range []string{"plugins/installed/*", "data/plugins/local/*", "data/plugins/devel/*", "data/plugins/system/*"} {
			matches, _ := filepath.Glob(glob)
			candidates = append(candidates, matches...)
		}
	}

	var bases []string
	for _, c := range candidates {
		if isDir(filepath.Join(c, "resources", "translations")) {
			bases = append(bases, c)
		}
	}
	return bases
}

func migrateBase(base string, dryRun, verbose, keepLegacy bool) (stats, error) {
	var s stats
	root := filepath.Join(base, "resources", "translations")
	enDir := filepath.Join(root, "en")
	enJSON := filepath.Join(root, "en.json")

	// Idempotent: already migrated when en.json exists and the en/ dir is gone.
	if pathExists(enJSON) && !isDir(enDir) {
		s.skipped = 1
		if verbose {
			fmt.Printf("  %s: already migrated, skipping\n", base)
		}
		return s, nil
	}
	if !isDir(enDir) {
		return s, fmt.Errorf("no en/ directory under %s; cannot derive English keys", root)
	}
	s.components = 1

	// English key maps. enKeys: relpath "<type>/<file>" -> English source text
	// (content), the fast path for hashed-consistent trees. enTextByType: every
	// English text per msgtype, used to recover a key from a partially-migrated
	// language file that still carries the old human-readable filename.
	enKeys := map[string]string{}
	enTextByType := map[string]map[string]bool{}
	if err := walkFiles(enDir, func(rel, content string) {
		enKeys[rel] = content
		msgType := strings.SplitN(rel, string(filepath.Separator), 2)[0]
		if enTextByType[msgType] == nil {
			enTextByType[msgType] = map[string]bool{}
		}
		enTextByType[msgType][content] = true
	}); err != nil {
		return s, err
	}

	langDirs := listLangDirs(root)

	// Recovery pre-pass: some language trees are only partially migrated and still
	// carry old human-readable filenames for keys absent from this component's en/
	// (e.g. fr/label/Cancel="Annuler" with no en/label/Cancel). The filename IS the
	// English key; register it so the build below attaches the translation and en.json
	// gains the key. The translator's code-scan --prune later drops any truly unused.
	recoveredEn := map[string]map[string]bool{} // msgType -> English text
	for _, lang := range langDirs {
		if lang == "en" {
			continue
		}
		walkFiles(filepath.Join(root, lang), func(rel, content string) {
			if _, ok := enKeys[rel]; ok {
				return
			}
			msgType := strings.SplitN(rel, string(filepath.Separator), 2)[0]
			cand := hashSuffix.ReplaceAllString(filepath.Base(rel), "")
			if cand == "" || enTextByType[msgType][cand] {
				return
			}
			if recoveredEn[msgType] == nil {
				recoveredEn[msgType] = map[string]bool{}
			}
			recoveredEn[msgType][cand] = true
			if enTextByType[msgType] == nil {
				enTextByType[msgType] = map[string]bool{}
			}
			enTextByType[msgType][cand] = true // so resolveEnglishKey finds it
		})
	}

	catalogs := map[string]translations.Catalog{}
	for _, lang := range langDirs {
		cat := translations.NewCatalog()
		langDir := filepath.Join(root, lang)
		err := walkFiles(langDir, func(rel, content string) {
			msgType := strings.SplitN(rel, string(filepath.Separator), 2)[0]
			enKey, ok := resolveEnglishKey(rel, msgType, enKeys, enTextByType)
			if !ok {
				s.orphans++
				fmt.Fprintf(os.Stderr, "ORPHAN %s/%s/%s: no en sibling (key unrecoverable), skipping\n", base, lang, rel)
				return
			}
			if lang == "en" {
				cat.Set(msgType, enKey, enKey) // registry: value == key
			} else if content != enKey { // omit untranslated; runtime falls back
				cat.Set(msgType, enKey, content)
			}
		})
		if err != nil {
			return s, err
		}
		catalogs[lang] = cat
	}

	// Fold recovered keys into en.json (value == key) so the registry is complete.
	for msgType, set := range recoveredEn {
		for text := range set {
			catalogs["en"].Set(msgType, text, text)
		}
	}

	// Verify losslessly against runtime resolution BEFORE deleting anything.
	if err := verifyLossless(root, langDirs, catalogs, enKeys, enTextByType); err != nil {
		return s, fmt.Errorf("losslessness check failed (no files changed): %w", err)
	}

	s.keys = countKeys(catalogs["en"])
	if verbose {
		fmt.Printf("  %s: %d languages, %d en keys\n", base, len(langDirs), s.keys)
	}

	if dryRun {
		s.catalogs = len(catalogs)
		return s, nil
	}

	// Write <lang>.json (pretty source form).
	for _, lang := range langDirs {
		out := filepath.Join(root, lang+".json")
		if err := translations.WriteCatalog(out, catalogs[lang], true); err != nil {
			return s, fmt.Errorf("write %s: %w", out, err)
		}
		s.catalogs++
	}

	if !keepLegacy {
		for _, lang := range langDirs {
			if err := os.RemoveAll(filepath.Join(root, lang)); err != nil {
				return s, fmt.Errorf("remove legacy dir %s: %w", lang, err)
			}
		}
		removeTarballs(root)
	}

	return s, nil
}

// verifyLossless asserts that, for every legacy language file, resolving its
// English key against the new catalog reproduces the file's content exactly —
// modelling runtime semantics (present key -> value; absent key -> English source).
func verifyLossless(root string, langDirs []string, catalogs map[string]translations.Catalog, enKeys map[string]string, enTextByType map[string]map[string]bool) error {
	for _, lang := range langDirs {
		cat := catalogs[lang]
		langDir := filepath.Join(root, lang)
		var verr error
		err := walkFiles(langDir, func(rel, content string) {
			if verr != nil {
				return
			}
			msgType := strings.SplitN(rel, string(filepath.Separator), 2)[0]
			enKey, ok := resolveEnglishKey(rel, msgType, enKeys, enTextByType)
			if !ok {
				return // orphan already reported; nothing to verify
			}
			got, present := cat[msgType][enKey]
			if !present {
				got = enKey // runtime fallback to English source
			}
			if got != content {
				verr = fmt.Errorf("%s/%s: key %q resolves to %q, want %q", lang, rel, enKey, got, content)
			}
		})
		if err != nil {
			return err
		}
		if verr != nil {
			return verr
		}
	}
	return nil
}

// resolveEnglishKey returns the English source text (catalog key) for a language
// file at relpath rel. Fast path: an en sibling at the same relpath (hashed-
// consistent trees). Recovery path: a partially-migrated file still carrying the
// old human-readable filename — its key is the filename (sans any "~<hash>" tail),
// accepted only if that exact English text exists in en under the same msgtype.
func resolveEnglishKey(rel, msgType string, enKeys map[string]string, enTextByType map[string]map[string]bool) (string, bool) {
	if enKey, ok := enKeys[rel]; ok {
		return enKey, true
	}
	cand := hashSuffix.ReplaceAllString(filepath.Base(rel), "")
	if enTextByType[msgType][cand] {
		return cand, true
	}
	return "", false
}

// walkFiles invokes fn for every regular file under dir with its slash-trimmed
// content and its relative path ("<type>/<file>"). Files directly under dir (no
// type segment) are ignored.
func walkFiles(dir string, fn func(rel, content string)) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		if !strings.Contains(rel, string(filepath.Separator)) {
			return nil // needs at least <type>/<file>
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fn(rel, strings.TrimSpace(string(data)))
		return nil
	})
}

func listLangDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var langs []string
	for _, e := range entries {
		if e.IsDir() {
			langs = append(langs, e.Name())
		}
	}
	sort.Strings(langs)
	return langs
}

func countKeys(c translations.Catalog) int {
	n := 0
	for _, m := range c {
		n += len(m)
	}
	return n
}

func removeTarballs(root string) {
	matches, _ := filepath.Glob(filepath.Join(root, "*.tar.gz"))
	for _, m := range matches {
		os.Remove(m)
	}
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
