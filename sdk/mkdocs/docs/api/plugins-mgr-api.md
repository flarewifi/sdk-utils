# IPluginsMgrApi

The `IPluginsMgrApi` provides methods to manage and access installed plugins in the Flarewifi system. It allows you to find plugins by name or package, and retrieve information about all installed plugins.

To get an instance of `IPluginsMgrApi`:

```go
pluginsMgr := api.PluginsMgr()
fmt.Println(pluginsMgr) // IPluginsMgrApi
```

## IPluginsMgrApi Methods

The following methods are available in `IPluginsMgrApi`:

### FindByName

Finds a plugin by its name as defined in the plugin's `plugin.json` file.

```go
plugin, found := api.PluginsMgr().FindByName("My Plugin")
if found {
    fmt.Printf("Found plugin: %s\n", plugin.Info().Name)
    fmt.Printf("Package: %s\n", plugin.Info().Package)
} else {
    fmt.Println("Plugin not found")
}
```

### FindByPkg

Finds a plugin by its package name as defined in the plugin's `plugin.json` file.

```go
plugin, found := api.PluginsMgr().FindByPkg("com.mydomain.myplugin")
if found {
    fmt.Printf("Found plugin: %s\n", plugin.Info().Name)
    fmt.Printf("Version: %s\n", plugin.Info().Version)
} else {
    fmt.Println("Plugin not found")
}
```

### Plugins

Returns all plugins installed in the system.

```go
allPlugins := api.PluginsMgr().Plugins()

fmt.Printf("Total plugins installed: %d\n", len(allPlugins))

for _, plugin := range allPlugins {
    info := plugin.Info()
    fmt.Printf("- %s (%s) v%s\n", info.Name, info.Package, info.Version)

    // Check plugin features
    features := plugin.Features()
    if len(features) > 0 {
        fmt.Printf("  Features: %v\n", features)
    }
}
```

### InstallPlugin

Installs a plugin from any source (`store`, `git`, `local`/`system`) and registers
it live, without a server restart. The core owns the entire operation — for store
plugins this includes requesting a server-side `.so` build, polling it to
completion, downloading the install-ready tarball, and installing it on the machine.

For a **store** plugin, `InstallPlugin` validates payment up front: if the plugin
is paid and this machine is not purchased to it, it returns `(nil,
ErrPaymentRequired)` **without** starting an install. Detect it with
`errors.Is(err, sdkapi.ErrPaymentRequired)` and redirect the owner to checkout
(see `GetPurchaseURL`). The cloud also withholds the download as an independent
backstop, so this up-front check is not the only gate.

Otherwise `InstallPlugin` returns an `IPluginInstall` **handle** with a `nil`
error; the install runs in the background. The handle exposes:

- `Progress() <-chan PluginInstallProgress` — a stream of stage events, **closed**
  after the install finishes. Sends are best-effort: a slow consumer may miss
  intermediate events, but `Done()` is always authoritative.
- `Done() error` — blocks until the install completes and returns the final error
  (`nil` on success). Safe to call without consuming `Progress()`.

Stages (in order): `resolving` → `queued` → `building` → `downloading` →
`installing` → terminal `done` / `failed`. `git`/`local` installs compile
on-device, so they skip the `queued`/`downloading` stages.

```go
def := sdkutils.PluginSrcDef{
    Src:                sdkutils.PluginSrcStore,
    StorePackage:       "com.example.payment",
    StorePluginVersion: "1.2.0", // or "" for latest
}

h, err := api.PluginsMgr().InstallPlugin(def)
if err != nil {
    if errors.Is(err, sdkapi.ErrPaymentRequired) {
        // Paid plugin not purchased — send the owner to checkout instead.
        return
    }
    api.Logger().Error(fmt.Sprintf("install failed to start: %v", err))
    return
}

// Stream progress (e.g. to drive a progress bar or SSE endpoint).
for ev := range h.Progress() {
    fmt.Printf("[%s] %d%% %s\n", ev.Stage, ev.Percent, ev.Message)
}

// Channel closed — read the authoritative result.
if err := h.Done(); err != nil {
    api.Logger().Error(fmt.Sprintf("install failed: %v", err))
    return
}
fmt.Println("Plugin installed successfully")
```

If you only need the result and not the progress, ignore `Progress()` entirely:

```go
h, err := api.PluginsMgr().InstallPlugin(def)
if err != nil {
    // payment required or another pre-flight failure
}
if err := h.Done(); err != nil {
    // install failed
}
```

