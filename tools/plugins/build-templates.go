package plugins

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildTemplates(pluginDir string) (err error) {
	fmt.Println("Checking for templates in ", pluginDir)

	templatesPath := filepath.Join(pluginDir, "resources/views")
	if !sdkutils.FsExists(templatesPath) {
		fmt.Println("No templates found in", templatesPath)
		return nil
	}

	var errout strings.Builder

	fmt.Println("Building templates in ", pluginDir)
	cmd := exec.Command("templ", "generate")
	cmd.Dir = pluginDir
	cmd.Stderr = &errout
	if err = cmd.Run(); err != nil {
		errmsg := errout.String()
		if errmsg != "" {
			return err
		}
		return fmt.Errorf("failed to build templates: %v", err)
	}

	fmt.Println("Templates built successfully")
	return nil
}
