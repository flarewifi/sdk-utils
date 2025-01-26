/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type ILoggerApi interface {
	// Logs title and body with info level to console and log file
	Info(title string, message string) error

	// Logs title and body with debug level to console and log file
	Debug(title string, message string) error

	// Logs title and body with error level to console and log file
	Error(title string, message string) error
}
