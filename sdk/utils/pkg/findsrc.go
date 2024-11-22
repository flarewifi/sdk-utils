package sdkpkg

import (
	"errors"
	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	"path/filepath"
)

func FindPluginSrc(dir string) (string, error) {
	files := []string{}
	err := sdkfs.LsFiles(dir, &files, true)
	if err != nil {
		return dir, err
	}

	for _, f := range files {
		if filepath.Base(f) == "plugin.json" {
			return filepath.Dir(f), nil
		}
	}

	return "", errors.New("Can't find plugin.json in " + sdkpaths.StripRoot(dir))
}
