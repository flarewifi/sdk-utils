package updates

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	sdkplugin "sdk/api/plugin"
	"strings"
	"syscall"

	rpc "core/internal/rpc"
	"core/internal/utils/pkg"

	sdkextract "github.com/flarehotspot/go-utils/extract"
	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdksemver "github.com/flarehotspot/go-utils/semver"
)

const (
	EnvSpawner     = "SPAWNER"
	EnvCoreVersion = "CORE_VERSION"
	EnvValFlare    = "flare"
	EnvValUpdater  = "updater"
)

type CoreReleaseUpdate struct {
	Version        sdksemver.Version
	CoreZipFileUrl string
	ArchBinFileUrl string
}

type UpdateFiles struct {
	LocalCoreFilesPath    string
	LocalArchBinFilesPath string
	CoreReleaseUpdate
}

type LatestGithubRelease struct {
	TagName string `json:"tag_name"`
}

// Helper function to check if the process is spawned by flare cli
func IsSpawnedFromFlare() bool {
	spawnedFromFlareEnv := os.Getenv(EnvSpawner)
	if strings.ToLower(spawnedFromFlareEnv) == EnvValFlare {
		return true
	}
	return false
}

// Updates the core plugin from a the extracted latest core release
func Update() error {
	// get cwd as the destination for the copying
	cwd, err := os.Getwd()
	if err != nil {
		log.Println("Error getting cwd: ")
	}

	// get latest core release path
	crVersion := strings.ToLower(os.Getenv(EnvCoreVersion))
	fmt.Println("copying and replacing old files..")
	latestCRPath := filepath.Join(".tmp", "updates", "core", crVersion, "extracted")

	// update/copy and replace
	if err := sdkfs.CopyDir(latestCRPath, cwd, &sdkfs.CopyOpts{NoOverride: false, NonRecursive: false}); err != nil {
		log.Println("Error copying/updating the latest core release to flare path:", err)
		return err
	}

	return nil
}

// Executes the copied latest core release
func ExecuteFlare() error {
	// get the latest path
	flarePath := filepath.Join("bin", "flare")
	flareCmd := fmt.Sprintf("./%s", flarePath)

	// run the latest cli with "update" params
	flare := exec.Command(flareCmd, "server")
	flare.Stdout = os.Stdout
	flare.Stderr = os.Stderr

	// set env vars
	flare.Env = append(flare.Env, fmt.Sprintf("%s=%s", EnvSpawner, EnvValUpdater))

	// start
	if err := flare.Start(); err != nil {
		log.Println("Error starting new flare:", err)
		return err
	}

	return nil

}

// Helper function to check if the process id is running
func IsProcRunning(proc *os.Process) bool {
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		log.Println("Error:", err)
		return false
	}

	return true
}

// Checks if all the necessary core release files exist
func EnsureUpdateFilesExist() error {
	// TODO: ensure core and arch bin files exist
	coreAndArchBinFiles := []string{
		// "",
	}
	for _, f := range coreAndArchBinFiles {
		// TODO: find out proper file path
		if sdkfs.Exists("") {
			fmt.Println(f, " exists")
			continue
		}

		// do not proceed the update
		fmt.Println(f, " does not exist")
		log.Println("Core files not complete.")
		log.Println("Aborting update..")
		return errors.New("updater: error: core files not present")
	}

	return nil
}

// Executes the new flare cli with update params
func ExecuteUpdater(version sdksemver.Version) error {
	// get the latest path
	// convention -> ./tmp/udpates/core/<version>/extracted/
	cliPath := filepath.Join(".tmp", "updates", "core", sdksemver.StringifyVersion(version), "extracted")
	flarePath := filepath.Join(cliPath, "bin", "flare")
	flareCmd := fmt.Sprintf("./%s", flarePath)

	// run the latest cli with "update" params
	updater := exec.Command(flareCmd, "update")
	updater.Stdout = os.Stdout
	updater.Stderr = os.Stderr

	// set env vars
	updater.Env = append(updater.Env, fmt.Sprintf("%s=%s", EnvSpawner, EnvValFlare))
	updater.Env = append(updater.Env, fmt.Sprintf("CORE_VERSION=%s", sdksemver.StringifyVersion(version)))

	// start
	if err := updater.Start(); err != nil {
		log.Println("Error starting updater:", err)
		return err
	}

	return nil
}

// Fetches the latest core release from flare-server
func FetchLatestCoreRelease() (CoreReleaseUpdate, error) {
	srv, ctx := rpc.GetCoreMachineTwirpServiceAndCtx()
	latestCoreRelease, err := srv.FetchLatestCoreRelease(ctx, &rpc.FetchLatestCoreReleaseRequest{})
	if err != nil {
		log.Println("Error: ", err)
		return CoreReleaseUpdate{}, err
	}

	return CoreReleaseUpdate{
		Version: sdksemver.Version{
			Major: int(latestCoreRelease.Major),
			Minor: int(latestCoreRelease.Minor),
			Patch: int(latestCoreRelease.Patch),
		},
		CoreZipFileUrl: latestCoreRelease.CoreZipFileUrl,
		ArchBinFileUrl: latestCoreRelease.ArchBinFileUrl,
	}, nil
}

