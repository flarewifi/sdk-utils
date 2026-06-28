// Manual software-release uploads.
//
// The manual upload page accepts two kinds of file: a raw OS firmware image
// (.bin/.img, flashed via sysupgrade) and a software-release tarball produced by the
// cloud builder (software-release[-mono]_*.tar.gz — a full, ABI-matched app/core
// bundle). This file holds the CONTENT-based detection that tells the two apart and
// classifies a release's build flavor, plus a helper to spool an upload to disk so it
// can be inspected and extracted. The build-tagged StageLocalSoftwareRelease (in
// software_release_local{,_mono}.go) does the actual apply.
package updates

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	// MaxSoftwareReleaseFileSize caps a manually uploaded software-release tarball. A
	// release bundles the core binary, compiled plugins, and local plugin sources, so
	// it is far larger than a firmware image — allow up to 300 MB.
	MaxSoftwareReleaseFileSize = 300 << 20

	// corePackage is the package id stamped in a release's core/plugin.json. Its
	// presence (with start.sh) is what identifies a tarball as a flarewifi release.
	corePackage = "com.flarego.core"

	// stagedCompleteMarkerName is the marker only the NON-mono start.sh writes/reads;
	// finding it in a release's bundled start.sh classifies that release as non-mono.
	stagedCompleteMarkerName = ".staged_complete"
)

var (
	// ErrNotSoftwareRelease means a gzip/tar upload lacks the core files that
	// identify a flarewifi software release.
	ErrNotSoftwareRelease = errors.New("not a valid software release archive")

	// ErrReleaseBuildMismatch means the release's build flavor (mono/non-mono) does
	// not match this machine's, so applying it would brick the device.
	ErrReleaseBuildMismatch = errors.New("software release build type does not match this machine")
)

// ReleaseInfo describes a software-release tarball, gathered by inspecting a handful
// of metadata files inside it. The zero value means "not a recognized release".
type ReleaseInfo struct {
	// IsRelease is true when the tarball carries both start.sh and a core/plugin.json
	// whose package is com.flarego.core — the two files every release ships.
	IsRelease bool
	// IsMono reports whether the bundled start.sh is the mono (whole-app swap) variant
	// rather than the non-mono (staged per-package overlay) variant.
	IsMono bool
	// CoreVersion is core/plugin.json "version" (the ABI identity).
	CoreVersion string
	// ProductVersion is core/product.json "version" (the per-partner release lineage).
	ProductVersion string
}

// GetAcceptedUploadExtensions returns the file extensions the manual upload control
// advertises (HTML accept + help text). It is intentionally broader than
// GetAllowedExtensions (the sysupgrade-only validation set): the real routing is
// content-based (see IsGzip/InspectRelease), so this only nudges the file picker.
// ".gz"/".tgz" cover software-release ".tar.gz" tarballs.
func GetAcceptedUploadExtensions() []string {
	return []string{".bin", ".img", ".gz", ".tgz"}
}

// IsGzip reports whether r starts with the gzip magic bytes (0x1f 0x8b), rewinding r
// to the start before returning so the caller can re-read the whole stream. A
// software-release tarball is always gzip-compressed; a raw firmware image is not, so
// this is the cheap first-pass discriminator (O(2 bytes), no decompression).
func IsGzip(r io.ReadSeeker) (bool, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	defer r.Seek(0, io.SeekStart)

	magic := make([]byte, 2)
	if _, err := io.ReadFull(r, magic); err != nil {
		// Too short to be a gzip stream — treat as not-gzip (a firmware image).
		return false, nil
	}
	return magic[0] == 0x1f && magic[1] == 0x8b, nil
}

// InspectRelease opens a gzip tarball at tarballPath and reads the few metadata files
// that identify and classify a flarewifi software release: start.sh (presence +
// mono/non-mono flavor) and core/{plugin,product}.json (versions). Reading stops once
// all the files we care about are seen, so it usually scans only the head of the
// archive. A non-gzip or unreadable file yields a zero ReleaseInfo (IsRelease=false).
func InspectRelease(tarballPath string) (ReleaseInfo, error) {
	var info ReleaseInfo

	f, err := os.Open(tarballPath)
	if err != nil {
		return info, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		// Not a gzip stream — definitively not a release tarball.
		return info, nil
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var sawStart, sawCore, sawProduct bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return info, err
		}

		// Tar entries are root-relative (the builder archives with filepath.Rel);
		// normalize a stray "./" prefix defensively.
		name := path.Clean(strings.TrimPrefix(hdr.Name, "./"))
		switch name {
		case "start.sh":
			sawStart = true
			data, _ := io.ReadAll(io.LimitReader(tr, 64<<10))
			// The non-mono start.sh stages per-package dirs and commits with a
			// .staged_complete marker; the mono start.sh swaps the whole app and keeps
			// a backup.tar.gz. The non-mono-only marker is the reliable discriminator.
			info.IsMono = !strings.Contains(string(data), stagedCompleteMarkerName)
		case "core/plugin.json":
			var meta struct {
				Package string `json:"package"`
				Version string `json:"version"`
			}
			if data, err := io.ReadAll(io.LimitReader(tr, 1<<20)); err == nil {
				if json.Unmarshal(data, &meta) == nil && meta.Package == corePackage {
					sawCore = true
					info.CoreVersion = meta.Version
				}
			}
		case "core/product.json":
			var meta struct {
				Version string `json:"version"`
			}
			if data, err := io.ReadAll(io.LimitReader(tr, 1<<20)); err == nil {
				_ = json.Unmarshal(data, &meta)
				info.ProductVersion = meta.Version
				sawProduct = true
			}
		}

		if sawStart && sawCore && sawProduct {
			break // everything we inspect is in hand; skip the rest of the archive
		}
	}

	info.IsRelease = sawStart && sawCore
	return info, nil
}

// SaveUploadToTemp spools an upload stream to a private temp file and returns its
// path. The caller owns the file and must remove it. Used for software-release
// uploads, which must land on disk to be inspected and extracted (firmware images
// stream straight to the sysupgrade path and never come here).
func SaveUploadToTemp(src io.Reader) (string, error) {
	if err := sdkutils.FsEnsureDir(sdkutils.PathTmpDir); err != nil {
		return "", err
	}

	tmpPath := filepath.Join(sdkutils.PathTmpDir, "upload-"+sdkutils.RandomStr(8)+".tar.gz")
	dst, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return tmpPath, nil
}
