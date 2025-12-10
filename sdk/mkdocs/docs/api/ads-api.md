# IAdsApi

The `IAdsApi` is used for displaying advertisements in the captive portal. It allows plugins to integrate third-party ad services (such as Google AdSense) into the portal pages.

To get an instance of `IAdsApi`:

```go
adsAPI := api.Ads()
fmt.Println(adsAPI) // IAdsApi
```

## IAdsApi Methods

### Init

Initializes the ads API with the given app ID from your ad provider. This should be called during plugin initialization to configure the ad service.

```go
func Init(api sdkapi.IPluginApi) error {
    // Initialize ads with your AdSense app ID
    api.Ads().Init("ca-pub-1234567890123456")
    return nil
}
```

## Usage Example

### Setting Up Google AdSense

```go
package main

import (
    sdkapi "sdk/api"
)

func Init(api sdkapi.IPluginApi) error {
    // Initialize the ads API with your AdSense publisher ID
    api.Ads().Init("ca-pub-1234567890123456")
    return nil
}
```

## Integration Notes

- The `Init` method should be called once during plugin initialization
- The app ID format depends on your ad provider (e.g., Google AdSense uses `ca-pub-XXXXXXXXXX`)
- Ads will be displayed in designated ad slots within the captive portal
- Ensure you comply with your ad provider's terms of service

## Ad Provider Configuration

| Provider | App ID Format |
| --- | --- |
| Google AdSense | `ca-pub-XXXXXXXXXX` |

## Best Practices

- **Call Init early**: Initialize the ads API during your plugin's `Init` function
- **Handle ad blockers gracefully**: The portal should still function if ads fail to load
- **Follow ad policies**: Ensure your implementation complies with the ad provider's policies
- **Test thoroughly**: Verify ads display correctly across different devices and browsers
