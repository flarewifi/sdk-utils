package sdkfs

import (
	"fmt"
	"os"
	"path/filepath"
)

func CopyAndReplaceDir(pathA, pathB string) error {
	// Walk through all files and directories in pathB
	return filepath.Walk(pathB, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Determine the corresponding path in pathA
		relPath, err := filepath.Rel(pathB, srcPath)
		if err != nil {
			return err
		}

		destPath := filepath.Join(pathA, relPath)

		if info.IsDir() {
			// If it's a directory, ensure the directory exists in pathA
			if err := os.MkdirAll(destPath, info.Mode()); err != nil {
				return fmt.Errorf("could not create directory: %v", err)
			}
		} else {
			// If it's a file, copy it and replace the one in pathA
			if err := CopyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy file from %s to %s: %v", srcPath, destPath, err)
			}
		}

		return nil
	})
}
