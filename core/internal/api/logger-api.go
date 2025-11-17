package api

import (
	"context"
	"core/db/models"
	"core/internal/utils/logger"
)

const (
	LogLevelInfo  string = "info"
	LogLevelDebug string = "debug"
	LogLevelError string = "error"
)

type LoggerApi struct {
	api *PluginApi
}

func NewLoggerApi(api *PluginApi) {
	loggerApi := &LoggerApi{api: api}
	api.LoggerAPI = loggerApi
}

func (l *LoggerApi) Info(message string) error {
	calldepth := 1
	level := 0

	info := l.api.Info()
	file, line := logger.GetCallerFileLine(calldepth)

	logger.LogToConsole(file, line, level, message)
	err := l.api.models.Log().Create(context.Background(), models.CreateLogParams{
		Package:    info.Package,
		Level:      LogLevelInfo,
		Message:    message,
		Filepath:   file,
		LineNumber: line,
	})
	return err
}

func (l *LoggerApi) Debug(message string) error {
	calldepth := 1
	level := 1

	info := l.api.Info()
	file, line := logger.GetCallerFileLine(calldepth)

	logger.LogToConsole(file, line, level, message)
	err := l.api.models.Log().Create(context.Background(), models.CreateLogParams{
		Package:    info.Package,
		Level:      LogLevelDebug,
		Message:    message,
		Filepath:   file,
		LineNumber: line,
	})
	return err
}

func (l *LoggerApi) Error(message string) error {
	calldepth := 1
	level := 2

	info := l.api.Info()
	file, line := logger.GetCallerFileLine(calldepth)

	logger.LogToConsole(file, line, level, message)
	err := l.api.models.Log().Create(context.Background(), models.CreateLogParams{
		Package:    info.Package,
		Level:      LogLevelError,
		Message:    message,
		Filepath:   file,
		LineNumber: line,
	})
	return err
}
