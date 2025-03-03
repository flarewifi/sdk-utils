/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 :xa
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package sdkutils

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	rootDir = "flarehotspot"
)

var (
	PathAppDir      = getRootDir()
	PathCoreDir     = filepath.Join(PathAppDir, "core")
	PathConfigDir   = filepath.Join(PathAppDir, "config")
	PathDefaultsDir = filepath.Join(PathConfigDir, ".defaults")
	PathPluginsDir  = filepath.Join(PathAppDir, "plugins")
	PathPublicDir   = filepath.Join(PathAppDir, "public")
	PathLogsDir     = filepath.Join(PathAppDir, "logs")
	PathSdkDir      = filepath.Join(PathAppDir, "sdk")
	PathTmpDir      = filepath.Join(PathAppDir, ".tmp")
	PathCacheDir    = filepath.Join(PathTmpDir, "cache")
	PathSqlcBin     = filepath.Join(PathAppDir, "bin", "sqlc")
)

// StripRootPath removes the project root directory prefix from absolute paths
func StripRootPath(p string) string {
	return strings.Replace(p, PathAppDir+string(filepath.Separator), "", 1)
}

func getRootDir() string {
	if dir := os.Getenv("APPDIR"); dir != "" {
		return dir
	}

	wd, _ := os.Getwd()
	for !strings.HasSuffix(wd, rootDir) {
		wd = filepath.Dir(wd)
		if wd == "/" {
			break
		}
	}

	if wd != "/" {
		return wd
	}

	dir, err := os.Getwd()
	if err == nil {
		return dir
	}

	dir = "."
	return dir
}
