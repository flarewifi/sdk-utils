# IThemesApi

The `IThemesApi` is used to register custom themes for the Flare Hotspot admin interface and user portal. Themes control the layout, styling, and appearance of the application.

## Accessing IThemesApi

```go
themesApi := api.Themes()
```

---

## IThemesApi Methods

The following methods are available in `IThemesApi`:

### NewAdminTheme

Registers a new admin theme. The admin theme controls the layout and appearance of the admin dashboard.

```go
api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
    JsFile:           "theme.js",
    CssFile:          "theme.css",
    CssLib:           sdkapi.CssLibBootstrap5,
    LayoutBuilder:    layoutBuilderFunc,
    IndexPageFactory: indexPageFactoryFunc,
})
```

### NewPortalTheme

Registers a new portal theme. The portal theme controls the layout and appearance of the captive portal (user-facing pages).

```go
api.Themes().NewPortalTheme(sdkapi.PortalThemeOpts{
    JsFile:           "theme.js",
    CssFile:          "theme.css",
    CssLib:           sdkapi.CssLibBootstrap3,
    LayoutBuilder:    layoutBuilderFunc,
    LoginPageFactory: loginPageFactoryFunc,
    IndexPageFactory: indexPageFactoryFunc,
})
```

### GetAdminTheme

Returns the plugin API for the currently configured admin theme. This allows you to access the theme plugin's APIs and resources.

```go
themePlugin := api.Themes().GetAdminTheme()
if themePlugin != nil {
    // Access theme plugin information
    info := themePlugin.Info()
    fmt.Printf("Admin theme: %s v%s\n", info.Name, info.Version)
    
    // Access theme plugin's storage
    logoData, err := themePlugin.Storage().Read("branding/logo.png")
    
    // Use theme plugin's translation
    message := themePlugin.Translate("label", "Welcome")
}
```

**Returns:**
- The `IPluginApi` for the admin theme plugin if configured and found
- `nil` if no admin theme is configured or the plugin is not installed

### GetPortalTheme

Returns the plugin API for the currently configured portal theme. This allows you to access the theme plugin's APIs and resources.

```go
themePlugin := api.Themes().GetPortalTheme()
if themePlugin != nil {
    // Access theme plugin information
    info := themePlugin.Info()
    fmt.Printf("Portal theme: %s v%s\n", info.Name, info.Version)
    
    // Get theme-specific configuration
    themeConfig, err := themePlugin.Config().Plugin().Read("theme_settings")
    
    // Access theme assets
    assetPath := themePlugin.Http().Helpers().PublicPath("logo.png")
}
```

**Returns:**
- The `IPluginApi` for the portal theme plugin if configured and found
- `nil` if no portal theme is configured or the plugin is not installed

---

## Types

### CSSLib

The `CSSLib` type specifies which CSS framework the theme uses:

```go
type CSSLib string

const (
    CssLibBootstrap5 CSSLib = "bootstrap5"  // Bootstrap 5.3.3 - Admin only
    CssLibBootstrap3 CSSLib = "bootstrap3"  // Bootstrap 3.4.1 - Portal only
)
```

!!! warning "CSS Library Restrictions"
    - **Admin themes** must use `CssLibBootstrap5`
    - **Portal themes** typically use `CssLibBootstrap3` for maximum device compatibility

### AdminThemeOpts

Options for registering an admin theme:

```go
type AdminThemeOpts struct {
    CssLib           CSSLib
    JsFile           string
    CssFile          string
    LayoutBuilder    func(w http.ResponseWriter, r *http.Request, builder IThemeComponents)
    IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}
```

| Field | Description |
|-------|-------------|
| `CssLib` | CSS framework to use (must be `CssLibBootstrap5` for admin) |
| `JsFile` | Main JavaScript file from assets manifest |
| `CssFile` | Main CSS file from assets manifest |
| `LayoutBuilder` | Function that renders the page layout |
| `IndexPageFactory` | Function that returns the admin dashboard page |

### PortalThemeOpts

Options for registering a portal theme:

