package sdkutils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/goccy/go-json"
)

const (
	PermDir  = 0755
	PermFile = 0644
)

type FsFindFilterFn func(currDir string, entry string, stat os.FileInfo) bool
type FsFindReturnFn func(currDir string, entry string, stat os.FileInfo) string
type FsFindOpts struct {
	StopRecursion bool
}

func FsFind(dir string, filter FsFindFilterFn, ret FsFindReturnFn, opts FsFindOpts) []string {
	results := []string{}

	if FsExists(dir) && FsIsDir(dir) {
		var files []string
		if err := FsListFiles(dir, &files, false); err != nil {
			return results
		}

		for _, entry := range files {
			stat, err := os.Stat(entry)
			if err != nil {
				continue
			}
			entry = filepath.Base(entry)
			ok := filter(dir, entry, stat)
			if ok {
				results = append(results, ret(dir, entry, stat))
			}

			if ok && opts.StopRecursion {
				return results
			}
		}

		var dirs []string
		if err := FsListDirs(dir, &dirs, false); err != nil {
			return results
		}

		for _, entry := range dirs {
			stat, err := os.Stat(entry)
			if err != nil {
				continue
			}
			entry = filepath.Base(entry)
			ok := filter(dir, entry, stat)
			if ok {
				results = append(results, ret(dir, entry, stat))
			}
			if ok && opts.StopRecursion {
				return results
			}
			subResults := FsFind(entry, filter, ret, opts)
			results = append(results, subResults...)
		}

		return results
	}

	return results

}

func FsIsFile(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false // Path does not exist or there was an error accessing it
	}

	return !info.IsDir() && (info.Mode()&os.ModeType == 0) // Check if it's not a directory and is a regular file
}

func FsReadFile(f string) (string, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FsAppendFile appends data to a file named by filename.
// If the file does not exist, FsAppendFile creates it with permissions perm.
func FsAppendFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func FsExists(paths ...string) bool {
	for _, path := range paths {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return false
		}
	}
	return true
}

func FsEnsureDir(dirs ...string) error {
	for _, dir := range dirs {
		if !FsExists(dir) {
			err := os.MkdirAll(dir, PermDir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type FsCopyOpts struct {
	NoOverride   bool
	NonRecursive bool
}

func FsCopyDir(srcDir, destDir string, opts *FsCopyOpts) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &FsCopyOpts{}
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if opts.NoOverride {
			if _, err := os.Stat(destPath); err == nil {
				continue
			}
		}

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			continue
		}

		dir := filepath.Dir(destPath)
		if err := FsEnsureDir(dir); err != nil {
			continue
		}

		if entry.IsDir() && !opts.NonRecursive {
			if err := FsCopyDir(sourcePath, destPath, opts); err != nil {
				continue
			}
		} else if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			if err := FsCopySymLink(sourcePath, destPath); err != nil {
				continue
			}
		} else {
			if err := FsCopyFile(sourcePath, destPath); err != nil {
				continue
			}
		}
	}

	return nil
}

func FsCopyFile(srcFile, dstFile string) error {
	FsEnsureDir(filepath.Dir(dstFile))

	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	srcStat, err := os.Stat(srcFile)
	if err != nil {
		return err
	}

	err = os.Chmod(dstFile, srcStat.Mode())
	if err != nil {
		return err
	}

	return nil
}

func FsCopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}

// Copy file or directory
func FsCopy(src string, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return FsCopyDir(src, dst, &FsCopyOpts{})
	} else {
		return FsCopyFile(src, dst)
	}
}

func JsonWrite(f string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f, b, PermFile)
}

