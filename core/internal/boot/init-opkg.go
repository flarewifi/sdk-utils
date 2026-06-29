package boot

import (
	"core/utils/pkgmgr"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// InitOpkg installs any bundled local package files in ./packages using the
// machine's package manager. opkg images bundle .ipk files there (the custom
// patched golang etc.); apk images bundle none (the build relies on stock feed
// packages, see IsApkOsVersion in the builder), so this is a no-op on apk. Both
// extensions are handled so the right manager installs the right artifact.
func InitOpkg() {
	var files []string

	packagesDir := filepath.Join(sdkutils.PathAppDir, "packages")
	if err := sdkutils.FsListFiles(packagesDir, &files, true); err != nil {
		return
	}

	mgr := pkgmgr.Detect()

	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != ".ipk" && ext != ".apk" {
			continue
		}

		if err := mgr.InstallFiles([]string{f}); err != nil {
			return
		}

		// remove file if installed successfully
		os.Remove(f)
	}
}
