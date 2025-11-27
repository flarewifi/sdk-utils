package sdkutils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CompressZip compresses files into a zip file
func CompressZip(srcDir string, destFile string) error {
	if err := FsEnsureDir(filepath.Dir(destFile)); err != nil {
		return err
	}

	fmt.Println("Zipping: ", StripRootPath(srcDir), " -> ", StripRootPath(destFile))
	cmd := exec.Command("zip", "-r", destFile, ".")
	cmd.Dir = srcDir
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error zipping: ", err)
		return err
	}
	return nil
}

// CompressTar compresses files into a tar.gz file
func CompressTar(sourceDir, outputFile string) error {
	if err := FsEnsureDir(filepath.Dir(outputFile)); err != nil {
		return err
	}

	// Create the output file
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	// Create a gzip writer
	gw := gzip.NewWriter(file)

	// Create a tar writer
	tw := tar.NewWriter(gw)

	// Walk through the directory
	walkErr := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path for the tar header
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself (when relPath is ".")
		if relPath == "." {
			return nil
		}

		// Use Lstat to get info without following symlinks
		info, err = os.Lstat(path)
		if err != nil {
			return err
		}

		// Handle symlinks properly
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		// Create a header for the current file
		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}

		// Update the name to reflect the correct path in the archive
		header.Name = relPath

		// Write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If the file is not a directory and not a symlink, write the file content
		if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			f, err := os.Open(path)
			if err != nil {
				return err
			}

			if _, err = io.Copy(tw, f); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}

		return nil
	})

	// Close writers in correct order: tar -> gzip -> file
	// Must close tar first to write final block
	if err := tw.Close(); err != nil {
		gw.Close()
		file.Close()
		return err
	}

	// Must close gzip to write gzip footer
	if err := gw.Close(); err != nil {
		file.Close()
		return err
	}

	// Close the file
	if err := file.Close(); err != nil {
		return err
	}

	return walkErr
}

// Untar extracts tar file to a output directory
func Untar(tarGzFile, outputDir string) error {
	if err := FsEnsureDir(filepath.Dir(outputDir)); err != nil {
		return err
	}

	// Open the tar.gz file
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip reader
	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	// Create a tar reader
	tr := tar.NewReader(gr)

	// Iterate through the files in the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}

		if err != nil {
			return err
		}

		// Construct the full output path
		outputPath := filepath.Join(outputDir, header.Name)

		// Handle directories
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(outputPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
			continue
		}

		// Handle files
		file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return err
		}

		if _, err = io.Copy(file, tr); err != nil {
			file.Close() // dont use defer
			return err
		}

		file.Close() // dont use defer

		fmt.Printf("Extracted: %s\n", outputPath)
	}

	return nil
}

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

		// Make sure to close the files at the end of every iteration
		err := func(file *zip.File, filePath string) error {
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

			return nil
		}(file, filePath)

		if err != nil {
			return err
		}

		fmt.Printf("Extracted: %s\n", filePath)
	}
	return nil
}
