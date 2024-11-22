package pkg

import (
	"fmt"
	"path/filepath"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

func LinkNodeModulesLib(workdir string) error {
	coreLibSrc := filepath.Join(sdkpaths.CoreDir, "resources/assets/lib")
	coreLibDest := filepath.Join(workdir, "node_modules/@flarehotspot/lib")

	if err := sdkfs.EmptyDir(filepath.Dir(coreLibDest)); err != nil {
		fmt.Println("Unable to initialize " + coreLibDest)
		return err
	}

	if err := sdkfs.Copy(coreLibSrc, coreLibDest); err != nil {
		fmt.Println("Error linking core assets lib to node_modules ", err)
		return err
	}

	fmt.Printf("Successfully linked %s -> %s\n", coreLibSrc, coreLibDest)
	return nil
}
