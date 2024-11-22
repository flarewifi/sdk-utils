package sdkconnmgr

import "context"

type FetchSessionsResult struct {
	Sessions []ISessionSource
	Pages    uint
	Count    uint
}

type ISessionProvider interface {

	// Get avaialable session for a client device
	GetSession(ctx context.Context, clnt IClientDevice) (s ISessionSource, ok bool)

	// Fetch available sessions for a client device
	FetchSessions(ctx context.Context, clnt IClientDevice, page int, perPage int) (result FetchSessionsResult, err error)
}
