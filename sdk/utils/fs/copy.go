package sdkfs

import "os"

func Copy(src string, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return CopyDir(src, dst, &CopyOpts{})
	} else {
		return CopyFile(src, dst)
	}
}
