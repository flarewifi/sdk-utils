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

## Which Alpine.js version runs on the portal and admin?

Core ships a single Alpine major — **v3** — loaded and auto-started on **both** the portal and admin. The full Alpine v3 API (`Alpine.store()` / `$store`, `Alpine.data()`, `x-effect`, the `@click.outside` modifier, adding reactive props after init) works on both surfaces. Both bundles target ES2017 and the app supports modern browsers only, so there are no legacy portal constraints.

Both surfaces use the core-provided Alpine, so don't vendor your own. See [Alpine.js: v3 on both surfaces](../api/assets-manifest.md#alpine-versions).