The percentages within the build phase are approximate: the cloud build reports
only coarse states (`queued` / `building` / `done`), so the core ramps a synthetic
percent to show forward motion during a long compile.

### UninstallPlugin

Removes a plugin or meta bundle. A regular plugin is marked for removal on the next restart. A meta bundle is detected automatically and its bundle record is removed, cascading to its members: any member owned by no remaining bundle and not installed standalone is also marked for removal on the next restart.

```go
err := api.PluginsMgr().UninstallPlugin("com.example.payment")
if err != nil {
    // handle error
}
fmt.Println("Plugin marked for removal on next restart")
```

### MetaPlugins

Returns all installed meta-plugin bundle records.

```go
bundles, err := api.PluginsMgr().MetaPlugins()
if err != nil {
    // handle error
}
for _, b := range bundles {
    fmt.Printf("Meta bundle: %s\n", b.Package)
}
```

### MetaMembership

Reports which installed meta bundles own a plugin package and whether it should be treated as a standalone install. A plugin installed on its own (not part of any bundle) is standalone. Returns `([]string{}, true)` when the plugins config cannot be read — the safe default.

```go
owners, standalone := api.PluginsMgr().MetaMembership("com.example.payment")
if standalone {
    fmt.Println("Plugin is installed standalone")
} else {
    fmt.Printf("Plugin is owned by bundles: %v\n", owners)
}
```

### IsToBeRemoved

Returns `true` if a plugin has been marked for removal on the next restart.

```go
if api.PluginsMgr().IsToBeRemoved("com.example.payment") {
    fmt.Println("Plugin is scheduled for removal")
}
```

### HasPendingUpdate

Returns `true` if a downloaded update is waiting to be applied for the given plugin.

```go
if api.PluginsMgr().HasPendingUpdate("com.example.payment") {
    fmt.Println("Update available — will apply on next restart")
}
```

### SourceDef

Returns the source definition for an installed plugin — where it came from and how it was installed (`git`, `store`, `system`, or `local`). Returns a zero-value and `false` if the package is not installed.

```go
def, ok := api.PluginsMgr().SourceDef("com.example.payment")
if !ok {
    fmt.Println("Plugin not installed")
    return
}
fmt.Printf("Installed from: %s\n", def.Src)
```

### CheckPurchase

Asks the cloud store whether this machine may install a given plugin and at what
price. Use it to **gate installs** and to **render store UI** ("Free" / a price /
"Purchase required"). It returns a `PluginPurchaseInfo`:

| Field | Meaning |
|-------|---------|
| `Package` | The package that was checked. |
| `Purchased` | `true` when the machine may install it — i.e. the plugin is **free**, already **paid**, or **covered by a meta** bundle. |
| `IsFree` | The plugin carries no price. |
| `PricingType` | `"one_time"` or `"subscription"`. |
| `SubscriptionInterval` | `"monthly"` / `"yearly"` (subscriptions only). |
| `PriceUsdCents` | International price in USD cents. |
| `LocalCurrency` / `LocalPriceCents` | Developer-chosen local price, when set. |
| `DisplayCurrency` / `DisplayPriceCents` | Price resolved into the **machine owner's own currency** (buyer's country → local price if it matches `LocalCurrency`, else USD). This is the amount the checkout will charge — render this for the buyer instead of re-deriving from the USD/local fields. Empty / `0` from an older cloud. |
| `ExpiresAt` | Unix seconds; `0` = none / perpetual. |
| `Available` | `false` when some issue prevents install at all — e.g. the developer has **withdrawn the plugin from the store**. `Reason` explains why. `true` (installable) from an older cloud. |
| `Reason` | Human-readable explanation when the plugin is **not available** or **not purchased**. |

`PluginPurchaseInfo.RequiresPayment()` is the convenience gate — it reports
`!IsFree && !Purchased`, i.e. a paid plugin this machine has not purchased.
**Availability is a separate, prior concern:** an `Available == false` plugin cannot
be installed at any price, so check `Available` **first** — show its `Reason` rather
than a "purchase required" prompt. `InstallPlugin` enforces the same order and
returns `ErrPluginDisabled` (not `ErrPaymentRequired`) for an unavailable package.

> **Note:** This call is for UX. The install path enforces payment independently
> (the cloud withholds the download for unpurchased paid plugins, and
> `InstallPlugin` re-checks up front), so a missing or stale `CheckPurchase`
> result can never let an unpaid install slip through.

