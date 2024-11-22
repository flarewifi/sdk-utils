package sdkzip

import (
	"fmt"
	"os/exec"

	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

func Zip(srcDir string, destFile string) error {
	fmt.Println("Zipping: ", sdkpaths.StripRoot(srcDir), " -> ", sdkpaths.StripRoot(destFile))
	cmd := exec.Command("zip", "-r", destFile, ".")
	cmd.Dir = srcDir
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error zipping: ", err)
		return err
	}
	return nil
}
