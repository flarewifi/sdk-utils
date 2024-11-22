package sdklogger

type ILoggerApi interface {
	// Logs title and body with info level to console and log file
	Info(title string, body ...any) error

	// Logs title and body with debug level to console and log file
	Debug(title string, body ...any) error

	// Logs title and body with error level to console and log file
	Error(title string, body ...any) error
}
