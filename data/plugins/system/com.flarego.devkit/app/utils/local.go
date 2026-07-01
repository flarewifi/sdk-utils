package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// requiredSrcFiles are the files a directory must contain to be a valid plugin
// source. They mirror the core's boot-time scanner (plugins.ValidateSrcPath): a
// plugin missing any of these is silently skipped, so we reject such uploads up
// front instead of letting them rot in the local directory.
var requiredSrcFiles = []string{"plugin.json", "go.mod", "main.go", "LICENSE.txt"}

// LocalPlugin is a lightweight view of one plugin found under data/plugins/local/.
type LocalPlugin struct {
	Package     string
	Name        string
	Description string
	Version     string
}

// ListLocalPlugins reads every plugin directory under data/plugins/local/ and
// returns the ones carrying a readable plugin.json, sorted by display name. A
// directory whose plugin.json is missing or unreadable is skipped rather than
// failing the whole listing.
func ListLocalPlugins() ([]LocalPlugin, error) {
	if !sdkutils.FsExists(sdkutils.PathPluginLocalDir) {
		return []LocalPlugin{}, nil
	}

	var dirs []string
	if err := sdkutils.FsListDirs(sdkutils.PathPluginLocalDir, &dirs, false); err != nil {
		return nil, err
	}

	plugins := make([]LocalPlugin, 0, len(dirs))
	for _, dir := range dirs {
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil || info.Package == "" {
			continue
		}
		plugins = append(plugins, LocalPlugin{
			Package:     info.Package,
			Name:        info.Name,
			Description: info.Description,
			Version:     info.Version,
		})
	}

	sort.Slice(plugins, func(i, j int) bool {
		return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
	})

	return plugins, nil
}

// ExtractArchive extracts a plugin archive (.zip, .tar.gz or .tar.xz) into destDir.
// The format is detected from the file's magic bytes, falling back to its name.
func ExtractArchive(archivePath, filename, destDir string) error {
	if err := sdkutils.FsEnsureDir(destDir); err != nil {
		return err
	}

	kind, err := detectArchiveKind(archivePath, filename)
	if err != nil {
		return err
	}

	switch kind {
	case "zip":
		return sdkutils.Unzip(archivePath, destDir)
	case "tar.gz":
		return sdkutils.Untar(archivePath, destDir)
	case "tar.xz":
		return sdkutils.UntarXz(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: upload a .zip, .tar.gz or .tar.xz file")
	}
}

// FindAndValidateSrc locates the plugin source root inside an extracted archive
// (the directory holding plugin.json, whether at the root or nested in a single
// wrapper folder), validates it has the required files and returns its info.
func FindAndValidateSrc(extractDir string) (sdkutils.PluginInfo, string, error) {
	srcRoot, err := sdkutils.FindPluginSrc(extractDir)
	if err != nil {
		return sdkutils.PluginInfo{}, "", fmt.Errorf("no plugin.json found in the uploaded archive")
	}

	for _, f := range requiredSrcFiles {
		if !sdkutils.FsExists(filepath.Join(srcRoot, f)) {
			return sdkutils.PluginInfo{}, "", fmt.Errorf("the plugin is missing a required file: %s", f)
		}
	}

	info, err := sdkutils.GetPluginInfoFromPath(srcRoot)
	if err != nil {
		return sdkutils.PluginInfo{}, "", fmt.Errorf("could not read plugin.json: %w", err)
	}
	if info.Package == "" {
		return sdkutils.PluginInfo{}, "", fmt.Errorf("plugin.json is missing the \"package\" field")
	}

	return info, srcRoot, nil
}

// SaveSource copies a validated plugin source tree into data/plugins/local/<pkg>,
// replacing any previous copy, and returns the destination path. Stale compiled
// .so artifacts shipped inside the archive are stripped so the install always
// rebuilds against the running core.
func SaveSource(srcRoot, pkg string) (string, error) {
	localDest := filepath.Join(sdkutils.PathPluginLocalDir, pkg)

	if err := os.RemoveAll(localDest); err != nil {
		return "", fmt.Errorf("could not clear existing source: %w", err)
	}
	if err := sdkutils.FsEnsureDir(localDest); err != nil {
		return "", err
	}
	if err := sdkutils.FsCopyDir(srcRoot, localDest, nil); err != nil {
		return "", fmt.Errorf("could not copy source: %w", err)
	}

	removeArtifacts(localDest)
	return localDest, nil
}

// ZipSource stages a clean copy of data/plugins/local/<pkg> (without VCS metadata
// or compiled artifacts), compresses it into <pkg>.zip and returns the zip path
// along with a cleanup func the caller must invoke once the file has been served.
func ZipSource(pkg string) (zipPath string, cleanup func(), err error) {
	localDir := filepath.Join(sdkutils.PathPluginLocalDir, pkg)
	if !sdkutils.FsExists(filepath.Join(localDir, "plugin.json")) {
		return "", func() {}, fmt.Errorf("plugin %q was not found in the local directory", pkg)
	}

	work := filepath.Join(sdkutils.PathTmpDir, "developer", "download", sdkutils.RandomStr(12))
	cleanup = func() { _ = os.RemoveAll(work) }

	stage := filepath.Join(work, pkg)
	if err := sdkutils.FsEmptyDir(stage); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := sdkutils.FsCopyDir(localDir, stage, nil); err != nil {
		cleanup()
		return "", func() {}, err
	}
	removeArtifacts(stage)

	zipPath = filepath.Join(work, pkg+".zip")
	if err := sdkutils.CompressZip(stage, zipPath); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("could not compress source: %w", err)
	}

	return zipPath, cleanup, nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// detectArchiveKind classifies an archive by its leading magic bytes, falling
// back to the filename extension when the magic is inconclusive.
func detectArchiveKind(archivePath, filename string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	head := make([]byte, 6)
	n, _ := f.Read(head)
	head = head[:n]

	switch {
	case len(head) >= 4 && head[0] == 'P' && head[1] == 'K' && (head[2] == 0x03 || head[2] == 0x05 || head[2] == 0x07):
		return "zip", nil
	case len(head) >= 2 && head[0] == 0x1f && head[1] == 0x8b:
		return "tar.gz", nil
	case len(head) >= 6 && head[0] == 0xfd && head[1] == '7' && head[2] == 'z' && head[3] == 'X' && head[4] == 'Z' && head[5] == 0x00:
		return "tar.xz", nil
	}

	name := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(name, ".zip"):
		return "zip", nil
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return "tar.gz", nil
	case strings.HasSuffix(name, ".tar.xz"), strings.HasSuffix(name, ".txz"):
		return "tar.xz", nil
	}

	return "", fmt.Errorf("unrecognized archive format")
}

// removeArtifacts strips VCS metadata and compiled plugin binaries from a copied
// source tree so neither bloats a download nor ships a stale .so into an install.
func removeArtifacts(root string) {
	_ = os.RemoveAll(filepath.Join(root, ".git"))

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".so") {
			_ = os.Remove(path)
		}
		return nil
	})
}
