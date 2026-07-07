# IPluginApi

The `IPluginApi` is the root Go interface of Flarewifi SDK. It provides access to methods used to manipulate system accounts, network devices, theme configuration, user sessions, payment system and more. Each plugin is provided with an instance of `IPluginApi`.

When the plugin is first loaded into the system, the system looks for the `Init` function of the plugin's `main` package. The `IPluginApi` instance is then passed to the plugin's `Init` function. From there, you can start configuring the routes and components of your plugin. An example of a plugin's init function:

```go title="plugins/com.mydomain.myplugin/main.go"
package main

import (
	sdkapi "sdk/api"
)

// Required for main package
func main() {}

// Plugin entry point
func Init(api sdkapi.IPluginApi) error {
    // You can start using the SDK here.
    // You can configure your routes, define your plugin components
    // and register items in the portal and admin navigation menu, and more.
    return nil
}
```

## IPluginApi Methods

The following are the available methods in `IPluginApi`.

### Acct

It returns the [IAccountsApi](./accounts-api.md) object which is used to access and modify the system admin accounts.

```go
acct := api.Acct()
fmt.Println(acct) // IAccountsApi
```

### Ads

It returns the [IAdsApi](./ads-api.md) object which is used to create and manage ads.

```go
ads := api.Ads()
fmt.Println(ads) // IAdsApi
```

### Config

It returns the [IConfigApi](./config-api.md) object which is used to access and modify the system configuration.

```go
config := api.Config()
fmt.Println(config) // IConfigApi
```

### Machine

It returns the [IMachineApi](./machine-api.md) object which is used to access machine-specific information and operations.

```go
machine := api.Machine()
fmt.Println(machine) // IMachineApi
```

### Dir

It returns the absolute path of the plugin's installation directory.

```go
dir := api.Dir()
fmt.Println(dir) // "/path/to/com.mydomain.myplugin"
```

### Events

It returns the [IEventsApi](./events-api.md) object which is used to react to session lifecycle, client, purchase, and voucher events.

```go
events := api.Events()
fmt.Println(events) // IEventsApi
```

### Firewall

It returns the [IFirewallAPI](./firewall-api.md) object which is used to manage firewall rules.

```go
firewall := api.Firewall()
fmt.Println(firewall) // IFirewallAPI
```

### Features

Returns the available features of the plugin.

```go
features := api.Features()
fmt.Println(features) // []string{"theme:admin", "theme:portal"}
```

Below are the available features and their descriptions:

| Feature | Description |
| --- | --- |
| `theme:admin` | Plugin provides an admin theme
| `theme:portal` | Plugin provides a portal theme

### Http

It returns the [IHttpApi](./http-api.md) object which is used to configure routes and serve HTTP requests.

```go
http := api.Http()
fmt.Println(http) // IHttpApi
```

### InAppPurchases

It returns the [IInAppPurchasesApi](./inapp-purchases-api.md) object which is used to create and manage in-app purchases.

```go
inAppPurchases := api.InAppPurchases()
fmt.Println(inAppPurchases) // IInAppPurchasesApi
```

### Info

It returns the [sdkutils.PluginInfo](../api/plugin-info.md) field defined in [plugin.json](./plugin.json.md).

```go
info := api.Info()
fmt.Println(info)
//  {
//    Name: "My Plugin",
//    Package: "com.mydomain.myplugin",
//    Version: "0.0.1",
//    Description: "My plugin description",
//    SystemPackages: [],
//    SDK: "1.0.0"
//  }
```

### Logger

It returns the [ILoggerAPI](./logger-api.md) object which is used to log events in a plugin.

### Network

It returns the [INetworkApi](./network-api.md) object which is used to manage the network.

```go
network := api.Network()
fmt.Println(network) // INetworkApi
```

### Notification

It returns the [INotificationAPI](./notification.md) object which is used to send notifications to users.

```go
notification := api.Notification()
fmt.Println(notification) // INotificationAPI
```

### Payments

It return the [IPaymentsApi](./payments-api.md) object which is used to create payment options or create system transactions.

```go
payments := api.Payments()
fmt.Println(payments) // IPaymentsApi
```

### PluginsMgr

It returns the [IPluginsMgrApi](./plugins-mgr-api.md) object which is used to manage plugins.

```go
pluginsMgr := api.PluginsMgr()
fmt.Println(pluginsMgr) // IPluginsMgrApi
```

### Resource

It returns the absolute path of the file under the plugin's resource directory.

```go
resource := api.Resource("/my-resource.txt")
fmt.Println(resource) // "/path/to/com.mydomain.myplugin/resources/my-resource.txt"
```

### Scheduler

It returns the [ISchedulerApi](./scheduler-api.md) object which is used to run long-running or periodic background work that stops cleanly on shutdown.

```go
scheduler := api.Scheduler()
fmt.Println(scheduler) // ISchedulerApi
```

### SessionsMgr

It returns the [ISessionsMgrApi](./sessions-mgr-api.md) object which is used to manage user sessions.

### Storage

It returns the [IStorageApi](./storage-api.md) object which is used to store and retrieve files in the plugin's storage directory.

```go
storage := api.Storage()
fmt.Println(storage) // IStorageApi
```

```go
sessionsMgr := api.SessionsMgr()
fmt.Println(sessionsMgr) // ISessionsMgrApi
```

### SqlDB

It returns [\*sql.DB](http://go-database-sql.org/overview.html) instance which is used to query, insert, update and delete database entities.

```go
db := api.SqlDB()
fmt.Println(db) // *sql.DB
```

### Themes

It returns the [`IThemesApi`](./themes-api.md) object which is used to manage system UI themes.

```go
themes := api.Themes()
fmt.Println(themes) // IThemesApi
```

### Translate

It is a utility function used to convert an English source text into a translated string. Example usage:

```go
msg := api.Translate("info", "Payment received: USD <% .amount %>", "amount", 1.00)
fmt.Println(msg) // "Payment received: USD 1.00"
```

Given that the [application](../api/config-api.md#application) language is set to `en`, the system looks up the source text in the per-language catalog `resources/translations/en.json` inside your plugin directory — a single JSON file keyed by message type, then by the English source text. If a translation is found it is used as the template; otherwise the English source text you passed is used as-is. See the [Translations guide](../guides/translations.md) for the full catalog format.

Sometimes we want to put variables inside the translation message. In this example, we pass the `amount` as a parameter to the message by appending key-value pairs to the `Translate` method. Internally, the param pairs are converted into a type `map[any]any`. To use the `amount` param in the message, enclose it with `<%` and `%>` delimiters (with a dot prefix). The catalog entry in `en.json` therefore looks like:

```json
{
  "info": {
    "Payment received: USD <% .amount %>": "Payment received: USD <% .amount %>"
  }
}
```

### Uci

It returns the [IUciApi](./uci-api.md) object which is a wrapper to [OpenWRT's UCI](https://openwrt.org/docs/guide-user/base-system/uci).

```go
uci := api.Uci()
fmt.Println(uci) // IUciApi
```

### UI

It returns the [IUIApi](./ui-api.md) object which is used for ui reusable templates.

```go
ui := api.UI()
fmt.Println(ui) // IUIApi
```

### Vouchers

It returns the [IVouchersApi](./voucher-api.md) object which is used to create and manage vouchers.

```go
vouchers := api.Vouchers()
fmt.Println(vouchers) // IVouchersApi
```

### Wifi

It returns the [IWifiApi](./wifi-api.md) object which is used to listen for WiFi client connect/disconnect events.

```go
wifi := api.Wifi()
fmt.Println(wifi) // IWifiApi
```