```go
type PortalThemeOpts struct {
    JsFile           string
    CssFile          string
    CssLib           CSSLib
    LayoutBuilder    func(w http.ResponseWriter, r *http.Request, builder IThemeComponents)
    LoginPageFactory func(w http.ResponseWriter, r *http.Request, data LoginPageData) ViewPage
    IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}
```

| Field | Description |
|-------|-------------|
| `JsFile` | Main JavaScript file from assets manifest |
| `CssFile` | Main CSS file from assets manifest |
| `CssLib` | CSS framework to use |
| `LayoutBuilder` | Function that renders the page layout |
| `LoginPageFactory` | Function that returns the admin login page |
| `IndexPageFactory` | Function that returns the portal home page |

### LoginPageData

Data passed to the login page factory:

```go
type LoginPageData struct {
    LoginError error  // Error from failed login attempt, nil if no error
}
```

### FlashMsg

Represents a flash message to display to the user:

```go
type FlashMsg struct {
    Type    string  // "success", "info", "warning", "error"
    Message string  // The message text
}
```

### ViewPage

The return type for page factory functions:

```go
type ViewPage struct {
    Assets      ViewAssets       // Page-specific assets
    PageContent templ.Component  // The page content component
}

type ViewAssets struct {
    JsFile  string  // JavaScript file from manifest
    CssFile string  // CSS file from manifest
}
```

---

## IThemeComponents

The `IThemeComponents` interface provides access to page components that must be included in your layout template. It is passed to the `LayoutBuilder` function.

```go
type IThemeComponents interface {
    HtmlAttrs() templ.Attributes    // Attributes for <html> tag
    Head() templ.Component          // Content for <head> section
    BodyAttrs() templ.Attributes    // Attributes for <body> tag
    PageContent() templ.Component   // The main page content
    Scripts() templ.Component       // JavaScript includes
}
```

### Using IThemeComponents in Templates

```templ
templ AdminLayout(data AdminLayoutData) {
    <!DOCTYPE html>
    <html lang="en" { data.Components.HtmlAttrs()... }>
        <head>
            <meta charset="utf-8"/>
            <meta name="viewport" content="width=device-width, initial-scale=1"/>
            @data.Components.Head()
        </head>
        <body { data.Components.BodyAttrs()... }>
            <nav>
                <!-- Your navigation -->
            </nav>
            <main>
                @data.Components.PageContent()
            </main>
            @data.Components.Scripts()
        </body>
    </html>
}
```

!!! important "Required Components"
    You **must** include all `IThemeComponents` methods in your layout:
    
    - `HtmlAttrs()` - Required HTML attributes
    - `Head()` - CSS, meta tags, and other head content
    - `BodyAttrs()` - Required body attributes
    - `PageContent()` - The actual page content
    - `Scripts()` - JavaScript files and inline scripts

---

## Usage Examples

### Getting Current Theme Information

```go
func DisplayThemeInfo(api sdkapi.IPluginApi) {
    // Get admin theme
    adminTheme := api.Themes().GetAdminTheme()
    if adminTheme != nil {
        info := adminTheme.Info()
        fmt.Printf("Admin Theme: %s (%s) v%s\n", 
            info.Name, info.Package, info.Version)
        
        // Check if theme has custom branding
        if adminTheme.Storage().Exists("branding/logo.png") {
            logoURL := adminTheme.Storage().UrlFor("branding/logo.png")
            fmt.Printf("Custom logo: %s\n", logoURL)
        }
    }
    
    // Get portal theme
    portalTheme := api.Themes().GetPortalTheme()
    if portalTheme != nil {
        info := portalTheme.Info()
        fmt.Printf("Portal Theme: %s (%s) v%s\n", 
            info.Name, info.Package, info.Version)
    }
}
```

### Accessing Theme Resources from Another Plugin

```go
func GetThemeBranding(api sdkapi.IPluginApi) ([]byte, error) {
    // Get the current admin theme
    theme := api.Themes().GetAdminTheme()
    if theme == nil {
        return nil, fmt.Errorf("no admin theme configured")
    }
    
    // Read branding image from theme's storage
    logoData, err := theme.Storage().Read("branding/company-logo.png")
    if err != nil {
        return nil, fmt.Errorf("failed to read theme logo: %w", err)
    }
    
    return logoData, nil
}
```

