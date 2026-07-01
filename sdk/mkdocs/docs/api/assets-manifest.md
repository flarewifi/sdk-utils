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

- `alpinejs-v3.15.1.js` — Alpine **v3**, used by **both admin and portal** assets (all bundles target ES2017).
- `bootstrap-v5.3.3` (CSS + JS) — Bootstrap **5.3.3**, loaded globally by core on **both admin and portal**. See [Bootstrap is a core global](#bootstrap-global).
- `bootstrap-icons-v1.13.1` — Bootstrap Icons **1.13.1**, a core global (admin).
- `htmx-v1.9.12.js` (plus the extensions `htmx-ext-sse-v1.19.12.js` and `htmx-ext-loading-states-v1.19.12.js`)
- `jquery-v3.7.1.js`
- `toastr.js` / `toastr.css`
- `awesome-notifications-v3.1.3.js` / `awesome-notifications-v3.1.3.css`
- Polyfills: `event-source-polyfill.js`, `domparser-polyfill.js`, `xmldom-entities-v0.6.0.js`

Note that core already exposes **jQuery**, **`window.Alpine`**, and **Bootstrap 5** as globals on **both** the portal and admin (it loads and auto-starts Alpine for you), so most plugins and themes can use them directly without importing or vendoring anything.

!!! warning "Do not vendor your own Alpine or Bootstrap"
    Core is the single source of Alpine and Bootstrap. Vendoring your own copy in a plugin or theme **double-loads** the library against the core-provided instance. Use the global `window.Alpine` / plain `x-*` attributes and the global Bootstrap 5 classes/JS instead. Both surfaces run the **same** Alpine major (v3) and the **same** Bootstrap major (5) — see below.

### Bootstrap is a core global {#bootstrap-global}

**Bootstrap 5.3.3 is provided by core as a global asset** — it is auto-loaded on **every** admin and portal page through core's global bundles (`globals.js` / `global.css` and the portal equivalents), exactly like jQuery, htmx, and Alpine. There is a single Bootstrap major everywhere; the old Bootstrap 3 portal path is gone.

Because Bootstrap is global, **plugins and themes must not vendor or reference their own Bootstrap** — no `vendor/bootstrap-*` files, no Bootstrap entries in your manifest, and no `@import`/`require` of Bootstrap. Just use the Bootstrap 5 classes and JS; they are already on the page. Bootstrap Icons 1.13.1 is likewise a core global (admin).

### Customizing the Bootstrap look {#customizing-bootstrap}

You do **not** need Sass to re-theme Bootstrap here (there is no Sass build step in this project — see [Vendoring frontend libraries](#vendoring)). Bootstrap 5 exposes a large library of **CSS custom properties (CSS variables)** at the `:root` level and inside individual components, so you can retheme it from a plain stylesheet — one of your theme/plugin's own manifest CSS bundles, which core loads **after** the global Bootstrap CSS so your rules win.

There are two supported approaches, both plain CSS:

**1. Override Bootstrap's CSS variables (preferred).** Re-declare the `--bs-*` variables. Because Bootstrap's own rules read these variables, overriding them recolors components globally without you having to target each selector:

```css
/* your theme's admin/css/theme.css (a manifest entry) */
:root {
  --bs-primary: #e8590c;
  --bs-primary-rgb: 232, 89, 12;   /* keep the -rgb pair in sync; used by rgba() utilities */
  --bs-body-font-family: "Helvetica Neue", sans-serif;
  --bs-border-radius: 0.5rem;
}

/* Component-scoped variables work too — restyle just one component */
.card {
  --bs-card-bg: #1a1d21;
  --bs-card-border-color: rgba(255, 255, 255, 0.15);
}
```

Dark mode uses the same mechanism: scope variable overrides under `[data-bs-theme="dark"]` (the attribute the admin layout already sets), e.g. `[data-bs-theme="dark"] { --bs-body-bg: #1a1d21; }`.

**2. Traditional CSS overrides.** For anything not exposed as a variable, just write normal rules that target Bootstrap's classes. Rely on load order (your bundle loads after core's global CSS) rather than `!important`:

```css
.btn-primary { text-transform: uppercase; letter-spacing: 0.03em; }
.navbar { box-shadow: 0 1px 0 rgba(0, 0, 0, 0.05); }
```

!!! warning "Don't fork Bootstrap to customize it"
    Re-theme through variables/overrides in **your own** CSS bundle. Re-vendoring or `@import`ing a modified `bootstrap.css` re-introduces the double-load this whole section warns against — and your copy silently drifts from the core version.

Reference: Bootstrap's [CSS variables](https://getbootstrap.com/docs/5.3/customize/css-variables/) and [color modes](https://getbootstrap.com/docs/5.3/customize/color-modes/) docs.

### Alpine.js: v3 on both surfaces {#alpine-versions}

Core ships a single Alpine major — **v3 (`alpinejs-v3.15.1.js`)** — loaded and auto-started on **both** the portal and admin. Both asset bundles compile to **ES2017**, so the full Alpine v3 API (`Alpine.store()` / `$store`, `Alpine.data()`, `x-effect`, `@click.outside`, adding reactive props after init, etc.) works everywhere, including the portal.

The app targets **modern browsers only**, so there are no legacy ES5/IE11 constraints on portal markup or scripts. Just use `window.Alpine` and `x-*` attributes directly.
