/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IPluginApi is the root of all plugin APIs.
type IPluginApi interface {

	// Returns an instance of accounts API.
	Acct() IAccountsApi

	// Returns an instance of ads API.
	Ads() IAdsApi

	// Returns an instance of config API.
	Config() IConfigApi

	// Returns an instance of the client register.
	DeviceHooks() IDeviceHooksApi

	// Returns the root directory of the plugin's installation path.
	Dir() string

	// Features returns a slice of strings representing the features supported by the plugin.
	Features() []string

	// Returns an instance of http API.
	Http() IHttpApi

	// Returns an instance of in-app purchase API.
	InAppPurchases() IInAppPurchasesApi

	// Returns the plugin info which includes the plugin's package, name, version and description
	Info() sdkutils.PluginInfo

	// Returns the logger API
	Logger() ILoggerApi

	// Returns an instance of network API.
	Network() INetworkApi

	// Returns an instance of payments API.
	Payments() IPaymentsApi

	// Returns an instance of the plugin manager.
	PluginsMgr() IPluginsMgrApi

	// Returns the absolute path to the given file in /resources folder of your plugin.
	// For example, if you have the following code:
	//  api.Resource("some-file.txt")
	// then it will return the absolute path to the file "[plugin_root_dir]/resources/some-file.txt" under the plugin's root directory.
	Resource(f string) (path string)

	// Returns an instance of the client manager.
	SessionsMgr() ISessionsMgrApi

	// Returns an instance of database/sql package from go standard library.
	SqlDb() *pgxpool.Pool

	// Returns an instance of the themes API.
	Themes() IThemesApi

	// Translate a message to the user's language.
	Translate(t string, msgk string, pairs ...any) string

	// Returns an instance of the uci API.
	Uci() IUciApi
}