// Returns the installed core release version
func GetCurrentCoreVersion() (sdksemver.Version, error) {
	// get file content
	var meta struct {
		Name        string `json:"Name"`
		Package     string `json:"Package"`
		Description string `json:"Description"`
		Version     string `json:"Version"`
	}
	pluginJsonFilePath := filepath.Join(sdkpaths.CoreDir, "plugin.json")
	if err := readPluginReleaseData(&meta, pluginJsonFilePath); err != nil {
		log.Printf("Error reading %v: %v", pluginJsonFilePath, err)
		return sdksemver.Version{}, err
	}

	coreVersion, err := sdksemver.ParseVersion(meta.Version)
	if err != nil {
		log.Println("Error parsing plugin version:", err)
		return sdksemver.Version{}, err
	}

	return coreVersion, nil
}

// reads the plugin.json from the specified path and populates the meta interface
func readPluginReleaseData(meta interface{}, pluginJsonFilePath string) error {
	b, err := os.ReadFile(pluginJsonFilePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, meta); err != nil {
		log.Println("Error unmarshaling the json: ", err)
		return err
	}

	return nil
}

// Extracts and runs the downloaded core release, flare, with update params
func UpdateCore(localUpdateFiles UpdateFiles) error {
	// extract path convention .tmp/updates/core/<version>/extracted
	extractPath := filepath.Join(sdkpaths.TmpDir, "updates", "core", sdksemver.StringifyVersion(localUpdateFiles.Version), "extracted")
	fmt.Println("Extracting downloaded latest release to: ", extractPath)

	sdkextract.Extract(localUpdateFiles.LocalCoreFilesPath, extractPath)
	sdkextract.Extract(localUpdateFiles.LocalArchBinFilesPath, extractPath)

	if err := ExecuteUpdater(localUpdateFiles.Version); err != nil {
		log.Println("Error executing updater: ", err)
		return err
	}

	return nil
}

func CheckForPluginUpdates(pDatum *pkg.PluginInstallData, pInfo sdkplugin.PluginInfo) (bool, error) {
	switch pDatum.Def.Src {
	case "git":
		hasUpdates, err := CheckUpdatesFromGithub(pDatum, pInfo)
		if err != nil {
			log.Println("Error checking plugin updates from github: ", err)
			return false, err
		}
		return hasUpdates, nil
	case "store":
		hasUpdates, err := CheckUpdatesFromStore(&pDatum.Def, pInfo)
		if err != nil {
			log.Println("Error checking plugin updates from store: ", err)
			return false, err
		}
		return hasUpdates, nil
	default:
		return false, nil
	}
}

func CheckUpdatesFromGithub(pDatum *pkg.PluginInstallData, pInfo sdkplugin.PluginInfo) (bool, error) {
	author := pkg.GetAuthorNameFromGitUrl(*pDatum)
	repo := pkg.GetRepoFromGitUrl(*pDatum)

	// NOTE: release tags should adhere to semver

	// build github api url
	// https://api.github.com/repos/<author>/<repo>/releases/latest
	gitApiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", author, repo)

	// fetch latest release from github
	resp, err := http.Get(gitApiUrl)
	if err != nil {
		log.Println("Error fetching plugin's latest from github api: ", err)
		return false, err
	}
	defer resp.Body.Close()

	// handle non-200 status response code
	if resp.StatusCode != 200 {
		log.Printf("Fetching latest release unsuccessful: %d %s", resp.StatusCode, resp.Status)
		return false, err
	}

	// decode body as json
	var latestPR LatestGithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&latestPR); err != nil {
		log.Println("Error decoding JSON response: ", err)
		return false, err
	}
	fmt.Printf("Latest plugin release: %v\n", latestPR)

	// parse json to semver
	latestPRVersion, err := sdksemver.ParseVersion(latestPR.TagName)
	if err != nil {
		log.Println("Error parsing latest pr version: ", err)
		return false, err
	}
	fmt.Printf("Latest plugin release version: %v\n", latestPRVersion)

	currentPRVersion, err := sdksemver.ParseVersion(pInfo.Version)
	if err != nil {
		log.Println("Error parsing string version to semver version: ", err)
		return false, err
	}

	return sdksemver.HasUpdates(currentPRVersion, latestPRVersion), nil
}

func CheckUpdatesFromStore(p *pkg.PluginSrcDef, pinfo sdkplugin.PluginInfo) (bool, error) {
	// fetch latest plugin release from flare-server rpc
	srv, ctx := rpc.GetCoreMachineTwirpServiceAndCtx()
	qPlugins, err := srv.FetchLatestValidPRByPackage(ctx, &rpc.FetchLatestValidPRByPackageRequest{
		PluginPackage: p.StorePackage,
	})
	if err != nil {
		log.Println("Error fetching latest plugin release: ", err)
		return false, err
	}

	currVersion, err := sdksemver.ParseVersion(pinfo.Version)
	if err != nil {
		log.Printf("Error parsing raw version of plugin: %s: %s\n", pinfo.Package, err.Error())
		return false, err
	}

	// update plugin release zip file url def temporarily
	p.StoreZipUrl = qPlugins.PluginRelease.ZipFileUrl

	latestVersion := sdksemver.Version{
		Major: int(qPlugins.PluginRelease.Major),
		Minor: int(qPlugins.PluginRelease.Minor),
		Patch: int(qPlugins.PluginRelease.Patch),
	}

	return sdksemver.HasUpdates(currVersion, latestVersion), nil
}
