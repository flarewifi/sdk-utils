package sdkutils

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func DownloadFile(url, dest string) (<-chan int, <-chan error) {
	progress := make(chan int)
	errChan := make(chan error, 1)

	log.Println("Downloading", url, "to", dest)

	go func() {
		defer close(progress)
		defer close(errChan)

		resp, err := http.Get(url)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- errors.New("bad status: " + strconv.Itoa(resp.StatusCode))
			return
		}

		if err := os.MkdirAll(filepath.Dir(dest), PermDir); err != nil {
			errChan <- err
			return
		}

		file, err := os.Create(dest)
		if err != nil {
			errChan <- err
			return
		}
		defer file.Close()

		totalSize := resp.ContentLength
		if totalSize <= 0 {
			errChan <- errors.New("unknown file size")
			return
		}

		written := int64(0)
		buf := make([]byte, 1024*8)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				_, writeErr := file.Write(buf[:n])
				if writeErr != nil {
					errChan <- writeErr
					return
				}
				written += int64(n)
				progress <- int((written * 100) / totalSize)
			}
			if err == io.EOF {
				if written != totalSize {
					errChan <- errors.New("short read")
					return
				}

				errChan <- nil
				break
			}
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	return progress, errChan
}
