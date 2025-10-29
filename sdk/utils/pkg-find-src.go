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

func ValidatePluginSrc(src string) error {
	requiredFiles := []string{"plugin.json", "go.mod", "main.go"}

	for _, f := range requiredFiles {
		if !FsExists(filepath.Join(src, f)) {
			return errors.New(f + " not found in source path")
		}
	}

	return nil
}
