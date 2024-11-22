package tools

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdkruntime "github.com/flarehotspot/go-utils/runtime"
	sdkstr "github.com/flarehotspot/go-utils/strings"
)

func InstallGo(installPath string) {
	if installPath == "" {
		installPath = os.Getenv("GO_CUSTOM_PATH")
	}

	if installPath == "" {
		installPath = filepath.Join(sdkpaths.AppDir, "go")
	}

	// get go version from ".go-version"
	b, err := os.ReadFile(".go-version")
	if err != nil {
		panic(fmt.Sprintf("Error reading .go-version file: %s", err))
	}

	GOOS := sdkruntime.GOOS
	GOARCH := sdkruntime.GOARCH
	GOVERSION := strings.ReplaceAll(strings.TrimSpace(string(b)), "go", "")
	// GOVERSION := sdkruntime.GO_VERSION

	if GoInstallExists(installPath) {
		fmt.Printf("Go version %s already installed to %s\n", GOVERSION, installPath)
		return
	}

	EXTRACT_PATH := filepath.Join(sdkpaths.CacheDir, "downloads", fmt.Sprintf("go%s-%s-%s", GOVERSION, GOOS, GOARCH))
	err = downloadAndExtractGo(GOOS, GOARCH, GOVERSION, EXTRACT_PATH)
	if err != nil {
		panic(err)
	}

	fmt.Println("Installing Go version to: ", installPath)

	err = os.RemoveAll(installPath)
	if err != nil {
		panic(err)
	}

	err = sdkfs.RenameDir(filepath.Join(EXTRACT_PATH, "go"), installPath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Go version %s installed to %s\n", GOVERSION, installPath)
	fmt.Printf("To use the newly installed Go version, run: \n\nexport PATH=%s/bin:$PATH\n", installPath)
}

func GoInstallExists(installPath string) bool {
	fmt.Println("Checking if Go is already installed...")

	goBin := filepath.Join(installPath, "bin", "go")
	cmd := exec.Command(goBin, "env")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error checking existing go install: ", err)
		return false
	}

	envValues := map[string]string{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envValues[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	GOOS := sdkruntime.GOOS
	GOARCH := sdkruntime.GOARCH
	GOVERSION := sdkruntime.GO_VERSION

	goos := sdkstr.TrimChars(envValues["GOOS"], "\"", "'")
	goarch := sdkstr.TrimChars(envValues["GOARCH"], "\"", "'")
	goversion := strings.TrimPrefix(sdkstr.TrimChars(envValues["GOVERSION"], "\"", "'"), "go")

	if goos == GOOS && goarch == GOARCH && goversion == GOVERSION {
		return true
	} else {
		log.Println("Go version check mismatch!")
		log.Println("goos: ", goos)
		log.Println("goarch: ", goarch)
		log.Println("GOOS: ", GOOS)
		log.Println("GOARCH: ", GOARCH)
		return false
	}
}

func downloadAndExtractGo(goos, goarch, version, extractPath string) error {
	err := sdkfs.EnsureDir(filepath.Dir(extractPath))
	if err != nil {
		return err
	}

	fmt.Printf("Downloading Go version %s for %s-%s\n", version, goos, goarch)
	url := fmt.Sprintf("https://golang.org/dl/go%s.%s-%s.tar.gz", version, goos, goarch)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a temporary file to store the downloaded tar.gz file
	tmpFile, err := os.CreateTemp("", "golang*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy the downloaded content to the temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}

	// Extract the tar.gz file to the specified path
	fmt.Println("Extracting Go to: ", extractPath)

	// TODO: Fix too many open files on MacOS
	// err = sdkextract.Extract(tmpFile.Name(), extractPath)
	err = extractTarGz(tmpFile.Name(), extractPath)
	if err != nil {
		return err
	}

	return nil
}

func extractTarGz(srcFile, destPath string) error {
	sdkfs.EmptyDir(destPath)
	cmd := exec.Command("tar", "xzf", srcFile, "-C", destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
