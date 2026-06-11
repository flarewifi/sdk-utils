package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func findResolved(deps []LockedGoModule, path string) (LockedGoModule, bool) {
	for _, d := range deps {
		if d.Path == path {
			return d, true
		}
	}
	return LockedGoModule{}, false
}

func TestResolvedGoModules(t *testing.T) {
	dir := t.TempDir()

	// external/pkg: a plain external dependency (recorded).
	// github.com/flarehotspot/sdk-utils: workspace-local (no go.sum entry, excluded).
	// down/pinned: required at v1.1 but replaced down to v1.0 (recorded at v1.0).
	// local/fork: filesystem-replaced (excluded).
	writeFixture(t, dir, "go.mod", `module example.com/plugin

go 1.21

require (
	external/pkg v1.2.3
	github.com/flarehotspot/sdk-utils v0.1.22 // indirect
	down/pinned v1.1.0
	local/fork v0.0.0
)

replace down/pinned => down/pinned v1.0.0

replace local/fork => ../fork
`)

	writeFixture(t, dir, "go.sum", `external/pkg v1.2.3 h1:zipEXTERNAL=
external/pkg v1.2.3/go.mod h1:modEXTERNAL=
down/pinned v1.0.0 h1:zipPINNED10=
down/pinned v1.0.0/go.mod h1:modPINNED10=
down/pinned v1.1.0/go.mod h1:modPINNED11=
`)

	deps, err := ResolvedGoModules(dir)
	if err != nil {
		t.Fatalf("ResolvedGoModules: %v", err)
	}

	if ext, ok := findResolved(deps, "external/pkg"); !ok {
		t.Errorf("external/pkg should be recorded")
	} else if ext.Version != "v1.2.3" || ext.Hash != "h1:zipEXTERNAL=" || ext.GoModHash != "h1:modEXTERNAL=" {
		t.Errorf("external/pkg recorded wrong: %+v", ext)
	}

	if _, ok := findResolved(deps, "github.com/flarehotspot/sdk-utils"); ok {
		t.Errorf("workspace-local sdk-utils (no go.sum) must be excluded")
	}

	// The version-replaced module must be recorded at the EFFECTIVE (replaced)
	// version + its hash, not the higher require version.
	if pin, ok := findResolved(deps, "down/pinned"); !ok {
		t.Errorf("down/pinned should be recorded")
	} else if pin.Version != "v1.0.0" || pin.Hash != "h1:zipPINNED10=" {
		t.Errorf("down/pinned must record the replaced version v1.0.0, got %+v", pin)
	}

	if _, ok := findResolved(deps, "local/fork"); ok {
		t.Errorf("filesystem-replaced local/fork must be excluded")
	}
}

func TestVerifyPinnedGoSum(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "go.sum", `external/pkg v1.2.3 h1:REAL=
external/pkg v1.2.3/go.mod h1:REALMOD=
`)

	// Matching hash passes; an unused pinned module (absent from go.sum) is skipped.
	ok := []LockedGoModule{
		{Path: "external/pkg", Version: "v1.2.3", Hash: "h1:REAL=", GoModHash: "h1:REALMOD="},
		{Path: "unused/pkg", Version: "v9.9.9", Hash: "h1:WHATEVER="},
	}
	if err := VerifyPinnedGoSum(dir, ok); err != nil {
		t.Errorf("expected pass, got %v", err)
	}

	// A drifted hash (same version label, different content) must fail.
	drift := []LockedGoModule{{Path: "external/pkg", Version: "v1.2.3", Hash: "h1:DIFFERENT="}}
	if err := VerifyPinnedGoSum(dir, drift); err == nil {
		t.Errorf("expected hash-mismatch error, got nil")
	}
}
