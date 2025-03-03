package plugins

import (
	"fmt"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func LinkNodeModulesLib(workdir string) error {
	coreLibSrc := filepath.Join(sdkutils.PathCoreDir, "resources/assets/lib")
	coreLibDest := filepath.Join(workdir, "node_modules/@flarehotspot/lib")

	if err := sdkutils.FsEmptyDir(filepath.Dir(coreLibDest)); err != nil {
		fmt.Println("Unable to initialize " + coreLibDest)
		return err
	}

	if err := sdkutils.FsCopy(coreLibSrc, coreLibDest); err != nil {
		fmt.Println("Error linking core assets lib to node_modules ", err)
		return err
	}

	fmt.Printf("Successfully linked %s -> %s\n", coreLibSrc, coreLibDest)
	return nil
}
