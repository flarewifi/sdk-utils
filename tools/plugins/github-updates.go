//go:build !mono

package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	views "core/resources/views/admin/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func GetGithubReleases(gitURL string) ([]views.Release, error) {
	repo, err := sdkutils.ParseGitSource(gitURL)
	if err != nil {
		return nil, fmt.Errorf("parse git source error: %w", err)
	}

	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", repo.Owner, repo.Repo))
	if err != nil {
		return nil, fmt.Errorf("get releases error: %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invaid response from github")

	}

	var releases []views.Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("unable to decode response body: %w", err)
	}

	return releases, nil
}

func GetTarballDownloadURL(ref, pkg string) (string, error) {
	def, err := GetPluginDef(pkg)
	if err != nil {
		return "", fmt.Errorf("unable to download: %w", err)
	}

	gitURL := sdkutils.NeutralizeGitURL(def.GitURL)
	tarballDownloadURL := fmt.Sprintf("%s?ref=%s", gitURL, ref)

	return tarballDownloadURL, nil
}

func GetTarballDownloadPath(pkg string) (string, error) {
	tarballSavedDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads")
	if err := sdkutils.FsEmptyDir(tarballSavedDir); err != nil {
		return "", fmt.Errorf("ensure dir error: %w", err)
	}

	path := filepath.Join(tarballSavedDir, fmt.Sprintf("%s.tar.gz", pkg))

	return path, nil
}

func CompileDownloadedTarball(tarball, pkg string) error {
	if err := extractTarball(tarball, pkg); err != nil {
		return err
	}

	srcPath, err := sdkutils.FindPluginSrc(filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads", pkg))
	if err != nil {
		return fmt.Errorf("unable to find plugin source: %w", err)
	}

	if err := BuildAssets(srcPath); err != nil {
		return fmt.Errorf("unable to build assets: %w", err)
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

// Removes the download directory.
func CleanupDownload() error {
	downloadsDir := filepath.Join(sdkutils.PathTmpDir, "plugins", "downloads")
	if err := os.RemoveAll(downloadsDir); err != nil {
		return fmt.Errorf("unable to remove from src: %w", err)
	}

	return nil
}
