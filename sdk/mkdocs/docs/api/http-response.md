# IHttpResponse
The `IHttpResponse` has utility functions which can be used to send html, json, and file response to the client.

## IHttpResponse methods {#httpresponse-methods}

### PortalView

This method is used to render html using the [portal theme](./themes-api.md#portal-theme) as the layout. Given that you already have a [view template](#template-parsing), the view template can be rendered using:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    name := "John"
    welcomePage := views.WelcomePage(name)
    api.Http().Response().PortalView(w, r, sdkapi.ViewPage{
        PageContent: welcomePage,
    })
}
```

### AdminView

This method is used to render html using the [admin theme](./themes-api.md#admin-theme) as the layout. Given that you already have a [view template](#template-parsing), the view template can be rendered using:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    name := "Admin"
    welcomePage := views.WelcomePage(name)
    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: welcomePage,
    })
}
```

### View

This method is similar to [PortalView](#portalview) and [AdminView](#adminview) but it renders the views **without** using any layout. Therefore you must enclose your [view templates](#template-parsing) with proper `html` tag and document type:

```templ title="plugins/local/com.mydomain.myplugin/resources/views/sample.templ"
package views

templ SamplePage(name string) {
    !DOCTYPE html
    <html>
        <head>
            <title>Sample Page</title>
        </head>
        <body>
            <p>Welcome, { name }</p>
        </body>
    </html>
}
```

```go title="main.go"
// handler
func (w http.ResponseWriter, r *http.Request) {
    name := "Admin"
    samplePage := views.SamplePage(name)
    api.Http().Response().View(w, r, sdkapi.ViewPage{
        PageContent: samplePage,
    })
}
```

### Json

This method is used to send json response to the client. Below is an example of how to send json response:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    data := map[string]string{
        "title": "Dashboard",
    }
    api.Http().Response().Json(w, r, data, http.StatusOK)
}
```

### Redirect

This method is used to redirect a user to another route using the route name as parameter.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    routename := "portal:welcome"
    user := "John"
    api.Http().Response().Redirect(w, r, routename, "name", user)
    // Will redirect to route named "portal:welcome" with GET params name=John
}
```

### FlashMsg

This method is used to set flash messages to the views. But it does not send an HTTP response so you must use redirect or render a view response to show the flash message.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    msg := "Payment successfull!"
    t := sdkapi.FlashMsgSuccess
    api.Http().Response().FlashMsg(w, r, msg, t)
    api.Http().Response().Redirect(w, r, "portal:welcome")
}
```

The available flash message types are:

- `FlashMsgSuccess`
- `FlashMsgInfo`
- `FlashMsgWarning`
- `FlashMsgError`

### RedirectToPortal

This method redirects the user to the portal page.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    api.Http().Response().RedirectToPortal(w, r)
}
```

### RedirectSuccess

This method redirects the user to a custom URL with a visual "Redirecting..." message and a 3-second delay. This provides a smooth user experience for success actions like completed payments, successful registrations, or external redirects.

```go
// handler - redirect to external URL
func (w http.ResponseWriter, r *http.Request) {
    api.Http().Response().RedirectSuccess(w, r, "https://example.com")
}

// handler - redirect to internal route
func (w http.ResponseWriter, r *http.Request) {
    successURL := api.Http().Helpers().UrlForRoute("portal:index")
    api.Http().Response().RedirectSuccess(w, r, successURL)
}
```

**Use cases:**

- Redirecting to external websites after successful actions
- Providing visual feedback before navigation
- Creating smooth transitions between pages
- Post-payment success redirects

**Note:** The redirect page displays a translated "Redirecting..." message with a loading spinner for 3 seconds before navigating to the specified URL.

### Error

This method is used to show consistent error page for unknown errors in your application.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    err := errors.New("Something went wrong!")
    api.Http().Response().Error(w, r, err, http.StatusInternalServerError)
}
```

## View Templates {#template-parsing}

We use [Templ](https://templ.guide) in generating our views. To create a sample view for your plugin, create a file in `resources/views/welcome.templ` with the following contents:

```templ title="plugins/local/com.mydomain.myplugin/resources/views/welcome.templ"
package views

templ WelcomePage(name string) {
    <p>Welcome { name }!</p>
}
```

The SDK runtime will automatically detect the new file and watch for changes. It will then generate
a file called `resources/views/welcome_templ.go` that can be imported and used in rendering [portal views](#portalview) and [admin views](#adminview).

## View Assets

To add assets to a [view template](#template-parsing), you need to register your assets first in the portal or admin [assets manifest](./assets-manifest.md).
After registering your assets in the manifest, you can then use the assets in your views.

For example, given the following portal assets manifest:

```json title="plugins/local/com.mydomain.myplugin/resources/assets/manifest.portal.json"
{
  "index.css": [
    "./portal/portal.css"
  ],
  "index.js": [
    "./portal/portal.js"
  ]
}
```

Then you can render a view together with assets `index.css` and `index.js`:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    name := "John"
    welcomePage := views.WelcomePage(name)
    api.Http().Response().PortalView(w, r, sdkapi.ViewPage{
        Assets: sdkapi.ViewAssets{
            CssFile: "index.css",
            JsFile: "index.js",
        },
        PageContent: welcomePage,
    })
}
```
