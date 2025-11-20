# IPluginsMgrApi

The `IPluginsMgrApi` provides methods to manage and access installed plugins in the Flare Hotspot system. It allows you to find plugins by name or package, and retrieve information about all installed plugins.

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

### All

Returns all plugins installed in the system.

```go
allPlugins := api.PluginsMgr().All()

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

## Usage Examples

### Checking if a Plugin is Installed

```go
func isPluginInstalled(packageName string) bool {
    _, found := api.PluginsMgr().FindByPkg(packageName)
    return found
}

// Usage
if isPluginInstalled("com.example.payment") {
    fmt.Println("Payment plugin is available")
} else {
    fmt.Println("Payment plugin is not installed")
}
```

### Getting Plugin Information

```go
func getPluginInfo(packageName string) {
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
    message := paymentPlugin.Translate("en", "payment_success", "amount", 29.99)
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