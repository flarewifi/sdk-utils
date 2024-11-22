package sdktargz

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

func TarGz(sourceDir, outputFile string) error {
	// Create the output file
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip writer
	gw := gzip.NewWriter(file)
	defer gw.Close()

	// Create a tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Walk through the directory
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a header for the current file
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update the name to reflect the correct path in the archive
		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If the file is not a directory, write the file content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if _, err = io.Copy(tw, file); err != nil {
				file.Close() // dont use defer
				return err
			}
			file.Close() // dont use defer
		}

		return nil
	})

	return err
}

func UntarGz(tarGzFile, outputDir string) error {
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

		return err
	}

	return nil
}