### Customizing Theme Behavior Based on Theme Plugin

```go
func CustomizeForTheme(api sdkapi.IPluginApi, w http.ResponseWriter, r *http.Request) {
    adminTheme := api.Themes().GetAdminTheme()
    if adminTheme == nil {
        // Use default behavior
        return
    }
    
    themePackage := adminTheme.Info().Package
    
    // Apply theme-specific customizations
    switch themePackage {
    case "com.example.dark-theme":
        // Enable dark mode features
        w.Header().Set("X-Theme-Mode", "dark")
        
    case "com.example.minimal-theme":
        // Use minimal UI components
        w.Header().Set("X-Theme-Style", "minimal")
        
    default:
        // Default theme behavior
    }
}
```

### Admin Theme Implementation

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
        CssLib:  sdkapi.CssLibBootstrap5,
        LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
            // Get navigation items for sidebar
            navs := api.Http().Navs().GetAdminNavs(r)

            // Flatten nav items for search/quick access
            var navItems []sdkapi.AdminNavItem
            for _, nav := range navs {
                navItems = append(navItems, nav.Items...)
            }

            // Get unread notifications
            notifs, err := api.Notification().GetUnreadNotifications(r.Context())
            if err != nil {
                notifs = []sdkapi.Notification{}
            }

            // Prepare layout data
            data := admin.AdminLayoutData{
                Components:    c,
                Navs:          navs,
                NavItems:      navItems,
                Notifications: notifs,
            }

            // Render the layout
            layout := admin.AdminLayout(api, data)
            if err := layout.Render(r.Context(), w); err != nil {
                fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
            }
        },
        IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
            // Create the admin dashboard page
            page := admin.AdminIndexPage(api)
            return sdkapi.ViewPage{
                PageContent: page,
            }
        },
    })
}
```

### Portal Theme Implementation

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
        CssLib:  sdkapi.CssLibBootstrap3,
        LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
            data := portal.PortalLayoutData{Components: c}
            layout := portal.PortalLayout(data)
            if err := layout.Render(r.Context(), w); err != nil {
                fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
            }
        },
        LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
            // Get CSRF token for the login form
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

            // Get client device information
            clnt, err := api.Http().GetClientDevice(r)
            if err != nil {
                api.Logger().Error("Error getting client device: " + err.Error())
                return sdkapi.ViewPage{}
            }

            // Get session summary for the device
            summary, err := api.SessionsMgr().SessionSummary(ctx, clnt)
            if err != nil {
                api.Logger().Error("Error getting session summary: " + err.Error())
                return sdkapi.ViewPage{}
            }

            // Check if there's a running session
            _, isRunning := api.SessionsMgr().RunningSession(clnt)

            // Get portal navigation items
            navs := api.Http().Navs().GetPortalItems(r)

            page := portal.PortalIndexPage(api, portal.PortalIndexData{
                Navs:             navs,
                SessionSummary:   summary,
                IsSessionRunning: isRunning,
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

### Login Page Template

```templ
package auth

import (
    sdkapi "sdk/api"
)

templ LoginPage(api sdkapi.IPluginApi, csrfHtml string, data sdkapi.LoginPageData) {
    <div class="login-container">
        <h1>{ api.Translate("label", "Admin Login") }</h1>

        if data.LoginError != nil {
            <div class="alert alert-danger">
                { api.Translate("error", "Invalid username or password") }
            </div>
        }

        <form method="post">
            @templ.Raw(csrfHtml)

            <div class="form-group">
                <label for="username">{ api.Translate("label", "Username") }</label>
                <input type="text" id="username" name="username" class="form-control" required/>
            </div>

            <div class="form-group">
                <label for="password">{ api.Translate("label", "Password") }</label>
                <input type="password" id="password" name="password" class="form-control" required/>
            </div>

            <button type="submit" class="btn btn-primary">
                { api.Translate("label", "Login") }
            </button>
        </form>
    </div>
}
```

### Admin Layout Template

```templ
package admin

