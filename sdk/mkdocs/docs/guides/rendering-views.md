# Rendering Views {#rendering-views}

Flarewifi renders HTML with [templ](https://templ.guide) — a templating engine with first-class Go integration: LSP autocompletion, compile-time type checking, and templates that compile to pure Go for speed.

Paired with [htmx](https://htmx.org/), you can build modern, interactive pages without a heavy frontend toolchain. The `htmx` object is always available when you render via [IHttpResponse.PortalView](../api/http-response.md#portalview) or [IHttpResponse.AdminView](../api/http-response.md#adminview).

## Creating a templ template

`Temple` templates must be created in the `resources/views` directory in your plugin. For example, we will create a template called `welcome.templ`:

```templ title="plugins/local/com.mydomain.myplugin/resources/views/welcome.templ"
package views

templ WelcomePage(name string) {
    <p>Welcome, { name } </p>
}
```

The SDK runtime will detect the file and watch for changes. It will then generate a new file based on the template we have created called `welcome_templ.go`:

```go title="plugins/local/com.mydomain.myplugin/resources/views/welcome_templ.go"
package views

func WelcomePage(name string) {
    // rest of the go code base on the templ template
}
```

## Rendering the template

We can then use this go code in our plugin and render it using one of the rendering methods of [IHttpResponse](../api/http-response.md).
In this example, we will use the [IHttpResponse.PortalView](../api/http-response.md#portalview) method to render the template:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    name := "Jhon"
    welcomePage := views.WelcomePage(name)
    api.Http().HttpResponse().PortalView(w, r, sdkapi.ViewPage{
        PageContent: welcomePage,
    })
}
```

This will render the template using the layout provided by the selected [portal theme](../api/themes-api.md#portal-theme). To render templates for the admin web interface, you should use the [IHttpResponse.AdminView](../api/http-response.md#adminview) method.

## Adding assets to views

To add assets to our views like `.js` and `.css` files, first we need to register our assets in the [Assets Manifest](../api/assets-manifest.md). The javascript and css assets must be placed inside `resources/assets` in your plugin directory. Suppose we want to add the following assets to our view:

- `plugins/com.mydomain.myplugin/resources/assets/js/script.js`
- `plugins/com.mydomain.myplugin/resources/assets/css/style.css`

We must add these files into the [portal manifest file](../api/assets-manifest.md#portal-manifest):

```json title="plugins/com.mydomain.myplugin/resources/assets/manifest.portal.json"
{
    "index.css": [
        "./css/style.css"
    ],
    "index.js": [
        "./js/script.js"
    ]
}
```

The path to `script.js` and `style.css` file is relative to the `manifest.portal.json` manifest file. These will then be bundled into `index.css` and `index.js` file respectively.
We can then render these assets together with our template:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    name := "Jhon"
    welcomePage := views.WelcomePage(name)
    api.Http().HttpResponse().PortalView(w, r, sdkapi.ViewPage{
        Assets: sdkapi.ViewAssets{
            CssFile: "index.css",
            JsFile: "index.js",
        },
        PageContent: welcomePage,
    })
}
```

!!! note "Modern browsers"
    The app targets modern browsers only. Both the captive portal and admin asset bundles compile to **ES2017**, so modern JavaScript (arrow functions, template literals, `class`, etc.) is fine on both surfaces.

## Adding global plugin assets

To add a global assets that is available to all the views in your plugin, you can add the assets to the `global.js` and `global.css` keys in the [portal manifest](../api/assets-manifest.md#portal-manifest) or [admin manifest](../api/assets-manifest.md#admin-manifest) file. For example:

```json title="plugins/com.mydomain.myplugin/resources/assets/manifest.portal.json"
{
    "global.js": [
        "./js/global.js"
    ],
    "global.css": [
        "./css/global.css"
    ]
}
```

In this example, the `global.js` and `global.css` assets will be available to all the portal views in the plugin.

To add global assets to the admin web interface, you should add the assets to the `global.js` and `global.css` keys in the [admin manifest](../api/assets-manifest.md#admin-manifest) file.

## Using fonts

Fonts are automatically bundled with your CSS and JS files using the ES Build tool. Just use it as you would in a normal web application. For example, if you have a font file located at `resources/assets/fonts/myfont.woff2` and your css is located in `resources/assets/css/style.css`, you can reference it in your CSS like this:

```css
@font-face {
    font-family: 'MyFont';
    src: url('../fonts/myfont.woff2') format('woff2');
    font-weight: normal;
    font-style: normal;
}
```

## Using public assets

Files in the `resources/assets/public` directory can be referenced using the [`IHttpHelpers.PublicPath`](../api/http-helpers.md#publicpath) method. For example, if you have an image file located at `resources/assets/public/images/logo.png`, you can reference it in your views like this:
```templ
<img src={ api.Http().Helpers().PublicPath("images/logo.png") } alt="Logo">
```

---

## Related

- [IHttpResponse](../api/http-response.md) — `PortalView`, `AdminView`, and other render methods
- [IHttpHelpers](../api/http-helpers.md) — `PublicPath`, `ResourcePath`, and URL generation helpers
- [Assets Manifest](../api/assets-manifest.md) — Bundling JS and CSS assets for portal and admin views
- [IHttpRouterApi](../api/http-router-api.md) — Registering routes that serve these views
