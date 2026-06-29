package translations

import (
	"os"
	"path/filepath"
	"sort"
)

// Component is one translatable unit (core or a plugin) with its catalog dir.
type Component struct {
	ID   string // stable identifier, e.g. "core" or "data/plugins/devel/com.flarego.wifi-hotspot"
	Dir  string // the resources/translations directory holding <lang>.json
	Name string // short display name (basename), e.g. "core" or "com.flarego.wifi-hotspot"
}

// componentRoots are the source-of-truth locations searched under a working root:
// core itself, plus each plugin's editable source repo. Build-output copies under
// plugins/installed/* are intentionally excluded — edits belong in the sources.
var componentRoots = []string{
	"core",
	"data/plugins/devel/*",
	"data/plugins/local/*",
}

// DiscoverComponents returns every component under root that has a migrated
// catalog (an en.json). Results are sorted by ID for stable output.
func DiscoverComponents(root string) []Component {
	var comps []Component
	for _, pattern := range componentRoots {
		matches, _ := filepath.Glob(filepath.Join(root, pattern))
		for _, base := range matches {
			dir := filepath.Join(base, "resources", "translations")
			if !fileExists(filepath.Join(dir, "en.json")) {
				continue
			}
			rel, err := filepath.Rel(root, base)
			if err != nil {
				rel = base
			}
			comps = append(comps, Component{ID: rel, Dir: dir, Name: filepath.Base(base)})
		}
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i].ID < comps[j].ID })
	return comps
}

// Languages returns the language codes (sorted) that have a <lang>.json in dir.
func Languages(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var langs []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".json" || name[0] == '.' {
			continue
		}
		langs = append(langs, name[:len(name)-len(".json")])
	}
	sort.Strings(langs)
	return langs
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