import (
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
            <title>{ api.Translate("label", "Admin Dashboard") }</title>
            @data.Components.Head()
        </head>
        <body { data.Components.BodyAttrs()... }>
            <div class="container-fluid">
                <div class="row">
                    <!-- Sidebar -->
                    <nav class="col-md-3 col-lg-2 d-md-block bg-light sidebar">
                        <div class="position-sticky pt-3">
                            <ul class="nav flex-column">
                                for _, navList := range data.Navs {
                                    <li class="nav-item">
                                        <span class="nav-header">{ navList.Label }</span>
                                        <ul class="nav flex-column">
                                            for _, item := range navList.Items {
                                                <li class="nav-item">
                                                    <a class="nav-link" href={ templ.SafeURL(item.Url) }>
                                                        { item.Label }
                                                    </a>
                                                </li>
                                            }
                                        </ul>
                                    </li>
                                }
                            </ul>
                        </div>
                    </nav>

                    <!-- Main content -->
                    <main class="col-md-9 ms-sm-auto col-lg-10 px-md-4">
                        <div class="container pt-3">
                            @data.Components.PageContent()
                        </div>
                    </main>
                </div>
            </div>
            @data.Components.Scripts()
        </body>
    </html>
}
```

### Portal Layout Template

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
    <html lang="en" { data.Components.HtmlAttrs()... }>
        <head>
            <meta charset="utf-8"/>
            <meta name="viewport" content="width=device-width, initial-scale=1"/>
            @data.Components.Head()
        </head>
        <body { data.Components.BodyAttrs()... }>
            <div class="container">
                <header class="text-center py-4">
                    <h1>WiFi Hotspot</h1>
                </header>
                <main>
                    @data.Components.PageContent()
                </main>
                <footer class="text-center py-4">
                    <p>Powered by Flare Hotspot</p>
                </footer>
            </div>
            @data.Components.Scripts()
        </body>
    </html>
}
```

---

## Theme Plugin Structure

A complete theme plugin follows this structure:

```
my-theme-plugin/
├── main.go                    # Plugin entry point
├── plugin.json                # Plugin metadata
├── go.mod
├── app/
│   └── themes/
│       ├── admin.go           # Admin theme registration
│       └── portal.go          # Portal theme registration
└── resources/
    ├── assets/
    │   ├── admin/
    │   │   ├── theme.js
    │   │   └── theme.css
    │   ├── portal/
    │   │   ├── theme.js
    │   │   └── theme.css
    │   ├── auth/
    │   │   ├── login.js
    │   │   └── login.css
    │   ├── manifest.admin.json
    │   └── manifest.portal.json
    ├── translations/
    │   └── en/
    │       └── label/
    └── views/
        ├── admin/
        │   ├── layout.templ
        │   └── index.templ
        ├── auth/
        │   └── login.templ
        └── portal/
            ├── layout.templ
            └── index.templ
```

### Plugin Entry Point

```go
//go:build !mono

package main

import (
    sdkapi "sdk/api"

    "com.example.my-theme/app/themes"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    themes.SetPortalTheme(api)
    themes.SetAdminTheme(api)
    return nil
}
```

---

## Best Practices

1. **Always include all IThemeComponents** - Missing components will cause rendering issues

2. **Use translations for all text** - Call `api.Translate()` for user-facing strings

3. **Use ES5 JavaScript** - Portal pages must support older browsers on embedded devices

4. **Test on multiple devices** - Portal themes should work on phones, tablets, and desktops

5. **Keep assets organized** - Use separate directories for admin, portal, and auth assets

6. **Handle errors gracefully** - Check for errors in factory functions and provide fallbacks

7. **Use the correct CSS library** - Bootstrap 5 for admin, Bootstrap 3 for portal

---

## Related

- [Creating a Theme Plugin](../tutorials/creating-a-theme/index.md) - Step-by-step tutorial
- [IHttpResponse](./http-response.md) - Rendering views with themes
- [Assets Manifest](./assets-manifest.md) - Configuring theme assets
- [IHttpNavsApi](./http-navs-api.md) - Navigation items for themes
