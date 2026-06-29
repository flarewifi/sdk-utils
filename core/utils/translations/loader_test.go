package translations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogRoundTripPretty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "es.json")

	cat := NewCatalog()
	cat.Set("error", "Device is blocked", "El dispositivo está bloqueado")
	cat.Set("label", "Save & Continue", "Guardar y continuar") // '&' must stay literal

	if err := WriteCatalog(path, cat, true); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	// Escaping disabled: the literal "Save & Continue" must survive verbatim.
	// HTML-escaping would emit "Save & Continue" instead.
	if !strings.Contains(string(raw), "Save & Continue") {
		t.Errorf("literal '&' should be preserved unescaped (no HTML escaping): %s", raw)
	}
	if !strings.Contains(string(raw), "\n  ") {
		t.Errorf("pretty form should be indented, got: %s", raw)
	}

	got, err := ReadCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["error"]["Device is blocked"] != "El dispositivo está bloqueado" {
		t.Errorf("round trip lost value: %#v", got["error"])
	}
}

func TestWriteCatalogCompact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fr.json")
	cat := NewCatalog()
	cat.Set("label", "Hello", "Bonjour")
	if err := WriteCatalog(path, cat, false); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	// Compact: a single content line (plus the encoder's trailing newline), no indent.
	if strings.Contains(string(raw), "\n  ") {
		t.Errorf("compact form should not be indented: %q", raw)
	}
	if n := strings.Count(strings.TrimSpace(string(raw)), "\n"); n != 0 {
		t.Errorf("compact form should be one line, got %d newlines: %q", n, raw)
	}
}

func TestInterpolate(t *testing.T) {
	cases := []struct {
		name, resolved, fallback, want string
		pairs                          []any
	}{
		{"static fast-path", "Just text", "key", "Just text", nil},
		{"trim", "  padded  ", "key", "padded", nil},
		{"template", "Hello <% .name %>", "key", "Hello World", []any{"name", "World"}},
		{"bad template falls back", "Hello <% .name", "FALLBACK", "FALLBACK", []any{"name", "x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := interpolate(c.resolved, c.fallback, c.pairs); got != c.want {
				t.Errorf("interpolate(%q) = %q, want %q", c.resolved, got, c.want)
			}
		})
	}
}

func TestLoadCatalogMissingIsEmpty(t *testing.T) {
	dir := t.TempDir() // no es.json present
	cat := loadCatalog(dir, "es")
	if cat == nil {
		t.Fatal("loadCatalog returned nil; want empty catalog")
	}
	if len(cat["error"]) != 0 {
		t.Errorf("missing catalog should be empty, got %#v", cat)
	}
	// A missing-file load must never create the file (no runtime writes).
	if _, err := os.Stat(filepath.Join(dir, "es.json")); !os.IsNotExist(err) {
		t.Error("loadCatalog created a file; it must never write")
	}
}

func TestMinifyAllCatalogs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "comp", "resources", "translations")
	cat := NewCatalog()
	cat.Set("label", "Hello", "Bonjour")
	if err := WriteCatalog(filepath.Join(dir, "fr.json"), cat, true); err != nil { // pretty
		t.Fatal(err)
	}
	// A dot-sidecar must be ignored by minify.
	if err := os.WriteFile(filepath.Join(dir, ".translator-state.json"), []byte("{ \"x\": 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MinifyAllCatalogs(root); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "fr.json"))
	if strings.Contains(string(raw), "\n  ") {
		t.Errorf("fr.json should be minified after MinifyAllCatalogs: %q", raw)
	}
	side, _ := os.ReadFile(filepath.Join(dir, ".translator-state.json"))
	if !strings.Contains(string(side), "  ") && !strings.Contains(string(side), "\n") {
		t.Errorf("sidecar should be untouched, got %q", side)
	}
}
