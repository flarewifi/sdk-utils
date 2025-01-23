package sdkutils

import (
	"errors"
	"path/filepath"
)

func FindPluginSrc(dir string) (string, error) {
	files := []string{}
	err := FsListFiles(dir, &files, true)
	if err != nil {
		return dir, err
	}

	for _, f := range files {
		if filepath.Base(f) == "plugin.json" {
			return filepath.Dir(f), nil
		}
	}

	return "", errors.New("Can't find plugin.json in " + StripRootPath(dir))
}
