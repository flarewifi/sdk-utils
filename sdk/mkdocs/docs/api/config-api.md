# IConfigApi

The `IConfigApi` contains methods that can be used to modify system configurations such as the language, currency and your custom plugin configuration.

## 1. Application {#application}

The application configuration API has the following fields:

Currency
: The currency used throughout the application. The default is `USD`.

Lang
: The language used throughout the application. The default is `en`. The supported languages are:

- `en` - English
- `es` - Spanish
- `id` - Indonesian
- `ms` - Malay

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
err := appCfgAPI.Save(sdkapi.AppCfg{
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
cfg, err := bwdAPI.Get("eth0")
if err != nil {
    // handle error
}
fmt.Println(cfg) // Bandwidth config
```

### Save

To set the bandwidth configuration of a network interface, use the `IBandwidthCfgApi.Save` method.

```go
err := bwdAPI.Save("eth0", sdkapi.BandwdCfg{
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
