# Frequently Ask Questions

## How to get client mac address?
To get the client device's MAC address, you can use the [HttpApi.GetClientDevice](../api/http-api.md#getclientdevice) method in your handler:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, err := api.Http().GetClientDevice(r)
    // handle error
    fmt.Println(clnt.MacAddr()) // print the mac address
}
```

## Why does my portal Alpine.js code behave differently from admin?

Core ships **two different Alpine majors**: the **portal** runs **Alpine v2** (an ES5/IE11 build that runs on old client-device WebViews), while the **admin** runs **Alpine v3**. So v3-only features — `Alpine.store()` / `$store`, `Alpine.data()`, `x-effect`, the `@click.outside` modifier — work in admin but **not** in the portal. In portal markup, declare all `x-data` properties up-front and use `@click.away` instead of `@click.outside`.

Both are loaded and auto-started by core, so don't vendor your own Alpine. See [Alpine.js: portal v2 vs admin v3](../api/assets-manifest.md#alpine-versions) for the full list of constraints.

