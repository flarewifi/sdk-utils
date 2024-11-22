/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkplugin

import (
	sdkacct "sdk/api/accounts"
	sdkads "sdk/api/ads"
	sdkcfg "sdk/api/config"
	sdkconnmgr "sdk/api/connmgr"
	sdkhttp "sdk/api/http"
	sdkinappur "sdk/api/inappur"
	sdklogger "sdk/api/logger"
	sdknet "sdk/api/network"
	sdkpayments "sdk/api/payments"
	sdkuci "sdk/api/uci"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IPluginApi is the root of all plugin APIs.
type IPluginApi interface {

	// Returns the package name of the plugin as defined in package.yml "package" field.
	Pkg() string

	// Returns the name of the plugin as defined in package.yml "name" field.
	Name() string

	// Returns the version of the plugin as defined in package.yml "version" field.
	Version() string

	// Returns the description of plugin.
	Description() string

	// Returns the root directory of the plugin's installation path.
	Dir() string

	// Translate a message to the user's language.
	Translate(t string, msgk string, pairs ...any) string

	// Returns the absolute path to the given file in /resources folder of your plugin.
	// For example, if you have the following code:
	//  api.Resource("some-file.txt")
	// then it will return the absolute path to the file "[plugin_root_dir]/resources/some-file.txt" under the plugin's root directory.
	Resource(f string) (path string)

	// Returns an instance of database/sql package from go standard library.
	SqlDb() *pgxpool.Pool

	// Run the plugin migration scripts in resources/migrations folder.
	Migrate() error

	// Returns an instance of accounts api.
	Acct() sdkacct.AccountsApi

	// Returns an instance of http api.
	Http() sdkhttp.IHttpApi

	// Returns an instance of config api.
	Config() sdkcfg.IConfigApi

	// Returns an instance of payments api.
	Payments() sdkpayments.IPaymentsApi

	// Returns an instance of network api.
	Network() sdknet.INetworkApi

	// Returns an instance of ads api.
	Ads() sdkads.IAdsApi

	// Returns an instance of in-app purchase api.
	InAppPurchases() sdkinappur.IInAppPurchasesApi

	// Returns an instance of the plugin manager.
	PluginsMgr() IPluginsMgrApi

	// Returns an instance of the client register.
	DeviceHooks() sdkconnmgr.IDeviceHooksApi

	// Returns an instance of the client manager.
	SessionsMgr() sdkconnmgr.ISessionsMgrApi

	// Returns an instance of the uci api.
	Uci() sdkuci.IUciApi

	// Returns an instance of the themes api.
	Themes() sdkhttp.IHttpThemesApi

	// Features returns a slice of strings representing the features supported by the plugin.
	Features() []string

	Logger() sdklogger.ILoggerApi
}
