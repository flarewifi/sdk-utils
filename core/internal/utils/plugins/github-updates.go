package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func GetGithubSrcURL(pkg string) (string, error) {
	def, err := GetPluginDef(pkg)
	if err != nil {
		return "", fmt.Errorf("unable to download: %w", err)
	}

	repoURL := sdkutils.NeutralizeGitURL(def.GitURL)

	return repoURL, nil
}

func GetTarballDownloadPath(pkg string) (string, error) {
	tarballSavedDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads")
	if err := sdkutils.FsEnsureDir(tarballSavedDir); err != nil {
		return "", fmt.Errorf("ensure dir error: %w", err)
	}

	path := filepath.Join(tarballSavedDir, fmt.Sprintf("%s.tar.gz", pkg))

	return path, nil
}

func CompileDownloadedTarball(tarball, pkg string) error {
	if err := extractTarball(tarball, pkg); err != nil {
		return err
	}

	srcPath := filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads", pkg)
	if err := BuildAssets(srcPath); err != nil {
		return fmt.Errorf("unable to build assets: %w", err)
	}

	if err := BuildTemplates(srcPath); err != nil {
		return fmt.Errorf("unable to build templates: %w", err)
	}

	workdir := filepath.Join(sdkutils.PathTmpDir, "builds", filepath.Base(srcPath))
	defer os.RemoveAll(workdir)

	if err := BuildPluginSo(srcPath, workdir); err != nil {
		return fmt.Errorf("unable to build plugin.so: %w", err)
	}

	return nil
}

func extractTarball(tarball, pkg string) error {
	// Extract the downloaded tar ball.
	dest := filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads")
	if err := sdkutils.FsEnsureDir(dest); err != nil {
		return fmt.Errorf("ensure dir exists error: %w", err)
	}

	if err := sdkutils.FsExtract(tarball, dest); err != nil {
		return fmt.Errorf("extracting error: %w", err)
	}

	if err := renameTarball(pkg); err != nil {
		return fmt.Errorf("error renaming extracted file: %w", err)
	}

	return nil
}

func renameTarball(pkg string) error {
	downloadsDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads")
	files, err := os.ReadDir(downloadsDir)
	if err != nil {
		return fmt.Errorf("failed to read extract directory: %w", err)
	}

	var extractedDirName string
	for _, file := range files {
		if file.IsDir() {
			if strings.Contains(file.Name(), pkg) {
				extractedDirName = file.Name()
			}
			break
		}
	}

	oldName := filepath.Join(downloadsDir, extractedDirName)
	newName := filepath.Join(downloadsDir, pkg)

	return os.Rename(oldName, newName)

}
