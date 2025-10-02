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
	PathAppDir            = getRootDir()
	PathCoreDir           = filepath.Join(PathAppDir, "core")
	PathDataDir           = filepath.Join(PathAppDir, "data")
	PathConfigDefaultsDir = filepath.Join(PathAppDir, "defaults")
	PathConfigDir         = filepath.Join(PathDataDir, "config")
	PathLogsDir           = filepath.Join(PathAppDir, "logs")
	PathSdkDir            = filepath.Join(PathAppDir, "sdk")
	PathStorageDir        = filepath.Join(PathAppDir, "storage")
	PathSqlcBin           = filepath.Join(PathAppDir, "bin", "sqlc")
	PathPluginSystemDir   = filepath.Join(PathAppDir, "plugins", "system")
	PathPluginInstallDir  = filepath.Join(PathAppDir, "plugins", "installed")
	PathPluginBackupsDir  = filepath.Join(PathAppDir, "plugins", "backups")
	PathPluginUpdatesDir  = filepath.Join(PathAppDir, "plugins", "updates")
	PathPluginLocalDir    = filepath.Join(PathDataDir, "plugins", "local")
	PathSystemUpdateDir   = filepath.Join(PathStorageDir, "system", "update")
	PathPublicDir         = filepath.Join(PathAppDir, "public")
	PathTmpDir            = filepath.Join(PathAppDir, ".tmp")
	PathCacheDir          = filepath.Join(PathTmpDir, "cache")
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
