package api

import (
	"context"
	"core/db/models"
	"core/internal/modules/logger"
	"core/utils/config"
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

// isLoggingEnabled checks if logging is enabled in the application config.
func (l *LoggerApi) isLoggingEnabled() bool {
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		return false // Default to disabled if config cannot be read
	}
	return appCfg.EnableLogging
}

func (l *LoggerApi) Info(message string) error {
	calldepth := 1
	level := 0

	file, line := logger.GetCallerFileLine(calldepth)

	// Always log to console
	logger.LogToConsole(file, line, level, message)

	// Only write to database if logging is enabled
	if !l.isLoggingEnabled() {
		return nil
	}

	info := l.api.Info()
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

	file, line := logger.GetCallerFileLine(calldepth)

	// Always log to console
	logger.LogToConsole(file, line, level, message)

	// Only write to database if logging is enabled
	if !l.isLoggingEnabled() {
		return nil
	}

	info := l.api.Info()
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

	file, line := logger.GetCallerFileLine(calldepth)

	// Always log to console
	logger.LogToConsole(file, line, level, message)

	// Only write to database if logging is enabled
	if !l.isLoggingEnabled() {
		return nil
	}

	info := l.api.Info()
	err := l.api.models.Log().Create(context.Background(), models.CreateLogParams{
		Package:    info.Package,
		Level:      LogLevelError,
		Message:    message,
		Filepath:   file,
		LineNumber: line,
	})
	return err
}
