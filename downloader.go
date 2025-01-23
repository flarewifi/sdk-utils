package sdkutils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Downloader struct {
	srcUrl   string
	destPath string
}

func NewDownloader(srcUrl string, destPath string) *Downloader {
	return &Downloader{
		srcUrl:   srcUrl,
		destPath: destPath,
	}
}

func (d *Downloader) Download() error {
	// Prepare the parent directory
	if err := os.MkdirAll(filepath.Dir(d.destPath), 0755); err != nil {
		return err
	}

	// Create the file
	out, err := os.Create(d.destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	log.Println("Downloading", d.srcUrl, "to", d.destPath)

	// Get the data
	resp, err := http.Get(d.srcUrl)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		os.Remove(d.destPath)
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
