package plugins

import (
	"core/internal/utils/logger"
)

type LoggerApi struct{}

func NewLoggerApi(pluginApi *PluginApi) {
	loggerApi := &LoggerApi{}
	pluginApi.LoggerAPI = loggerApi
}

func (l *LoggerApi) Info(title string, message string) error {
	calldepth := 1
	level := 0

	file, line := logger.GetCallerFileLine(calldepth)

	logger.LogToConsole(file, line, level, title, message)
	err := logger.LogToFile(file, line, level, title, message)
	return err
}

func (l *LoggerApi) Debug(title string, message string) error {
	calldepth := 1
	level := 1

	file, line := logger.GetCallerFileLine(calldepth)

	logger.LogToConsole(file, line, level, title, message)
	err := logger.LogToFile(file, line, level, title, message)
	return err
}

func (l *LoggerApi) Error(title string, message string) error {
	calldepth := 1
	level := 2

	file, line := logger.GetCallerFileLine(calldepth)

	logger.LogToConsole(file, line, level, title, message)
	err := logger.LogToFile(file, line, level, title, message)
	return err
}
