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
completion, downloading the install-ready tarball, and installing it on the device.

`InstallPlugin` returns immediately with an `IPluginInstall` **handle**; the install
runs in the background. The handle exposes:

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

h := api.PluginsMgr().InstallPlugin(def)

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
if err := api.PluginsMgr().InstallPlugin(def).Done(); err != nil {
    // handle failure
}
```

The percentages within the build phase are approximate: the cloud build reports
only coarse states (`queued` / `building` / `done`), so the core ramps a synthetic
percent to show forward motion during a long compile.

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
    message := paymentPlugin.Translate("label", "payment_success")
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