// Package translations is the runtime + tooling home for the per-language JSON
// translation catalogs that replaced the legacy one-file-per-string tree.
//
// On-disk format (one file per component per language):
//
//	resources/translations/<lang>.json
//	{ "<msgtype>": { "<English source text>": "<translation>" }, ... }
//
// The English source text is the lookup key; en.json is the registry (value ==
// key for every entry) and its existence marks a component as migrated. An absent
// key resolves to the English source text itself, so non-English catalogs are
// legitimately sparse (untranslated strings are simply omitted, never copied as
// English). catalog_io.go owns the on-disk ENCODING (pretty for committed source,
// compact for the production build artifact); loader.go owns reading + lookup.
package translations

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// MsgTypes are the six translation categories, also the top-level keys of every
// catalog. A NewCatalog seeds all six so committed files are consistent even when
// a component uses only a couple of them.
var MsgTypes = []string{"label", "error", "success", "info", "warning", "type"}

// Catalog maps msgtype -> source text -> translation.
type Catalog map[string]map[string]string

// NewCatalog returns an empty catalog with all six msgtype buckets present.
func NewCatalog() Catalog {
	c := make(Catalog, len(MsgTypes))
	for _, t := range MsgTypes {
		c[t] = map[string]string{}
	}
	return c
}

// Set assigns a translation, lazily creating the msgtype bucket so unexpected
// (non-standard) types never panic on a nil map.
func (c Catalog) Set(msgtype, key, value string) {
	if c[msgtype] == nil {
		c[msgtype] = map[string]string{}
	}
	c[msgtype][key] = value
}

// ReadCatalog parses a <lang>.json catalog. A missing file is reported via the
// returned error (callers that treat absence as "empty" should check os.IsNotExist).
func ReadCatalog(path string) (Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := Catalog{}
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return c, nil
}

// WriteCatalog encodes a catalog to disk. pretty=true gives the 2-space-indented,
// human-reviewable source form (used by the tools); pretty=false gives the compact
// single-line form (used by the production build). Both disable HTML escaping so
// values containing <% %>, <, >, & stay literal, and encoding/json sorts map keys
// so diffs are byte-stable.
func WriteCatalog(path string, c Catalog, pretty bool) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(c); err != nil { // Encode appends a trailing newline
		return err
	}
	return sdkutils.FsWriteFile(path, buf.Bytes())
}

// MinifyAllCatalogs rewrites every resources/translations/<lang>.json under root
// into the compact form. It is the production build step that replaces the old
// per-language .tar.gz compression: per-language JSON is tiny, so minification is
// all the size reduction the device needs. Dot-prefixed sidecars (e.g.
// .translator-state.json) are skipped.
func MinifyAllCatalogs(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".json" {
			return nil
		}
		if strings.HasPrefix(filepath.Base(p), ".") {
			return nil
		}
		if filepath.Base(filepath.Dir(p)) != "translations" ||
			filepath.Base(filepath.Dir(filepath.Dir(p))) != "resources" {
			return nil
		}
		c, err := ReadCatalog(p)
		if err != nil {
			return err
		}
		return WriteCatalog(p, c, false)
	})
}