func JsonRead(f string, v any) error {
	b, err := os.ReadFile(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// FsListDirs returns directories inside dir. Returned directory paths are prepended with parent directory path.
func FsListDirs(path string, directories *[]string, recursive bool) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if stat.Mode() == os.ModeSymlink {
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}

		path = target
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if FsIsDir(filepath.Join(path, entry.Name())) {
			*directories = append(*directories, filepath.Join(path, entry.Name()))

			if recursive {
				subdirPath := filepath.Join(path, entry.Name())
				err := FsListDirs(subdirPath, directories, recursive)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// FsIsDir returns true if path is a directory. It follows symlinks.
func FsIsDir(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	if stat.IsDir() {
		return true // It's a directory
	}

	if stat.Mode() == os.ModeSymlink {
		target, err := os.Readlink(path)
		if err != nil {
			return false // Error reading symbolic link target
		}

		targetInfo, err := os.Stat(target)
		if err != nil {
			return false // Error getting information about the target
		}

		return targetInfo.IsDir()
	}

	return false
}

// FsListFiles returns list if files within dir. File paths are prepended with dir. It follows symlinks.
func FsListFiles(dir string, files *[]string, recursive bool) error {
	stat, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if stat.Mode() == os.ModeSymlink {
		target, err := os.Readlink(dir)
		if err != nil {
			return err
		}

		dir = target
	}

	fileEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range fileEntries {
		entryPath := filepath.Join(dir, entry.Name())
		if FsIsDir(entryPath) {
			if recursive {
				err = FsListFiles(entryPath, files, recursive)
				if err != nil {
					return err
				}
			}
		} else {
			*files = append(*files, entryPath)
		}
	}

	return nil
}

func FsMoveDir(sourceDir, destDir string) error {
	// Check if the source directory exists
	_, err := os.Stat(sourceDir)
	if err != nil {
		return err
	}

	// Create the destination directory if it doesn't exist
	err = os.MkdirAll(destDir, 0755) // Directories with permission mode 0755
	if err != nil {
		return err
	}

	// Walk through the source directory and move files and subdirectories
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create the destination path for the current item
		destPath := filepath.Join(destDir, path[len(sourceDir):])

		if info.IsDir() {
			// Create the directory in the destination path with permission mode 0755
			err := os.MkdirAll(destPath, 0755) // Directories with permission mode 0755
			if err != nil {
				return err
			}
		} else {
			// Move the file to the destination path
			err := os.Rename(path, destPath)
			if err != nil {
				return err
			}

			// Set the permission mode for the moved file
			err = os.Chmod(destPath, 0644) // Files with permission mode 0644
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Remove the source directory after successfully moving its contents
	err = os.RemoveAll(sourceDir)
	if err != nil {
		return err
	}

	return nil
}

// FsMoveFile moves a file from sourcePath to destPath.
func FsMoveFile(sourcePath, destPath string) error {
	// Attempt to rename the file (this is a quick move if on the same filesystem)
	if err := os.Rename(sourcePath, destPath); err != nil {
		// If renaming fails, we check if the error is because the file is on a different filesystem
		if linkErr, ok := err.(*os.LinkError); ok {
			fmt.Printf("Link error encountered: %v\n", linkErr)
			// Attempt to copy and then remove the source file
			if err := FsCopyFile(sourcePath, destPath); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
			if err := os.Remove(sourcePath); err != nil {
				return fmt.Errorf("failed to remove source file: %w", err)
			}
		} else {
			return fmt.Errorf("failed to move file: %w", err)
		}
	}
	return nil
}

func FsPrettyByteSize(b int) string {
	bf := float64(b)
	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}

// Alias to FsMoveDir(src, dst)
func FsRenameDir(src string, dst string) (err error) {
	return FsMoveDir(src, dst)
}

// Alias to FsMoveFile(src, dst)
func FsRenameFile(src string, dst string) (err error) {
	return FsMoveFile(src, dst)
}

// Makes sure the directory exists and is empty. Removes contents of the directory if it already exists.
func FsEmptyDir(dirPath string) error {
	if FsExists(dirPath) {
		if err := os.RemoveAll(dirPath); err != nil {
			return err
		}
	}
	return os.MkdirAll(dirPath, PermDir)
}

// Removes empty directories
func FsRmEmpty(dirPath string) error {
	emptyDirs := make([]string, 0)
	err := FsFindEmptyDirs(dirPath, &emptyDirs)
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
		if isEmpty, err := FsIsEmptyDir(parentDir); err == nil && isEmpty {
			removeErr := os.Remove(parentDir)
			if removeErr != nil {
				fmt.Println("Error removing directory:", removeErr)
			}
		}
	}

	return nil
}

// Returns a list of empty directories
func FsFindEmptyDirs(dirPath string, emptyDirs *[]string) error {
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
			err := FsFindEmptyDirs(subDirPath, emptyDirs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Returns true of the directory is empty
func FsIsEmptyDir(dirPath string) (bool, error) {
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

// Remove files that match the glob pattern
func FsRmPattern(dirPath string, globPattern string) error {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			matched, matchErr := filepath.Match(globPattern, info.Name())
			if matchErr != nil {
				return matchErr
			}

			if matched {
				removeErr := os.Remove(path)
				if removeErr != nil {
					return removeErr
				}
			}
		}
		return nil
	})

	return err
}

// Write to file with default perssion 0644
func FsWriteFile(path string, data []byte) error {
	if err := FsEnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, PermFile); err != nil {
		return err
	}

	return nil
}

var (
	MagicNumZip                 = []byte{0x50, 0x4B, 0x03, 0x04}
	MagicNumGzip                = []byte{0x1F, 0x8B}
	ErrUnknownCompressionFormat = errors.New("unknown compression format")
)

// Extract a zip or tar file
func FsExtract(file string, dest string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	// Read the first 4 bytes (or more if needed for other formats)
	buf := make([]byte, 4)
	if _, err := f.Read(buf); err != nil {
		return err
	}

	// identify compression format
	switch {
	case bytes.HasPrefix(buf, MagicNumZip):
		return Unzip(file, dest)
	case bytes.HasPrefix(buf, MagicNumGzip):
		return Untar(file, dest)
	}

	return ErrUnknownCompressionFormat
}
