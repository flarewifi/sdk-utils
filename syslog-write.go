/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkutils

import (
	"os"
	"path"
	"time"
)

func LogNotice(msg string) error {
	return write(TypeNotice, msg)
}

func LogSuccess(msg string) error {
	return write(TypeSuccess, msg)
}

func LogError(msg string) error {
	return write(TypeError, msg)
}

func Log(msg string) error {
	return write(TypeLog, msg)
}

func write(t LogType, msg string) error {
	stamp := time.Now().Format("20060102150405")
	file := path.Join(PathLogsDir, string(t)+"-"+stamp+".log")
	err := os.WriteFile(file, []byte(msg), 0644)
	return err
}
