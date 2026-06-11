package plugins

import "testing"

// TestLocalPluginNeedsRepin covers the store-install conflict detector: a local
// plugin must be recompiled when it shares a module with the lock at a different
// version or content hash, and must NOT be touched otherwise.
func TestLocalPluginNeedsRepin(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "go.mod", `module example.com/plugin

go 1.21

require external/pkg v1.2.3
`)
	writeFixture(t, dir, "go.sum", `external/pkg v1.2.3 h1:zipEXTERNAL=
external/pkg v1.2.3/go.mod h1:modEXTERNAL=
`)
	// ResolvedGoModules(dir) => external/pkg v1.2.3, hash h1:zipEXTERNAL=

	cases := []struct {
		name string
		lock []LockedGoModule
		want bool
	}{
		{
			name: "agreeing lock",
			lock: []LockedGoModule{{Path: "external/pkg", Version: "v1.2.3", Hash: "h1:zipEXTERNAL=", GoModHash: "h1:modEXTERNAL="}},
			want: false,
		},
		{
			name: "different locked version",
			lock: []LockedGoModule{{Path: "external/pkg", Version: "v2.0.0", Hash: "h1:newZIP=", GoModHash: "h1:newMOD="}},
			want: true,
		},
		{
			name: "same version, different hash (moved tag)",
			lock: []LockedGoModule{{Path: "external/pkg", Version: "v1.2.3", Hash: "h1:MOVEDZIP=", GoModHash: "h1:modEXTERNAL="}},
			want: true,
		},
		{
			name: "module the plugin does not use",
			lock: []LockedGoModule{{Path: "other/mod", Version: "v9.9.9", Hash: "h1:x=", GoModHash: "h1:y="}},
			want: false,
		},
		{
			name: "empty lock",
			lock: nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LocalPluginNeedsRepin(dir, tc.lock)
			if err != nil {
				t.Fatalf("LocalPluginNeedsRepin: %v", err)
			}
			if got != tc.want {
				t.Fatalf("LocalPluginNeedsRepin = %v, want %v", got, tc.want)
			}
		})
	}
}
