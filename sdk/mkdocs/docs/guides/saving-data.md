# Saving Data

## Saving Data

To save the plugin data like plugin settings, configuration and statistics, we use the  [IPluginCfgApi.Write](../api/config-api.md#write) method.
Using this API ensures that the user-defined configuration data for your plugin can be migrated properly when a machine owner upgrades or migrate the sytem to another machine.

```go
import "encoding/json"
// ...

// Define custom struct for your plugin settings
type MyPluginConfig struct {
    MySetting       string  `json:"my_setting"`
    OtherSetting    int     `json:"other_setting"`
}

// Create an instance of the struct and assign values
myConfig := MyPluginConfig{
    MySetting:      "my_value",
    OtherSetting:   123,
}

// Convert the values into []bytes
data, err := json.Marshal(myConfig)
if err != nil {
    // handle error
}

// Save the data into a string key "my_key"
err := api.Config().Plugin().Write("my_key", data)
if err != nil {
    // handle error
}
```

Plugin configuration is separated into different keys for ease of management. The data must be serializable to JSON.

## Retreiving Data

To get the plugin data for a specific key, use the [IPluginCfgApi.Read](../api/config-api.md#read) method:
```go
import "encoding/json"
// ...

// Read the data as bytes
data, err := api.Config().Plugin().Read("my_key")
if err != nil {
    // handle error
}

// Assign the data into your struct instance
var myConfig MyPluginConfig
if err := json.Unmarshal(data, &myConfig); err != nil {
    // handle error
}

fmt.Println(myConfig) // {MySetting: "my_value", OtherSetting: 123}
```


