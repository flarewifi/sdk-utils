# Assets Manifest {#assets-manifest}

## About Assets Manifest

Asset manifests are declaration of assets to be bundled into a given filename index. There are two manifest files:

- `manifest.portal.json` - Manifest file for portal assets
- `manifest.admin.json` - Manifest file for admin assets

## Portal manifest file {#portal-manifest}

The portal assets manifest file must be located in `resources/assets/manifest.portal.json` inside your plugin directory. An example of portal assets manifest file:

```json
{
  "index.css": [
    "./portal/portal.css",
    "./portal/another-file.css"
  ],
  "index.js": [
    "./portal/portal.js",
    "./portal/another-file.js"
  ]
}
```

In this example, the files `./portal/portal.css` and `./portal/anoter-file.css` are relative to `manifest.portal.json` file. These files will be bundled into `index.css` file which you can then reference in [view assets](./http-response.md#view-assets).

Likewise, the files `./portal/portal.js` and `./portal/another-file.js` will be bundled into `index.js` and can be rendered in the views.

The `index.css` and `index.js` can be used with any views that is rendered using the [IHttpResponse.PortalView](../api/http-response.md#portalview) method.

## Admin manifest file {#admin-manifest}

The admin assets manifest file must be located in `resources/assets/manifest.admin.json` inside your plugin directory. An example of admin assets manifest file:

```json
{
  "index.css": [
    "./admin/admin.css",
    "./admin/another-file.css"
  ],
  "index.js": [
    "./admin/portal.js",
    "./admin/another-file.js"
  ]
}
```

The files `./admin/admin.css` and other files in the list are relative to `manifest.admin.json` file.

Similar to [portal assets manifest](#portal-manifest), `index.css` and `index.js` files are a bundle of assets which can be [rendered in the views](./http-response.md#view-assets).

The `index.css` and `index.js` can be used with any views that is rendered using the [IHttpResponse.AdminView](../api/http-response.md#adminview) method.

## Global assets {#global-assets}

To add global assets to either portal or admin views of your plugin, you can add the assets to the `global.js` and `global.css` keys in the [portal manifest](#portal-manifest) or [admin-manifest](#admin-manifest) file. For example:
```json
{
  "global.js": [
    "./global/global.js",
    "./global/another-file.js"
  ],
  "global.css": [
    "./global/global.css",
    "./global/another-file.css"
  ]
}
```

## Vendoring frontend libraries (no npm) {#vendoring}

Flarewifi bundles plugin assets with a **Go-native esbuild** and **does not run `npm install`** — there is no `node`/`node_modules` on the machine (the apk OpenWRT feeds no longer ship a `node` binary, and on-device plugin recompiles must still work). The bundler only resolves **real files in your source tree**. A bare module specifier therefore **fails the asset build**:

```js
// ❌ Fails the build: esbuild "could not resolve 'jquery'"
var $ = require("jquery");
import Alpine from "alpinejs";
```

To use a third-party library, **vendor** its browser/ESM dist file into your plugin and import it by **relative path**.

### How to vendor a library {#vendor-how-to}

1. Download the library's browser/ESM build (the single `.js`/`.css` dist file, not the npm package) into your plugin's `resources/assets/lib/vendor/` directory.
2. Name it `<libname>-v<version>.<ext>` so the version is visible in source, e.g. `jquery-v3.7.1.js`.
3. Import or `require()` it by a path **relative to the importing file**.

For example, a plugin admin script at `resources/assets/admin/js/layout.js` reaches `resources/assets/lib/vendor/jquery-v3.7.1.js` by going up two directories:

```js
// before — bare specifier, fails the build
var $ = require("jquery");

// after — relative path to the vendored file
var $ = require("../../lib/vendor/jquery-v3.7.1.js");
```

Always compute the `../` prefix from the location of the file doing the import.

CSS libraries vendor exactly the same way: drop the `.css` dist into `resources/assets/lib/vendor/` and list it (or `@import` it by relative path) in your CSS manifest. esbuild handles fonts and images referenced by the CSS through its file loaders, so no extra tooling is needed.

### Shared libraries provided by core {#shared-libs}

Common libraries are already vendored by core, so you do **not** need to copy them into every plugin. They live in `core/resources/assets/lib/vendor/` and are reachable through the esbuild alias **`@flare/lib`** → `core/resources/assets/lib`. Import a shared file via `@flare/lib/vendor/<file>`:

```js
import Alpine from "@flare/lib/vendor/alpinejs-v3.15.1.js";
```

Libraries currently vendored under `@flare/lib/vendor`:

- `alpinejs-v3.15.1.js` — Alpine **v3**, used by **admin** assets (admin bundles target ES2017). See [Alpine.js: portal v2 vs admin v3](#alpine-versions).
- `alpinejs-ie11-v2.8.2.js` — Alpine **v2** (IE11 build), used by **portal** assets (portal bundles target ES5). See [Alpine.js: portal v2 vs admin v3](#alpine-versions).
- `htmx-v1.9.12.js` (plus the extensions `htmx-ext-sse-v1.19.12.js` and `htmx-ext-loading-states-v1.19.12.js`)
- `jquery-v1.12.4.js`
- `toastr.js` / `toastr.css`
- `awesome-notifications-v3.1.3.js` / `awesome-notifications-v3.1.3.css`
- Polyfills: `event-source-polyfill.js`, `domparser-polyfill.js`, `xmldom-entities-v0.6.0.js`

Note that core already exposes **jQuery** and **`window.Alpine`** as globals on **both** the portal and admin (it loads and auto-starts Alpine for you), so most plugins and themes can use Alpine directly without importing or vendoring anything.

!!! warning "Do not vendor your own Alpine"
    Core is the single source of Alpine. Vendoring your own copy in a plugin or theme **double-loads** Alpine against the core-provided instance (and may load the wrong major). Use the global `window.Alpine` / plain `x-*` attributes instead. The portal and admin run **different Alpine majors** — see below.

### Portal JavaScript stays ES5 {#vendor-es5}

The portal runs on older client browsers, so **portal JavaScript must remain ES5** (`var`, `function () {}`, no arrow functions / template literals) — this applies to vendored portal libraries too. Pick the ES5-compatible dist of any library you vendor for the portal. Admin assets are not subject to this restriction (admin bundles target ES2017).

### Alpine.js: portal v2 vs admin v3 {#alpine-versions}

Core ships **two different majors** of Alpine, one per surface, because the portal and admin asset bundles compile to different JavaScript targets:

| Surface | Alpine version | esbuild target | Why |
|---------|----------------|----------------|-----|
| **Admin** | v3 (`alpinejs-v3.15.1.js`) | ES2017 | Operator-facing; runs on modern browsers |
| **Portal** | **v2** (`alpinejs-ie11-v2.8.2.js`) | ES5 | Client-device-facing; must run on old captive-portal WebViews |

**Why the portal can't use Alpine v3:** Alpine v3's reactivity is built on the JavaScript `Proxy`, which **cannot be polyfilled** for engines that predate it (e.g. the Android 4.x WebView / Chromium &lt;49 that still show up on budget client phones). Alpine v2's official IE11 build ships a `defineProperty`-based Proxy polyfill and is already ES5, so it works on those devices and bundles cleanly at the portal's ES5 target. Alpine v3 has no ES5 path.

Both are loaded and auto-started by core, so you normally just use `window.Alpine` and `x-*` attributes. But **portal markup and scripts must stay within the Alpine v2 API**:

!!! warning "Portal Alpine is v2 — API constraints"
    - **Declare every `x-data` property up-front.** The v2 Proxy polyfill cannot make properties reactive if they are added *after* the component is created.
    - **No v3-only features** in portal code: `Alpine.store()` / `$store`, `Alpine.data()`, `x-effect`, `x-id`, `x-teleport`, `x-modelable`.
    - **Use `@click.away`, not `@click.outside`** (`.outside` and `.self` are v3-only modifiers).
    - Array mutations that don't reassign the variable (`push`, `pop`, index assignment) may not be detected — reassign the array to trigger updates.

    The **admin** surface is on v3 and has none of these restrictions — `Alpine.store`, `$store`, `x-effect`, etc. are all available there.
