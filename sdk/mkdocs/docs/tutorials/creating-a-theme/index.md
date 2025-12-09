# Creating a Theme Plugin

This tutorial will guide you through creating a custom theme plugin for Flare Hotspot. Theme plugins allow you to customize the appearance and layout of both the admin interface and the user portal.

## Prerequisites

- Basic knowledge of Go programming
- Understanding of HTML, CSS, and JavaScript
- Familiarity with the [templ](https://templ.guide/) template engine
- A Flare Hotspot development environment set up

## Step 1: Create the Basic Plugin Structure

First, follow the [general plugin creation guide](../guides/creating-a-plugin.md) to create a new plugin using the `create-plugin` command. This will set up the basic plugin structure, `main.go` file, and necessary configuration.

After creating the basic plugin, continue with the theme-specific implementation below.

## Theme Plugin Structure

A theme plugin follows this structure:

```
your-theme-plugin/
├── main.go
├── plugin.json
├── go.mod
├── go.sum
├── app/
│   ├── routes.go
│   └── themes/
│       ├── admin.go
│       └── portal.go
├── resources/
│   ├── assets/
│   │   ├── admin/
│   │   │   ├── css/
│   │   │   └── js/
│   │   ├── portal/
│   │   │   ├── css/
│   │   │   └── js/
│   │   ├── auth/
│   │   └── manifest.*.json
│   ├── translations/
│   └── views/
│       ├── admin/
│       ├── auth/
│       └── portal/
└── package.json (optional, for asset building)
```

## Step 2: Update Plugin Configuration

Update the `plugin.json` created by the `create-plugin` command to reflect that this is a theme plugin:

```json
{
  "name": "My Custom Theme",
  "package": "com.example.my-theme",
  "version": "1.0.0",
  "description": "A custom theme for Flare Hotspot"
}
```

## Step 3: Update the Go Module

The `go.mod` file should already be created by the `create-plugin` command. Update it to include the templ dependency for template compilation:

```go
module com.example.my-theme

go 1.21

toolchain go1.21.13

require (
    github.com/a-h/templ v0.2.793
)
```

!!! note
    The SDK dependency is not included in go.mod because plugins are loaded dynamically by the core system, which provides the SDK API at runtime.

## Step 4: Implement the Main Plugin File

Update the `main.go` file created by the `create-plugin` command to include theme initialization:

```go
//go:build !mono

package main

import (
    "net/http"

    sdkapi "sdk/api"

    "com.example.my-theme/app"
    "com.example.my-theme/app/themes"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    app.SetupRoutes(api)
    themes.SetPortalTheme(api)
    themes.SetAdminTheme(api)
    return nil
}
```

## Step 5: Set Up Routes (Optional)

Create the `app` directory and `app/routes.go` file for any custom routes your theme might need:

```go
package app

import (
    sdkapi "sdk/api"
)

func SetupRoutes(api sdkapi.IPluginApi) {
    // Register any custom routes here
    // api.Http().Router().HandleFunc("/custom-route", handler)
}
```

## Step 6: Implement the Admin Theme

Create the `app/themes` directory and `app/themes/admin.go` file:

```go
package themes

import (
    "fmt"
    "net/http"
    sdkapi "sdk/api"

    "com.example.my-theme/resources/views/admin"
)

func SetAdminTheme(api sdkapi.IPluginApi) {
    api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
        JsFile:  "theme.js",
        CssFile: "theme.css",
        CssLib:  sdkapi.CssLibBootstrap5, // Bootstrap 5 is the only supported admin CSS library
        LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
            // Get navigation items
            navs := api.Http().Navs().GetAdminNavs(r)
            var navItems []sdkapi.AdminNavItem
            for _, nav := range navs {
                navItems = append(navItems, nav.Items...)
            }

            // Get notifications
            notifs, err := api.Notification().GetUnreadNotifications(r.Context())
            if err != nil {
                notifs = []sdkapi.Notification{}
            }

            // Create layout data
            data := admin.AdminLayoutData{
                Components:    c,
                Navs:          navs,
                NavItems:      navItems,
                Notifications: notifs,
            }

            // Render layout
            layout := admin.AdminLayout(api, data)
            if err := layout.Render(r.Context(), w); err != nil {
                fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
            }
        },
        IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
            // Implement your admin dashboard page
            page := admin.AdminIndexPage(api, nil)
            return sdkapi.ViewPage{PageContent: page}
        },
    })
}
```

## Step 7: Implement the Portal Theme

Create `app/themes/portal.go`:

```go
package themes

import (
    "fmt"
    "net/http"
    sdkapi "sdk/api"

    "com.example.my-theme/resources/views/auth"
    "com.example.my-theme/resources/views/portal"
)

func SetPortalTheme(api sdkapi.IPluginApi) {
    api.Themes().NewPortalTheme(sdkapi.PortalThemeOpts{
        JsFile:  "theme.js",
        CssFile: "theme.css",
        LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
            data := portal.PortalLayoutData{Components: c}
            layout := portal.PortalLayout(data)
            if err := layout.Render(r.Context(), w); err != nil {
                fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
            }
        },
        LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
            csrfHtml := api.Http().Helpers().CsrfHtmlTag(r)
            page := auth.LoginPage(api, csrfHtml, data)
            return sdkapi.ViewPage{
                Assets: sdkapi.ViewAssets{
                    JsFile:  "auth/login.js",
                    CssFile: "auth/login.css",
                },
                PageContent: page,
            }
        },
        IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
            ctx := r.Context()
            clnt, err := api.Http().GetClientDevice(r)
            if err != nil {
                api.Logger().Error("Error getting client device: " + err.Error())
                return sdkapi.ViewPage{}
            }

            summary, err := api.SessionsMgr().SessionSummary(ctx, clnt)
            if err != nil {
                api.Logger().Error("Error getting session summary: " + err.Error())
                return sdkapi.ViewPage{}
            }

            _, ok := api.SessionsMgr().RunningSession(clnt)
            navs := api.Http().Navs().GetPortalItems(r)

            page := portal.PortalIndexPage(api, portal.PortalIndexData{
                Navs:             navs,
                SessionSummary:   summary,
                IsSessionRunning: ok,
            })

            return sdkapi.ViewPage{
                Assets: sdkapi.ViewAssets{
                    JsFile: "portal/index.js",
                },
                PageContent: page,
            }
        },
    })
}
```

## Step 8: Create Template Views

Use the templ template engine to create your views. See the [rendering views guide](../guides/rendering-views.md) for detailed information on creating and using templ templates.

Create files like:

- `resources/views/admin/layout.templ`
- `resources/views/admin/index.templ`
- `resources/views/portal/layout.templ`
- `resources/views/portal/index.templ`
- `resources/views/auth/login.templ`

## Basic Template Examples

### resources/views/admin/layout.templ

```templ
package admin

import (
	"encoding/json"
	"fmt"
	sdkapi "sdk/api"
)

type AdminLayoutData struct {
	Components    sdkapi.IThemeComponents
	Navs          []sdkapi.AdminNavList
	NavItems      []sdkapi.AdminNavItem
	Notifications []sdkapi.Notification
}

templ AdminLayout(api sdkapi.IPluginApi, data AdminLayoutData) {
	<!DOCTYPE html>
	<html lang="en" data-bs-theme="auto" { data.Components.HtmlAttrs()... }>
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			@data.Components.Head()
		</head>
		<body { data.Components.BodyAttrs()... }>
			<header>
				<h1>Admin Header</h1>
			</header>
			<main>
				@data.Components.PageContent()
			</main>
			@data.Components.Scripts()
		</body>
	</html>
}
```

### resources/views/admin/index.templ

```templ
package admin

import (
	sdkapi "sdk/api"
)

templ AdminIndexPage(api sdkapi.IPluginApi, data interface{}) {
	<h1>Admin Dashboard</h1>
	<p>Welcome to the admin panel.</p>
}
```

### resources/views/portal/layout.templ

```templ
package portal

import (
	sdkapi "sdk/api"
)

type PortalLayoutData struct {
	Components sdkapi.IThemeComponents
}

templ PortalLayout(data PortalLayoutData) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			@data.Components.Head()
		</head>
		<body>
			<header>
				<h1>Portal Header</h1>
			</header>
			<main>
				@data.Components.PageContent()
			</main>
			@data.Components.Scripts()
		</body>
	</html>
}
```

### resources/views/portal/index.templ

```templ
package portal

import (
	sdkapi "sdk/api"
)

type PortalIndexData struct {
	Navs             []sdkapi.PortalNav
	SessionSummary   interface{}
	IsSessionRunning bool
}

templ PortalIndexPage(api sdkapi.IPluginApi, data PortalIndexData) {
	<h1>Hotspot Portal</h1>
	<p>Please log in to access the internet.</p>
}
```

### resources/views/auth/login.templ

```templ
package auth

import (
	sdkapi "sdk/api"
)

templ LoginPage(api sdkapi.IPluginApi, csrfHtml string, data sdkapi.LoginPageData) {
	<h1>Login</h1>
	<form method="post">
		@templ.Raw(csrfHtml)
		<div>
			<label for="username">Username:</label>
			<input type="text" id="username" name="username"/>
		</div>
		<div>
			<label for="password">Password:</label>
			<input type="password" id="password" name="password"/>
		</div>
		<button type="submit">Login</button>
	</form>
}
```

Example `resources/views/admin/layout.templ` (see the [rendering views guide](../guides/rendering-views.md) for templ syntax details):

```templ
package admin

import (
    "net/http"
    sdkapi "sdk/api"
)

type AdminLayoutData struct {
    Components    sdkapi.IThemeComponents
    Navs          []sdkapi.AdminNavList
    NavItems      []sdkapi.AdminNavItem
    Notifications []sdkapi.Notification
}

templ AdminLayout(api sdkapi.IPluginApi, data AdminLayoutData) {
    <!DOCTYPE html>
    <html lang="en" data-bs-theme="auto" { data.Components.HtmlAttrs()... }>
        <head>
            <meta charset="utf-8"/>
            <meta name="viewport" content="width=device-width, initial-scale=1"/>
            @data.Components.Head()
        </head>
        <body { data.Components.BodyAttrs()... }>
            <nav>
                // Your navigation implementation
            </nav>
            <main class="col-md-9 ms-sm-auto col-lg-10 px-md-4">
                <div class="container pt-3">
                    @data.Components.PageContent()
                </div>
            </main>
            @data.Components.Scripts()
        </body>
    </html>
}
```

## Step 9: Add Assets

Create your CSS and JavaScript files in `resources/assets/`. Create manifest files to define which assets to include. See the [assets manifest documentation](../api/assets-manifest.md) for detailed information on manifest file structure.

Based on the code examples above, you'll need these asset files:

- `resources/assets/admin/theme.js`
- `resources/assets/admin/theme.css`
- `resources/assets/portal/theme.js`
- `resources/assets/portal/theme.css`
- `resources/assets/auth/login.js`
- `resources/assets/auth/login.css`

`resources/assets/manifest.admin.json`:
```json
{
  "index.js": ["./admin/theme.js"],
  "index.css": ["./admin/theme.css"]
}
```

`resources/assets/manifest.portal.json`:
```json
{
  "index.js": ["./portal/theme.js"],
  "index.css": ["./portal/theme.css"],
  "auth/login.js": ["./auth/login.js"],
  "auth/login.css": ["./auth/login.css"]
}
```

## Step 10: Add Translations (Optional)

Create translation files in `resources/translations/` for different languages. See the [translations guide](../guides/translations.md) for detailed information on how to implement translations in your plugin.

```
resources/translations/
├── en/
│   ├── info/
│   │   └── Logged in successfully.txt
│   └── error/
│       └── Invalid credentials.txt
└── es/
    ├── info/
    │   └── Logged in successfully.txt
    └── error/
        └── Invalid credentials.txt
```

Example translation file `resources/translations/en/label/dashboard.txt`:
```
Dashboard
```

Example translation file `resources/translations/es/label/dashboard.txt`:
```
Panel de Control
```

Use translations in your templates:
```templ
<h1>{ api.Translate("label", "dashboard") }</h1>
```

## Step 11: Build and Test

The development environment automatically watches for file changes and rebuilds your plugin. Simply:

1. Start the development server (if not already running)
2. Make changes to your theme files
3. The system will automatically rebuild and reload your plugin
4. Test both admin and portal interfaces in your browser
5. Verify that all assets load correctly and the theme displays properly

## Tips

- Start by copying the default theme and modifying it
- Use the SDK API documentation for available methods
- Test your theme on different screen sizes
- Ensure your CSS is compatible with the chosen CSS library (Bootstrap 3 or 5)
- Keep your JavaScript ES5 compatible for maximum browser support
- Changes are automatically rebuilt - just refresh your browser to see updates

## Next Steps

- [Customizing Admin Interface](page2.md)
- [Advanced Portal Features](page3.md)