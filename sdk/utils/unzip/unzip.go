package sdkunzip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Unzip extracts the contents of a zip archive to a target directory
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Iterate through each file in the zip archive
	for _, file := range r.File {
		// Create the full file path
		filePath := filepath.Join(dest, file.Name)

		// Check for directory traversal vulnerability (protect against malicious zips)
		if !strings.HasPrefix(filePath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", filePath)
		}

		// If the file is a directory, create it
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Ensure the directory for the file exists
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		// Open the file in the zip archive
		srcFile, err := file.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// Create the destination file
		destFile, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		// Copy the contents from the zip archive to the destination file
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return err
		}

		fmt.Printf("Extracted: %s\n", filePath)
	}
	return nil
}
