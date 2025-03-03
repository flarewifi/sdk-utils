package migrate

import (
	"sort"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type MigDirection int

const (
	migration_Down MigDirection = iota
	migration_Up
)

func listFiles(dir string, d MigDirection) (files []string, err error) {
	list := []string{}
	if err = sdkutils.FsListFiles(dir, &list, false); err != nil {
		return files, err
	}

	files = []string{}
	if d == migration_Down {
		for _, f := range list {
			if strings.HasSuffix(f, ".down.sql") && !strings.HasPrefix(f, ".") {
				files = append(files, f)
			}
		}
		sdkutils.SliceReverseString(files)
	} else {
		for _, f := range list {
			if strings.HasSuffix(f, ".up.sql") && !strings.HasPrefix(f, ".") {
				files = append(files, f)
			}
		}
		sort.Strings(files)
	}

	return files, nil
}
