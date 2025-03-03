package boot

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitDirs() {
	dirs := []string{
		sdkutils.PathConfigDir,
		sdkutils.PathCacheDir,
		sdkutils.PathPublicDir,
		filepath.Join(sdkutils.PathCacheDir, "assets"),
		filepath.Join(sdkutils.PathConfigDir, "plugins"),
		filepath.Join(sdkutils.PathConfigDir, "accounts"),
	}
	wg := sync.WaitGroup{}
	wg.Add(len(dirs))
	for _, d := range dirs {
		go func(d string) {
			defer wg.Done()
			if err := os.MkdirAll(d, sdkutils.PermDir); err != nil {
				log.Fatal(err)
			}
		}(d)
	}
	wg.Wait()

}
