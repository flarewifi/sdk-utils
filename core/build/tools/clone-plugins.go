package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	GITHUB_TOKEN = os.Getenv("GITHUB_TOKEN")
)

func GitCheckoutMain() {
	dirPaths := []string{"core"}

	var pluginDirs []string
	sdkutils.FsListDirs("plugins", &pluginDirs, false)
	dirPaths = append(dirPaths, pluginDirs...)

	for _, dirPath := range dirPaths {
		fmt.Printf("Checking out main branch for %s...\n", dirPath)
		cmd := exec.Command("git", "checkout", "main")
		cmd.Dir = dirPath
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
	}
}

func GitCloneRepo(repo string, workDir string) {
	var gitUrl string
	if GITHUB_TOKEN != "" {
		gitUrl = fmt.Sprintf("https://oauth2:%s@github.com/%s.git", GITHUB_TOKEN, repo)
	} else {
		gitUrl = fmt.Sprintf("git@github.com:%s.git", repo)
	}

	dirname := filepath.Base(repo)
	os.RemoveAll(filepath.Join(workDir, dirname))

	fmt.Println("Cloning " + gitUrl + " in " + workDir)

	cmd := exec.Command("git", "clone", gitUrl)
	cmd.Dir = workDir
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
