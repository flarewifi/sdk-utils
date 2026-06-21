# IConfigApi

The `IConfigApi` contains methods that can be used to modify system configurations such as the language, currency and your custom plugin configuration.

## 1. Application {#application}

The application configuration API has the following fields:

Currency
: The currency used throughout the application. The default is `USD`.

Lang
: The language used throughout the application. The default is `en`. The supported languages are:

- `en` - English
- `am` - Amharic
- `ar` - Arabic (Sudan)
- `es` - Spanish
- `fr` - French
- `id` - Indonesian
- `in` - Hindi
- `prs` - Dari
- `ps` - Pashto
- `ru` - Russian
- `sw` - Swahili

Secret
: The secret key used to sign the JWT tokens and other encryptions.

### Get

To get the application configuration, use the `IAppCfgApi.Get` method.

```go
appCfgAPI := api.Config().Application()
val, err := appCfgAPI.Get()
if err != nil {
    // handle error
}
fmt.Println(val) // {Currentcy: "USD", Lang: "en", Secret: "*****"}
```

### Save

To modify the application configuration, use the `IAppCfgApi.Save` method.

```go
err := appCfgAPI.Save(sdkapi.AppConfig{
    Currency: "USD",
    Lang: "en",
    Secret: "xxxxxxxxxx"
})
```

## 2. Bandwidth {#bandwidth}

Bandwidth configuration is set per interface and has the following fields:

UseGlobal
: Whether to use the global bandwidth configuration. The default is `false`.

GlobalDownMbits
: The global download bandwidth in megabits per second. The default is `2`.

GlobalUpMbits
: The global upload bandwidth in megabits per second. The default is `2`.

UserDownMbits
: The download speed per user in megabits per second. The default is `2`.

UserUpMbits
: The upload speed per user in megabits per second. The default is `2`.

### Get

To get the bandwidth configuration of a network interface, use the `IBandwidthCfgApi.Get` method.

```go
bwdAPI := api.Config().Bandwidth()
cfg, ok := bwdAPI.Get("eth0")
if !ok {
    // handle not found
}
fmt.Println(cfg) // Bandwidth config
```

### Save

To set the bandwidth configuration of a network interface, use the `IBandwidthCfgApi.Save` method.

```go
err := bwdAPI.Save("eth0", sdkapi.IBandwdCfg{
    UseGlobal: true,
    GlobalDownMbits: 2,
    GlobalUpMbits: 2,
    UserDownMbits: 2,
    UserUpMbits: 2,
})
if err != nil {
    // handle error
}
```

## 3. Plugin {#plugin}

The plugin configuration API is used to store custom configuration specific to the plugin you are developing. Using this API ensures that your plugin configuration can be migrated properly to a new system in case you want to flash new firmware or migrate to a new hardware.

The plugin configuration data is stored by `key`. This makes it easier to manage multiple configuration options. The resulting files can be found under your plugin name in the `config/plugins` directory inside the root SDK folder.

### Write

To save your plugin configuration, use the `IPluginCfgApi.Write` method.

```go
import "encoding/json"
// ...

cfgAPI := api.Config().Plugin()

type MyCustomConfig struct {
    Field1 string `json:"field1"`
    Field2 string `json:"field2"`
}

customConfig := MyCustomConfig{
    Field1: "value1",
    Field2: "value2",
}

data, err := json.Marshal(customConfig)
if err != nil {
    // handle error
}

if err := cfgAPI.Write("some_key", data); err != nil {
    // handle error
}
```

### Read

To get your plugin configuration for a specific key, use the `IPluginCfgApi.Read` method.

```go
import "encoding/json"
// ...

cfgAPI := api.Config().Plugin()

data, err := cfgAPI.Read("some_key")
if err != nil {
    // handle error
}

var myCfg MyCustomConfig
if err := json.Unmarshal(data, &myCfg); err != nil {
    // handle error
}

fmt.Println(myCfg) // {Field1: "value1", Field2: "value2"}

fmt.Printf("Field1: %s, Field2: %s", myCfg.Field1, myCfg.Field2)
```

### List

To list all configuration entries under a specific path, use the `IPluginCfgApi.List` method.

```go
cfgAPI := api.Config().Plugin()

entries, err := cfgAPI.List("some/path")
if err != nil {
    // handle error
}

for _, entry := range entries {
    fmt.Printf("Name: %s, Path: %s\n", entry.Entry.Name(), entry.Path)
}
```

### Delete

Use the `IPluginCfgApi.Delete` method to remove a configuration entry.

```go
cfgAPI := api.Config().Plugin()

if err := cfgAPI.Delete("some_key"); err != nil {
    // handle error
}
```

This method removes the specified path from the plugin's configuration directory. It works for both individual files and directories with nested contents.

## 4. Plugins Config {#plugins-config}

The `config-plugins.go` helpers provide low-level access to `data/config/plugins.json`, which stores metadata about installed plugins and meta-plugin bundle records. These are package-level functions, not methods on `IConfigApi`.

### PluginsConfigPath

Returns the absolute filesystem path to `data/config/plugins.json`.

```go
path := sdkapi.PluginsConfigPath()
fmt.Println(path) // /opt/flarehotspot/app/data/config/plugins.json
```

### ReadPluginsConfig

Reads and parses `data/config/plugins.json` into an `sdkutils.PluginsConfig` struct.

```go
cfg, err := sdkapi.ReadPluginsConfig()
if err != nil {
    // handle error
}
fmt.Println(cfg) // sdkutils.PluginsConfig{...}
```

### WritePluginsConfig

Writes an `sdkutils.PluginsConfig` back to `data/config/plugins.json`. Callers should mutate a value obtained from `ReadPluginsConfig` rather than constructing a partial one.

```go
cfg, err := sdkapi.ReadPluginsConfig()
if err != nil {
    // handle error
}

// mutate cfg...

if err := sdkapi.WritePluginsConfig(cfg); err != nil {
    // handle error
}
