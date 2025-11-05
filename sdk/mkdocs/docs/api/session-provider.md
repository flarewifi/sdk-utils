# SessionProvider

A `SessionProvider` is an interface that can be implemented by plugins to provide sessions to [ClientDevices](./client-device.md) from external sources like remote servers. It has the following methods:

Below is the definition of the `SessionProvider` interface:
```go
type SessionProvider interface {

    // Get avaialable session for a client device
	GetSession(ctx context.Context, clnt ClientDevice) (s SessionSource, ok bool)

    // Fetch available sessions for a client device
	FetchSessions(ctx context.Context, clnt ClientDevice, page int, perPage int) (result FetchSessionsResult, err error)
}
```

## 1. SessionProvider Methods {#sessionprovider-methods}

### GetSession

Returns a single [SessionSource](./session-source.md) for the given [ClientDevice](./client-device.md). It accepts a `context.Context` and a [ClientDevice](./client-device.md) parameters and returns a [SessionSource](./session-source.md) and a `bool` indicating if the session was found.

### FetchSessions

Returns a list of [SessionSource](./session-source.md) for the given [ClientDevice](./client-device.md). It accepts a `context.Context`, a [ClientDevice](./client-device.md), the current page and per-page parameters (used for pagination). It returns a [FetchSessionsResult](#fetchsessionsresult) object and an `error` if any.

## 2. FetchSessionsResult {#fetchsessionsresult}

A `FetchSessionsResult` is a struct that contains `Sessions` field which is a list of [SessionSource](./session-source.md), a `Page` field that indicates the number of pages and a `Count` field that indicates the number of sessions found.

Below is the definition of the `FetchSessionsResult` struct:

```go
type FetchSessionsResult struct {
	Sessions []SessionSource
	Pages    uint
	Count    uint
}
```