```go
info, err := api.PluginsMgr().CheckPurchase("com.example.payment")
if err != nil {
    api.Logger().Error(fmt.Sprintf("purchase check failed: %v", err))
    return
}

switch {
case !info.Available:
    // show "Unavailable" with info.Reason; no install/checkout
case info.IsFree:
    // show an "Install" button
case info.RequiresPayment():
    // show "Purchase required" — link the owner to GetPurchaseURL(...)
default:
    // already purchased/covered — show "Install"
}
```

### GetPurchaseURL

Builds the cloud **checkout URL** for purchasing a plugin. After the machine
owner pays, the cloud redirects the browser back to `callbackRouteName` — a route
registered by the **calling** plugin on this machine — where the plugin should
call `InstallPlugin(pkg)` to complete the purchase.

```go
GetPurchaseURL(r *http.Request, pkg string, callbackRouteName string, pairs ...string) (string, error)
```

- `r` supplies the machine's browser-facing `scheme://host`, so the cloud
  redirect lands back on **this device** (the same host the owner used to reach
  the store), not on loopback. The machine id and cloud checkout host are
  resolved internally.
- `pairs` are forwarded to the callback route exactly like `UrlForRoute`
  (`key, value, key, value, …`) to fill its path parameters.
- The resolved callback URL **always** carries a `?pkg=<pkg>` query param, so the
  callback handler knows what to install regardless of the route's own
  parameters.
- Returns an error if `pkg`/`callbackRouteName` are empty or the callback route
  is not registered. **Render a disabled control rather than a link to an empty
  string when this errors.**

```go
purchaseURL, err := api.PluginsMgr().GetPurchaseURL(
    r,
    "com.example.payment",
    "admin:store:purchase:callback",
)
if err != nil {
    api.Logger().Error(fmt.Sprintf("build purchase url: %v", err))
    // render a disabled "Unavailable" button instead of an empty href
    return
}
// redirect the owner to purchaseURL (or render it as a "Buy Now" link)
```

The callback route then completes the flow:

```go
// GET /store/purchase/callback?pkg=com.example.payment
func PurchaseCallback(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        pkg := r.URL.Query().Get("pkg")
        def := sdkutils.PluginSrcDef{Src: sdkutils.PluginSrcStore, StorePackage: pkg}
        // Purchase is now granted, so InstallPlugin proceeds past the payment gate.
        if _, err := api.PluginsMgr().InstallPlugin(def); err != nil {
            // handle (errors.Is(err, sdkapi.ErrPaymentRequired) should not occur here)
        }
        // redirect to the install/progress page
    }
}
```

## Usage Examples

### Checking if a Plugin is Installed

```go
func isPluginInstalled(api sdkapi.IPluginApi, packageName string) bool {
    _, found := api.PluginsMgr().FindByPkg(packageName)
    return found
}

// Usage
if isPluginInstalled(api, "com.example.payment") {
    fmt.Println("Payment plugin is available")
} else {
    fmt.Println("Payment plugin is not installed")
}
```

### Getting Plugin Information

```go
func getPluginInfo(api sdkapi.IPluginApi, packageName string) {
    plugin, found := api.PluginsMgr().FindByPkg(packageName)
    if !found {
        fmt.Printf("Plugin %s not found\n", packageName)
        return
    }

    info := plugin.Info()
    fmt.Printf("Plugin Name: %s\n", info.Name)
    fmt.Printf("Package: %s\n", info.Package)
    fmt.Printf("Version: %s\n", info.Version)
    fmt.Printf("Description: %s\n", info.Description)
    fmt.Printf("SDK Version: %s\n", info.SDK)

    // System packages required by the plugin
    if len(info.SystemPackages) > 0 {
        fmt.Printf("System Packages: %v\n", info.SystemPackages)
    }
}
```

### Accessing Plugin APIs

Once you have a plugin instance, you can access its APIs:

```go
paymentPlugin, found := api.PluginsMgr().FindByPkg("com.example.payment")
if found {
    // Access the plugin's HTTP API
    httpAPI := paymentPlugin.Http()

    // Access the plugin's configuration
    configAPI := paymentPlugin.Config()

    // Use the plugin's translation function
    message := paymentPlugin.Translate("label", "Payment successful")
    fmt.Println(message)
}
```

## Plugin Discovery

The plugins manager is particularly useful for:

- **Inter-plugin communication**: Allow plugins to discover and interact with each other
- **Feature detection**: Check if specific plugins with certain capabilities are installed
- **Dynamic routing**: Route requests to appropriate plugins based on their capabilities
- **Plugin management**: Build admin interfaces for managing installed plugins</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/plugins-mgr-api.md