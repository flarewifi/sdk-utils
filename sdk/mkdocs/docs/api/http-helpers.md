# IHttpHelpers

`IHttpHelpers` is a set of helper methods that can be used in your http handlers to perform common tasks such as getting the client device, fetching client-side assets, translating messages, and more. It is accessible using the [IHttpApi.Helpers](./http-api.md#helpers) method:

```go
helprs := api.Http().Helpers()
```

## Methods

Below are the methods available in the `IHttpHelpers`:

### AdminAssetPath

Returns the URI path of a manifest index filename in [admin manifest](./assets-manifest.md#admin-manifest).

```go
url := api.Http().Helpers().AdminAssetPath("css/style.css")
```

### PortalAssetPath

Returns the URI path of a manifest index filename in [portal manifest](./assets-manifest.md#portal-manifest).

```go
url := api.Http().Helpers().AdminAssetPath("css/style.css")
```

### PublicPath

Returns the URI path of a static file in `resources/assets/public` directory from your plugin which can be used to link your assets into your views.
For example to get the uri path of the file in `resources/assets/public/images/logo.png`, you can use the following code:

```go
uri := api.Http().Helpers().PublicPath("images/logo.png")
fmt.Println(uri) // Returns "/assets/plugin/your-plugin-id/0.0.1/resources/assets/public/images/logo.png"
```

### AdsView

TODO: implement advertisements feature

### CsrfHtmlTag

Returns the CSRF HTML input tag as plain `string` to be used in HTML forms.

### Translate

Alias to [IPluginApi.Translate](./plugin-api.md#translate) method.

### UrlForRoute

Alias to [IHttpRouterApi.UrlForRoute](./http-router-api.md#urlforroute) method.

### UrlForPkgRoute

Alias to [IHttpRouterApi.UrlForPkgRoute](./http-router-api.md#urlforpkgroute) method.
