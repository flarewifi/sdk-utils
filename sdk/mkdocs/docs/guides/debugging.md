# Debugging

Debugging can be done by using the [ILoggerApi](../api/logger-api.md). Logs can be viewed in the admin dashboard `Admin Web Interface > System > Logs`. Below is an example of using the logger:

```go
title := "Some Error"
message := "Some error message body"

api.Logger().Error(title, message)
```
