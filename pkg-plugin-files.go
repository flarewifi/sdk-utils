package sdkutils

import (
	"path/filepath"
)

// PluginFiles is the file set staged under plugins/installed for a NON-MONO
// plugin install — one whose Go code is compiled into its own plugin.so and
// loaded via plugin.Open at runtime. It carries the standalone plugin.so and the
// Go build inputs (go.mod/go.sum) alongside the runtime resources.
var PluginFiles = []string{
	"go.mod",
	"go.sum",
	"LICENSE.txt",
	"plugin.json",
	"plugin.so",
	"resources/assets/dist",
	"resources/assets/public",
	"resources/migrations",
	"resources/translations",
}

// PluginFilesMono is the file set staged for a plugin whose Go code is BUNDLED
// into a binary rather than loaded from a standalone plugin.so: a mono build
// (everything in one binary) or a system plugin statically linked into
// core/plugin.so. No plugin.so exists and the Go build inputs (go.mod/go.sum)
// are not needed on the device, so it carries LICENSE.txt (kept for
// distribution), plugin.json (which the loader reads for PluginInfo) and the
// runtime resource trees.
var PluginFilesMono = []string{
	"LICENSE.txt",
	"plugin.json",
	"resources/assets/dist",
	"resources/assets/public",
	"resources/migrations",
	"resources/translations",
}

func CopyPluginFiles(pluginSrc string, dest string) error {
	return copyPluginFileList(pluginSrc, dest, PluginFiles)
}

func CopyPluginFilesMono(pluginSrc string, dest string) error {
	return copyPluginFileList(pluginSrc, dest, PluginFilesMono)
}

// copyPluginFileList copies the named entries from pluginSrc into dest, skipping
// any entry missing from the source. Entries are best-effort: go.sum and the
// resources/* trees may be absent, and plugin.so never exists for a bundled
// (mono / core-linked) plugin.
func copyPluginFileList(pluginSrc string, dest string, files []string) error {
	if err := FsEnsureDir(dest); err != nil {
		return err
	}

	for _, f := range files {
		src := filepath.Join(pluginSrc, f)
		if !FsExists(src) {
			continue
		}
		if err := FsCopy(src, filepath.Join(dest, f)); err != nil {
			return err
		}
	}
	return nil
}
