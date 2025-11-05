# SessionSource

## 1. SessionSource Interface {#session-source}

A `SessionSource` represents a source of data for a session. It is used to create a session from external sources like remote servers.
Below is the definition of the `SessionSource` interface:

```go
type SessionSource interface {

	// Return the session data.
	Data() SessionData

	// Save data to the source, e.g. database.
	Save(context.Context, SessionData) error

	// Reload data from the source, e.g. database.
	Reload(context.Context) (SessionData, error)
}
```

Below is the description of each method:

### Data

The `Data` method returns the session data. It returns a [SessionData](#session-data) struct that contains the session data.

### Save

The `Save` method saves the session data to the source. It accepts a `context.Context` and a [SessionData](#session-data) struct as parameters. It returns an error if any.

### Reload

The `Reload` method reloads the session data from the source. It accepts a `context.Context` as a parameter. It returns a newly loaded [SessionData](#session-data) struct and an error if any.

## 2. SessionData Struct {#session-data}

A `SessionData` struct represents the data of a session. It contains the following fields:

```go
type SessionData struct {
	Provider       string
	Type           uint8
	TimeSecs       uint
	DataMb         float64
	TimeCons       uint
	DataCons       float64
	StartedAt      *time.Time
	ExpDays        *uint
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
	CreatedAt      time.Time
}
```

Below is the description of each field:

### Provider

The name of the provider of the session. It is a string value that can be used to identify the provider, e.g. plugin name.

### Type

The type of the session. It is an unsigned 8-bit integer value that can be used to identify the type of the session. See the [session types](./client-session.md#type) documentation.

### TimeSecs

The total time in seconds of the session. This is only applicable to session types `time (0)` and `time_or_data (2)`.

### DataMb

The total data in megabytes of the session. This is only applicable to session types `data (1)` and `time_or_data (2)`.

### TimeCons

The total time in seconds consumed by the session. This is only applicable to session types `time (0)` and `time_or_data (2)`. It is used to track the consumed time of the session.

### DataCons

The total data in megabytes consumed by the session. This is only applicable to session types `data (1)` and `time_or_data (2)`. It is used to track the consumed data of the session.

### StartedAt

The time when the session was started. It is a pointer to a `time.Time` value. A `nil` value indicates that the session has not started yet.

### ExpDays

The number of days the session is valid from the time that it is started plus the [TimeSecs](#timesecs) value. It is a pointer to an unsigned integer value. A `nil` value indicates that the session does not expire.

### DownMbits

The download speed of the session in megabits per second (mbps).

### UpMbits

The upload speed of the session in megabits per second (mbps).

### UseGlobalSpeed

Used to determine if the session should use the [global](./config-api.md#bandwidth) download and upload speed limit. If `true`, it ignores the download and upload speed arguments.

### CreatedAt

This is the time when the session was created. It is a `time.Time` value that represents the time the session was created.
