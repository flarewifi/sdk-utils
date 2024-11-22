/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkfs

import (
	"fmt"
	"os"
	"path/filepath"
)

// Makes sure the directory exists and is empty. Removes contents of the directory if it already exists.
func EmptyDir(dirPath string) error {
	if Exists(dirPath) {
		if err := os.RemoveAll(dirPath); err != nil {
			return err
		}
	}
	return os.MkdirAll(dirPath, PermDir)
}

func RmEmpty(dirPath string) error {
	emptyDirs := make([]string, 0)
	err := FindEmptyDirs(dirPath, &emptyDirs)
	if err != nil {
		return err
	}

	// Remove empty directories.
	for _, dir := range emptyDirs {
		removeErr := os.Remove(dir)
		if removeErr != nil {
			fmt.Println("Error removing directory:", removeErr)
		}

		// Remove empty parent directories.
		parentDir := filepath.Dir(dir)
		if isEmpty, err := IsEmptyDir(parentDir); err == nil && isEmpty {
			removeErr := os.Remove(parentDir)
			if removeErr != nil {
				fmt.Println("Error removing directory:", removeErr)
			}
		}
	}

	return nil
}

func FindEmptyDirs(dirPath string, emptyDirs *[]string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		*emptyDirs = append(*emptyDirs, dirPath)
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(dirPath, entry.Name())
			err := FindEmptyDirs(subDirPath, emptyDirs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func IsEmptyDir(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return false, err
	}

	return len(entries) == 0, nil
}
