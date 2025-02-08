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
