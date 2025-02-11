/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
)

type ClientCreatedHookFn func(ctx context.Context, clnt IClientDevice) error
type ClientChangedHookFn func(ctx context.Context, current IClientDevice, old IClientDevice) error

type IDeviceHooksApi interface {
	ClientCreatedHook(...ClientCreatedHookFn)
	ClientChangedHook(...ClientChangedHookFn)
}
