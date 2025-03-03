//go:build dev

package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildTemplates(pluginDir string) (err error) {
	templatesPath := filepath.Join(pluginDir, "resources/views")
	if !sdkutils.FsExists(templatesPath) {
		fmt.Println("No templates found in", templatesPath)
		return nil
	}

	fmt.Println("Building templates in ", pluginDir)
	cmd := exec.Command("templ", "generate")
	cmd.Dir = pluginDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Templates built successfully")
	return nil
}

func removeDanglingTemplFile(templgoFile string) (err error) {
	templFile := strings.Replace(templgoFile, "_templ.go", ".templ", 1)
	if !sdkutils.FsExists(templFile) {
		err = os.Remove(templgoFile)
	}
	return
}
