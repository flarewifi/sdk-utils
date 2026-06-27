package updates

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// writeTarGz builds a gzip tarball at path from name→content entries. Entry names
// are root-relative, matching what the cloud builder's CompressTar produces.
func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatalf("write header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write body %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
}

// nonMonoStartScript / monoStartScript model the discriminating bit: only the
// non-mono start.sh references the .staged_complete marker.
const (
	nonMonoStartScript = "#!/bin/sh\n# Non-mono boot + staged-update applier.\nSTAGED_COMPLETE_MARKER=\"$SOFTWARE_UPDATE_DIR/.staged_complete\"\n"
	monoStartScript    = "#!/bin/sh\n# Mono boot.\nDOWNLOAD_COMPLETE_MARKER=\"$SOFTWARE_UPDATE_DIR/.dl_software_update_complete\"\nrevert_updates() { tar -xzf $BACKUP_DIR/backup.tar.gz -C $APP_DIR; }\n"
	corePluginJSON     = `{"name":"Core System","package":"com.flarego.core","version":"1.1.23"}`
	productJSON        = `{"version":"2.4.0"}`
)

func TestInspectRelease(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name        string
		files       map[string]string
		wantRelease bool
		wantMono    bool
		wantCoreVer string
		wantProdVer string
	}{
		{
			name: "non-mono release",
			files: map[string]string{
				"start.sh":          nonMonoStartScript,
				"core/plugin.json":  corePluginJSON,
				"core/product.json": productJSON,
				"bin/flare":         "ELF...",
			},
			wantRelease: true,
			wantMono:    false,
			wantCoreVer: "1.1.23",
			wantProdVer: "2.4.0",
		},
		{
			name: "mono release",
			files: map[string]string{
				"start.sh":          monoStartScript,
				"core/plugin.json":  corePluginJSON,
				"core/product.json": productJSON,
				"bin/flare":         "ELF...",
			},
			wantRelease: true,
			wantMono:    true,
			wantCoreVer: "1.1.23",
			wantProdVer: "2.4.0",
		},
		{
			name: "gzip tar but missing start.sh",
			files: map[string]string{
				"core/plugin.json": corePluginJSON,
			},
			wantRelease: false,
		},
		{
			name: "gzip tar with wrong core package",
			files: map[string]string{
				"start.sh":         nonMonoStartScript,
				"core/plugin.json": `{"package":"com.flarego.something-else","version":"1.0.0"}`,
			},
			wantRelease: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, "rel.tar.gz")
			writeTarGz(t, path, tc.files)

			info, err := InspectRelease(path)
			if err != nil {
				t.Fatalf("InspectRelease error: %v", err)
			}
			if info.IsRelease != tc.wantRelease {
				t.Errorf("IsRelease = %v, want %v", info.IsRelease, tc.wantRelease)
			}
			if !tc.wantRelease {
				return
			}
			if info.IsMono != tc.wantMono {
				t.Errorf("IsMono = %v, want %v", info.IsMono, tc.wantMono)
			}
			if info.CoreVersion != tc.wantCoreVer {
				t.Errorf("CoreVersion = %q, want %q", info.CoreVersion, tc.wantCoreVer)
			}
			if info.ProductVersion != tc.wantProdVer {
				t.Errorf("ProductVersion = %q, want %q", info.ProductVersion, tc.wantProdVer)
			}
		})
	}
}

// TestInspectReleaseOnNonGzip ensures a raw (non-gzip) firmware-like file is never
// mistaken for a software release.
func TestInspectReleaseOnNonGzip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firmware.bin")
	if err := os.WriteFile(path, []byte("\x00not a gzip stream\x00"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := InspectRelease(path)
	if err != nil {
		t.Fatalf("InspectRelease error: %v", err)
	}
	if info.IsRelease {
		t.Errorf("a non-gzip file was misdetected as a software release")
	}
}

func TestIsGzip(t *testing.T) {
	// gzip stream
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write([]byte("hello"))
	gw.Close()

	gzReader := bytes.NewReader(gzBuf.Bytes())
	ok, err := IsGzip(gzReader)
	if err != nil {
		t.Fatalf("IsGzip(gzip) error: %v", err)
	}
	if !ok {
		t.Errorf("IsGzip(gzip) = false, want true")
	}
	// Must be rewound to the start for the caller to re-read.
	if pos, _ := gzReader.Seek(0, 1); pos != 0 {
		t.Errorf("IsGzip did not rewind: pos = %d, want 0", pos)
	}

	// plain (firmware-like) bytes
	plain := bytes.NewReader([]byte{0x00, 0x01, 0x02, 0x03})
	ok, err = IsGzip(plain)
	if err != nil {
		t.Fatalf("IsGzip(plain) error: %v", err)
	}
	if ok {
		t.Errorf("IsGzip(plain) = true, want false")
	}
}
