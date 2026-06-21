# Saving Data

## Writing Data

Use the [IPluginCfgApi.Write](../api/config-api.md#write) method to save plugin data such as settings, configuration, and statistics. This API ensures that your plugin's data can be properly migrated when the system is upgraded or moved to new hardware.

```go
import "encoding/json"
// ...

// Define a custom struct for your plugin settings
type MyPluginConfig struct {
    MySetting       string  `json:"my_setting"`
    OtherSetting    int     `json:"other_setting"`
}

// Create an instance and assign values
myConfig := MyPluginConfig{
    MySetting:      "my_value",
    OtherSetting:   123,
}

// Convert to bytes
data, err := json.Marshal(myConfig)
if err != nil {
    // handle error
}

// Save the data using a key
err := api.Config().Plugin().Write("my_key", data)
if err != nil {
    // handle error
}
```

Plugin configuration uses a `string` key to identify data when writing and reading.

## Reading Data

Use the [IPluginCfgApi.Read](../api/config-api.md#read) method to retrieve plugin data for a specific key:

```go
import "encoding/json"
// ...

// Read the data as bytes
data, err := api.Config().Plugin().Read("my_key")
if err != nil {
    // handle error
}

// Unmarshal into your struct
var myConfig MyPluginConfig
if err := json.Unmarshal(data, &myConfig); err != nil {
    // handle error
}

fmt.Println(myConfig) // {MySetting: "my_value", OtherSetting: 123}
```

## Deleting Data

Use the [IPluginCfgApi.Delete](../api/config-api.md#delete) method to remove plugin data for a specific key:

```go
err := api.Config().Plugin().Delete("my_key")
if err != nil {
    // handle error
}
```

This method removes the specified path from the plugin's configuration directory. It works for both individual files and directories with nested contents.

---

## Related

- [IConfigApi](../api/config-api.md) — Complete config API reference: `Plugin()`, `Application()`, `Read`, `Write`, `Delete`
- [Storing Files](./storing-files.md) — For binary files and uploads (images, documents); `IStorageApi` is separate from `IConfigApi`
