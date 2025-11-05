/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

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
