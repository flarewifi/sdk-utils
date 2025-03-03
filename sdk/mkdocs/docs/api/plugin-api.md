# IPluginApi

The `IPluginApi` is the root Go interface of Flare Hotspot SDK. It provides access to methods used to manipulate system accounts, network devices, theme configuration, user sessions, payment system and more. Each plugin is provided with an instance of `IPluginApi`.

When the plugin is first loaded into the system, the system looks for the `Init` function of the plugin's `main` package. The `IPluginApi` instance is then passed to the plugin's `Init` function. From there, you can start configuring the routes and components of your plugin. An example of a plugin's init function:

```go title="plugins/com.mydomain.myplugin/main.go"
package main

import (
	sdkapi "sdk/api"
)

// Required for main package
func main() {}

// Plugin entry point
func Init(api sdkapi.IPluginApi) {
    // You can start using the SDK here.
    // You can configure your routes, define your plugin components
    // and register items in the portal and admin navigation menu, and more.
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

### DeviceHooks

It returns the [IDeviceHooksApi](./device-hooks-api.md) object which is used to manage device registration hooks.

```go
deviceHooks := api.DeviceHooks()
fmt.Println(deviceHooks) // IDeviceHooksApi
```

### Dir

It returns the absolute path of the plugin's installation directory.

```go
dir := api.Dir()
fmt.Println(dir) // "/path/to/com.mydomain.myplugin"
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

It returns the [IInAppPurchasesApi](./in-app-purchases-api.md) object which is used to create and manage in-app purchases.

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

It returns the [INetworkApi](../network-api/) object which is used to manage the network.

```go
network := api.Network()
fmt.Println(network) // INetworkApi
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

### SessionsMgr

It returns the [ISessionsMgrApi](./sessions-mgr-api.md) object which is used to manage user sessions.

```go
sessionsMgr := api.SessionsMgr()
fmt.Println(sessionsMgr) // ISessionsMgrApi
```

### SqlDb

It returns [\*sql.DB](http://go-database-sql.org/overview.html) instance which is used to query, insert, update and delete database entities.

```go
db := api.SqlDb()
fmt.Println(db) // *sql.DB
```

### Themes

It returns the [`IThemesApi`](./themes-api.md) object which is used to manage system UI themes.

```go
themes := api.Themes()
fmt.Println(themes) // IThemesApi
```

### Translate

It is a utility function used to convert a message key into a translated string. Example usage:

```go
msg := api.Translate("info", "payment_received", "amount", 1.00)
fmt.Println(msg) // "Payment received USD 1.0.0"
```

In this example, given that the [application](../api/config-api.md#application) language is set to `en`, the system will look for the file `resources/translations/en/info/payment_received.txt` inside your plugin directory. If the file is found, the system will use the contents of the file as the translation template.

Sometimes we want to put variables inside the translation message. In this example, we want to pass the `amount` as a parameter to the message. We can do that by passing the amount param as key-value pairs to the `Translate` method. Internally, the param pairs are converted into a type `map[any]any`. To use the `amount` param in the translation file, we'll enclose it with `<%` and `%>` delimiters (with dot prefix). Therefore the content of `payment_received.txt` should be:

```go
Payment received: USD <% .amount %>
```

### Uci

It returns the [IUciApi](./uci-api.md) object which is a wrapper to [OpenWRT's UCI](https://openwrt.org/docs/guide-user/base-system/uci).

```go
uci := api.Uci()
fmt.Println(uci) // IUciApi
```
