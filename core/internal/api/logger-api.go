package api

import (
	"core/internal/modules/logger"
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

// Info/Debug/Error emit to stdout (syslog/logread) + the rotating app.log file
// + live SSE subscribers via logger.Emit. They no longer write to the database,
// so they are safe to call from inside a DB transaction (a logger DB write on
// the single-connection pool used to self-deadlock an enclosing transaction).
func (l *LoggerApi) Info(message string) error {
	file, line := logger.GetCallerFileLine(1)
	logger.Emit(0, file, line, message)
	return nil
}

func (l *LoggerApi) Debug(message string) error {
	file, line := logger.GetCallerFileLine(1)
	logger.Emit(1, file, line, message)
	return nil
}

func (l *LoggerApi) Error(message string) error {
	file, line := logger.GetCallerFileLine(1)
	logger.Emit(2, file, line, message)
	return nil
}
