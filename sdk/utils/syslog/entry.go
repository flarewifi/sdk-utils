/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdksyslog

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

type LogType string

const (
	TypeNotice  LogType = "notice"
	TypeSuccess LogType = "success"
	TypeError   LogType = "error"
	TypeLog     LogType = "log"
)

type LogEntry struct {
	path string
}

func NewLogEntry(path string) *LogEntry {
	return &LogEntry{path}
}

func (self *LogEntry) Type() LogType {
	file := filepath.Base(self.path)

	if strings.HasPrefix(string(TypeNotice), file) {
		return TypeNotice
	}

	if strings.HasPrefix(string(TypeSuccess), file) {
		return TypeSuccess
	}

	if strings.HasPrefix(string(TypeError), file) {
		return TypeError
	}

	return TypeLog
}

func (self *LogEntry) Read() (msg string) {
	b, err := os.ReadFile(self.path)
	if err != nil {
		log.Println(err)
		return "Unable to read log message."
	}
	return string(b)
}
