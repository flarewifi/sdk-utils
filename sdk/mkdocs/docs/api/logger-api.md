# ILoggerApi

The `ILoggerApi` is used for logging messages at different log levels. It provides a standardized way to log informational, debug, and error messages. Developers can use this API to ensure consistent logging across the system. Logs can be viewed in the admin dashboard `Admin Web Interface > System > Logs`.

## Interface Definition

```go
type ILoggerApi interface {
    Info(message string) error
    Debug(message string) error
    Error(message string) error
}
```

## Log Levels

| Method | Description |
| ---- | ---- |
| `Info(message string) error` | Logs a general information message. |
| `Debug(message string) error` | Logs a debug message to help with troubleshooting. |
| `Error(message string) error` | Logs an error message indicating an issue that needs attention. |


## Usage Examples

### Logging an Informational Message

```go
api.Logger().Info("Application started succesfully!")
```

### Logging a Debug Message

```go
api.Logger().Debug("Fetching user data from database...")
```

### Logging an Error Message

```go
api.Logger().Error("Database connection failed!")
```

## Additional Resources

For more details on logging configuration used for debugging, see [Debugging](../guides/debugging.md) to learn more.
