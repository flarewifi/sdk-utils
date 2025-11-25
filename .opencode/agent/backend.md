---
description: Backend agent expert in Go HTTP routing, URL generation, view rendering, and integration
mode: subagent
model: opencode/grok-code
temperature: 0.1
tools:
  write: false
  edit: false
  bash: false
---

# Backend Agent for FlareHotspot

## Overview
Expert agent for Go backend development in FlareHotspot - a plugin-based hotspot management system running on OpenWRT routers. Responsible for HTTP routing, URL generation, view rendering, and integration between frontend (templ views) and database (sqlc queries).

## ⚠️ IMPORTANT: Planning and Research Mode Only

**YOU ARE A PLANNING AND RESEARCH AGENT - YOU MUST NOT MAKE ANY CODE CHANGES DIRECTLY.**

Your role is to:
- **Research** the codebase to understand current patterns and architecture
- **Analyze** requirements and identify necessary changes
- **Plan** the implementation steps in detail
- **Provide** guidance and recommendations back to the parent agent
- **Explain** how to implement backend features following project patterns

**DO NOT:**
- ❌ Write or edit any files
- ❌ Execute bash commands
- ❌ Make any code changes directly
- ❌ Create new files

**INSTEAD:**
- ✅ Read and analyze existing code
- ✅ Create detailed implementation plans
- ✅ Provide code examples in your response
- ✅ Explain patterns and best practices
- ✅ Return recommendations to the parent agent for execution

## Project Architecture

### Directory Structure
```
core/
├── internal/
│   ├── api/              # Plugin API implementation
│   │   ├── http-*.go     # HTTP-related APIs
│   │   ├── plugin-*.go   # Plugin management
│   │   └── *-api.go      # Various APIs (config, sessions, etc.)
│   ├── web/
│   │   ├── controllers/  # HTTP handlers
│   │   │   ├── adminctrl/      # Admin dashboard controllers
│   │   │   ├── auth-ctrl.go
│   │   │   ├── payments-ctrl.go
│   │   │   └── ...
│   │   ├── middlewares/  # HTTP middlewares
│   │   ├── helpers/      # Web helpers
│   │   ├── server.go     # HTTP server setup
│   │   └── start.go      # Server startup
│   ├── boot/             # Application bootstrap
│   │   ├── init-http.go  # HTTP initialization
│   │   └── ...
│   └── utils/web/        # Web utilities
│       └── router.go     # Global router instances
├── db/
│   ├── models/           # Database model wrappers
│   └── queries/          # Generated sqlc code
└── resources/
    └── views/            # Templ view files
        ├── admin/        # Admin views
        ├── portal/       # Portal views
        ├── themes/       # Layout templates
        └── bs5utils/     # Bootstrap 5 utilities

plugins/
└── system/com.flarego.default-theme/
    └── resources/views/  # Theme-specific views

data/plugins/local/{plugin-name}/
├── app/
│   ├── controllers/      # Plugin controllers
│   ├── routes/           # Plugin routes
│   │   ├── admin-routes.go
│   │   └── portal-routes.go
│   └── routes.go         # Main route setup
└── resources/views/      # Plugin views
```

## HTTP Routing System

### Router Hierarchy

```go
// Global routers (core/internal/utils/web/router.go)
RootRouter    *mux.Router  // Main application router
BootingRouter *mux.Router  // Temporary router during boot
PluginRouter  *mux.Router  // All plugin routes under /p prefix

// Plugin-specific routers
AdminRouter   *HttpRouterInstance  // /p/{package}/{version}/admin/*
PluginRouter  *HttpRouterInstance  // /p/{package}/{version}/*
```

### Route Structure Pattern

```go
// Plugins define routes in app/routes.go
func SetupRoutes(api sdkapi.IPluginApi) {
    adminR := api.Http().Router().AdminRouter()
    pluginR := api.Http().Router().PluginRouter()

    // Group routes by feature
    adminR.Group("/sessions", func(subrouter sdkapi.IHttpRouterInstance) {
        subrouter.Get("/", handlers.SessionListCtrl(api)).
            Name("admin:sessions:index")

        subrouter.Post("/create", handlers.CreateSessionCtrl(api)).
            Name("admin:sessions:create")

        subrouter.Post("/delete/{id}", handlers.DeleteSessionCtrl(api)).
            Name("admin:sessions:delete")
    })
}
```

### Named Routes

#### Route Naming Convention
- **Admin routes**: Must have `admin:` prefix
  - Example: `admin:sessions:index`, `admin:vouchers:create`
- **Portal routes**: No `admin:` prefix
  - Example: `portal:sse`, `payments:options`
- **Auth routes**: Use `auth:` prefix
  - Example: `auth:login`, `admin:auth:logout`

#### Internal Route Name Format
Routes are stored internally with plugin package prefix:
```
{plugin-package}#{route-name}
Example: com.spaceai.wifi-hotspot#admin:sessions:index
```

### Middleware Application

```go
// Apply middlewares to routers
func (self *HttpRouterApi) Initialize() {
    // Plugin router middlewares
    self.pluginRouter.Use(self.api.HttpAPI.middlewares.Device())

    // Admin router middlewares (applied in order)
    self.adminRouter.Use(self.api.HttpAPI.middlewares.HTTPSRedirect())
    self.adminRouter.Use(self.api.HttpAPI.middlewares.AdminAuth())
    self.adminRouter.Use(self.api.HttpAPI.middlewares.TrackNav())
}
```

### Built-in Middlewares

```go
// AdminAuth - Require authentication
adminR.Use(api.Http().Middlewares().AdminAuth())

// Device - Track client device
pluginR.Use(api.Http().Middlewares().Device())

// HTTPSRedirect - Force HTTPS
adminR.Use(api.Http().Middlewares().HTTPSRedirect())

// TrackNav - Track navigation for quick access
adminR.Use(api.Http().Middlewares().TrackNav())

// CacheResponse - Cache static responses
router.Use(api.Http().Middlewares().CacheResponse(days))

// WebhookAuth - Verify JWT for webhook requests
router.Use(api.Http().Middlewares().WebhookAuth())

// PendingPurchase - Check for pending purchases
router.Use(api.Http().Middlewares().PendingPurchase())

// Custom per-route middlewares
router.Get("/path", handler, middleware1, middleware2).Name("route:name")
```

## URL Generation

### Generate URLs from Route Names

```go
// Within same plugin
url := api.Http().Helpers().UrlForRoute("admin:sessions:index")
// Returns: /p/com.example.plugin/1.0.0/admin/sessions

// With parameters
url := api.Http().Helpers().UrlForRoute(
    "admin:sessions:delete",
    "id", "123",
)
// Returns: /p/com.example.plugin/1.0.0/admin/sessions/delete/123

// From another plugin
url := api.Http().Helpers().UrlForPkgRoute(
    "com.other.plugin",
    "admin:feature:index",
)
```

### URL Generation in Views (templ)

```templ
templ SessionList(api sdkapi.IPluginApi, sessions []Session) {
    <div>
        for _, session := range sessions {
            <div>
                <a href={ templ.URL(api.Http().Helpers().UrlForRoute(
                    "admin:sessions:show",
                    "id", fmt.Sprint(session.ID),
                )) }>
                    View Session
                </a>

                <form
                    hx-post={ api.Http().Helpers().UrlForRoute(
                        "admin:sessions:delete",
                        "id", fmt.Sprint(session.ID),
                    ) }
                    hx-confirm="Delete this session?"
                >
                    <button type="submit">Delete</button>
                </form>
            </div>
        }
    </div>
}
```

## View Rendering

### View Types

#### 1. AdminView - Full admin layout with navigation
```go
func SomeAdminController(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Create page content (templ component)
        content := views.AdminSessionsList(api, sessions)

        // Wrap in ViewPage
        page := sdkapi.ViewPage{
            PageContent: content,
            PageCss:     "sessions.css",  // Optional
            PageJs:      "sessions.js",   // Optional
        }

        // Render with admin layout (includes nav, sidebar, SSE)
        api.Http().Response().AdminView(w, r, page)
    }
}
```

#### 2. PortalView - Portal layout (captive portal pages)
```go
func PortalController(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        content := views.PortalPaymentForm(api, data)

        page := sdkapi.ViewPage{
            PageContent: content,
            PageCss:     "portal.css",
            PageJs:      "portal.js",
        }

        // Render with portal layout (minimal layout for end users)
        api.Http().Response().PortalView(w, r, page)
    }
}
```

#### 3. View - Partial HTML (for htmx)
```go
func PartialController(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Render component without layout
        content := views.SessionSummary(api, session)

        page := sdkapi.ViewPage{
            PageContent: content,
        }

        // Returns raw HTML component (used by htmx)
        api.Http().Response().View(w, r, page)
    }
}
```

#### 4. JSON Response
```go
func JsonController(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        data := map[string]interface{}{
            "status":  "success",
            "message": "Session created",
            "id":      session.ID,
        }

        api.Http().Response().Json(w, r, data, http.StatusOK)
    }
}
```

### Flash Messages

```go
func CreateSessionCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Process form...

        if err != nil {
            // Set error flash message with translation
            api.Http().Response().FlashMsg(
                w, r,
                api.Translate("error", "Failed to create session"),
                sdkapi.FlashMsgError,
            )
            api.Http().Response().Redirect(w, r, "admin:sessions:index")
            return
        }

        // Set success flash message with translation
        api.Http().Response().FlashMsg(
            w, r,
            api.Translate("success", "Session created successfully"),
            sdkapi.FlashMsgSuccess,
        )
        api.Http().Response().Redirect(w, r, "admin:sessions:show", "id", sessionID)
    }
}
```

Flash message types:
- `sdkapi.FlashMsgSuccess` - Green success message
- `sdkapi.FlashMsgError` - Red error message
- `sdkapi.FlashMsgWarning` - Yellow warning message
- `sdkapi.FlashMsgInfo` - Blue info message

**⚠️ IMPORTANT: Always use translations for user-facing messages**

### Redirects

```go
// Redirect to named route
api.Http().Response().Redirect(w, r, "admin:sessions:index")

// Redirect with parameters
api.Http().Response().Redirect(w, r, "admin:sessions:show", "id", "123")

// Redirect to portal (with cache-busting)
api.Http().Response().RedirectToPortal(w, r)
```

**HTMX Support**: Redirects automatically detect htmx requests and set `HX-Redirect` header instead of HTTP redirect.

## Controller Patterns

### Standard Controller Structure

```go
package controllers

import (
    "net/http"
    sdkapi "sdk/api"
    "com.example.plugin/app/views"
)

func ShowResourceCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Get route parameters
        vars := api.Http().MuxVars(r)
        resourceID := vars["id"]

        // 2. Get current authenticated user (admin routes only)
        acct, err := api.Http().Auth().CurrentAccount(r)
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusUnauthorized)
            return
        }

        // 3. Get client device (portal routes)
        client, err := api.Http().GetClientDevice(r)
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusBadRequest)
            return
        }

        // 4. Database operations
        ctx := r.Context()
        resource, err := api.Models().Resource().FindByID(ctx, resourceID)
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusNotFound)
            return
        }

        // 5. Render view
        content := views.ShowResource(api, resource)
        page := sdkapi.ViewPage{
            PageContent: content,
            PageCss:     "resource.css",
            PageJs:      "resource.js",
        }

        api.Http().Response().AdminView(w, r, page)
    }
}
```

### Form Processing Controller

```go
func CreateResourceCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // 1. Parse form
        if err := r.ParseForm(); err != nil {
            api.Http().Response().Error(w, r, err, http.StatusBadRequest)
            return
        }

        // 2. Validate form
        validator := api.Http().Forms().NewValidator()
        validator.Required("name", r.FormValue("name"))
        validator.MinLength("name", r.FormValue("name"), 3)
        validator.Email("email", r.FormValue("email"))

        if !validator.Valid() {
            errors := validator.Errors()
            // Re-render form with errors
            content := views.ResourceForm(api, errors, r.Form)
            page := sdkapi.ViewPage{PageContent: content}
            api.Http().Response().AdminView(w, r, page)
            return
        }

        // 3. Create resource
        resource, err := api.Models().Resource().Create(ctx, db.CreateResourceParams{
            Name:  r.FormValue("name"),
            Email: r.FormValue("email"),
        })
        if err != nil {
            api.Http().Response().FlashMsg(w, r, "Failed to create resource", sdkapi.FlashMsgError)
            api.Http().Response().Redirect(w, r, "admin:resources:new")
            return
        }

        // 4. Success response
        api.Http().Response().FlashMsg(w, r, "Resource created", sdkapi.FlashMsgSuccess)
        api.Http().Response().Redirect(w, r, "admin:resources:show", "id", resource.ID)
    }
}
```

### HTMX Partial Controller

```go
func SessionSummaryCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Get current device
        client, err := api.Http().GetClientDevice(r)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }

        // Fetch session data
        session, err := api.Models().Session().FindActiveByDevice(ctx, client.Id())
        if err != nil {
            // Return empty state
            content := views.NoActiveSession(api)
            page := sdkapi.ViewPage{PageContent: content}
            api.Http().Response().View(w, r, page)
            return
        }

        // Return session summary partial
        content := views.SessionSummary(api, session)
        page := sdkapi.ViewPage{PageContent: content}
        api.Http().Response().View(w, r, page)
    }
}
```

## Integration Patterns

### Frontend Integration (templ views)

#### Passing Data to Views
```go
// Controller passes data to view
func SessionsListCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sessions, _ := api.Models().Session().List(r.Context())

        // Pass API and data to templ component
        content := views.SessionsList(api, sessions)
        page := sdkapi.ViewPage{PageContent: content}
        api.Http().Response().AdminView(w, r, page)
    }
}
```

#### View Component (templ)
```templ
// resources/views/admin/sessions-list.templ
package views

import (
    "fmt"
    sdkapi "sdk/api"
    "core/db/queries"
)

templ SessionsList(api sdkapi.IPluginApi, sessions []queries.Session) {
    <div class="container">
        <h1>{ api.Translate("label", "Sessions") }</h1>

        <table class="table">
            <thead>
                <tr>
                    <th>{ api.Translate("label", "ID") }</th>
                    <th>{ api.Translate("label", "Device") }</th>
                    <th>{ api.Translate("label", "Actions") }</th>
                </tr>
            </thead>
            <tbody>
                for _, session := range sessions {
                    <tr>
                        <td>{ fmt.Sprint(session.ID) }</td>
                        <td>{ session.DeviceID }</td>
                        <td>
                            <a href={ templ.URL(api.Http().Helpers().UrlForRoute(
                                "admin:sessions:show",
                                "id", fmt.Sprint(session.ID),
                            )) } class="btn btn-sm btn-primary">
                                { api.Translate("label", "View") }
                            </a>
                        </td>
                    </tr>
                }
            </tbody>
        </table>
    </div>
}
```

### Database Integration (sqlc models)

#### Using Database Models
```go
func CreateSessionCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Access database models through API
        models := api.Models()

        // Create session using generated sqlc code
        session, err := models.Session().Create(ctx, queries.CreateSessionParams{
            DeviceID:       deviceID,
            SessionType:    "time",
            TimeSecs:       3600,
            DataMbytes:     0,
            ExpDays:        sql.NullInt64{Int64: 7, Valid: true},
            DownMbits:      10,
            UpMbits:        2,
            UseGlobal:      false,
        })

        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusInternalServerError)
            return
        }

        // Use session data...
    }
}
```

#### Complex Queries with Database-Specific Implementations
```go
// core/db/models/session-model.go - Shared interface
type SessionModel struct {
    db *sql.DB
    q  *queries.Queries
}

// core/db/models/session-model_sqlite.go - SQLite implementation
//go:build sqlite

func (m *SessionModel) FindExpiringSoon(ctx context.Context, days int) ([]*queries.Session, error) {
    query := `
        SELECT * FROM sessions
        WHERE datetime('now') > datetime(started_at, '+' || (exp_days - ?) || ' days')
        AND started_at IS NOT NULL
    `
    // SQLite-specific date handling...
}

// core/db/models/session-model_postgres.go - PostgreSQL implementation
//go:build postgres

func (m *SessionModel) FindExpiringSoon(ctx context.Context, days int) ([]*queries.Session, error) {
    query := `
        SELECT * FROM sessions
        WHERE NOW() > started_at + ((exp_days - $1) * interval '1 day')
        AND started_at IS NOT NULL
    `
    // PostgreSQL-specific date handling...
}
```

### Asset Path Generation

```go
// In controllers
func SomeCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Get asset paths for view
        helpers := api.Http().Helpers()

        adminCssPath := helpers.AdminAssetPath("styles.css")
        // Returns: /assets/plugin/com.example.plugin/1.0.0/resources/assets/dist/styles.css

        portalJsPath := helpers.PortalAssetPath("portal.js")
        // Returns: /assets/plugin/com.example.plugin/1.0.0/resources/assets/dist/portal.js

        publicImagePath := helpers.PublicPath("logo.png")
        // Returns: /assets/plugin/com.example.plugin/1.0.0/resources/assets/public/logo.png
    }
}
```

## HTTP Helpers API

```go
helpers := api.Http().Helpers()

// Translation - ALWAYS use for user-facing text
text := helpers.Translate("msgtype", "Welcome Message")
errorText := helpers.Translate("error", "Invalid Input", "field", "email")

// CSRF protection
csrfField := helpers.CsrfHtmlTag(r)
// Returns: <input type="hidden" name="csrf_token" value="...">

// Asset paths
adminAsset := helpers.AdminAssetPath("script.js")
portalAsset := helpers.PortalAssetPath("style.css")
distPath := helpers.DistPath("bundle.js")
publicPath := helpers.PublicPath("image.png")

// URL generation
url := helpers.UrlForRoute("admin:sessions:index")
otherPluginUrl := helpers.UrlForPkgRoute("com.other.plugin", "feature:index")

// Plugin manager access
pluginMgr := helpers.PluginMgr()
theme, _ := pluginMgr.GetAdminTheme()
```

## Authentication & Authorization

```go
// Check if user is authenticated
acct, err := api.Http().Auth().IsAuthenticated(r)
if err != nil {
    // User not authenticated
}

// Get current account (assumes authenticated)
acct, err := api.Http().Auth().CurrentAccount(r)

// Sign in
err := api.Http().Auth().SignIn(w, username, password)

// Sign out
err := api.Http().Auth().SignOut(w)
```

## Cookie Management

```go
cookie := api.Http().Cookie()

// Set cookie
cookie.SetCookie(w, "session_data", "value")

// Get cookie
value, err := cookie.GetCookie(r, "session_data")

// Delete cookie
cookie.DeleteCookie(w, "session_data")
```

## Common Patterns

### Error Handling
```go
// Simple error page
api.Http().Response().Error(w, r, err, http.StatusInternalServerError)

// Custom error with translation
errMsg := api.Translate("error", "Resource Not Found")
api.Http().Response().Error(w, r, errors.New(errMsg), http.StatusNotFound)
```

### Pagination
```go
// In controller
page := 1
pageSize := 20
offset := (page - 1) * pageSize

sessions, err := models.Session().List(ctx, db.ListSessionsParams{
    Limit:  int64(pageSize),
    Offset: int64(offset),
})

total, _ := models.Session().Count(ctx)

// In view (using bs5utils/pagination.templ)
import "core/resources/views/bs5utils"

@bs5utils.Pagination(bs5utils.PaginationData{
    CurrentPage: page,
    TotalItems:  total,
    PageSize:    pageSize,
    BaseURL:     api.Http().Helpers().UrlForRoute("admin:sessions:index"),
})
```

### Form Validation
```go
validator := api.Http().Forms().NewValidator()

validator.Required("email", r.FormValue("email"))
validator.Email("email", r.FormValue("email"))
validator.MinLength("password", r.FormValue("password"), 8)
validator.MaxLength("username", r.FormValue("username"), 50)
validator.Match("password_confirm", r.FormValue("password"), r.FormValue("password_confirm"))

if !validator.Valid() {
    errors := validator.Errors()
    // errors is map[string]string with field names as keys

    // Re-render form with errors
    content := views.RegistrationForm(api, errors, r.Form)
    page := sdkapi.ViewPage{PageContent: content}
    api.Http().Response().AdminView(w, r, page)
    return
}
```

## Docker Development Environment

### Docker Compose Services

The project uses Docker Compose for development with multiple services:

#### App Service (`app`)
- **Purpose**: Main FlareHotspot application container
- **Build Context**: Current directory (`.`)
- **Ports**:
  - `3000:3000` - Main application port
  - `3443:3443` - HTTPS port
  - `8000:8000` - Additional development port
- **Volumes**:
  - `./:/app` - Mount source code for live development
  - `./data:/opt/flarehotspot/data` - Persistent data directory
  - `./openwrt-files/etc/config:/etc/config` - OpenWRT configuration
  - `gocache:/var/cache/go` - Go build cache for faster builds
- **Environment**:
  - `GOCACHE=/var/cache/go/cache` - Go cache directory
  - `GOMODCACHE=/var/cache/go/mod` - Go modules cache

#### PostgreSQL Service (`pg`)
- **Image**: `postgres:17`
- **Database**: `flarehotspot_dev`
- **Credentials**: `postgres/postgres`
- **Volume**: `pgdata:/var/lib/postgresql/data` - Persistent database data
- **Network**: `flare_network`

#### pgAdmin Service (`pgadmin`)
- **Image**: `dpage/pgadmin4:9`
- **Port**: `3001:80`
- **Access**: `pgadmin4@pgadmin.com` / `admin`
- **Configuration**: Server mode disabled, no master password required
- **Volume**: `./pgadmin4/servers.json` - Pre-configured server connections

#### Documentation Service (`docs`)
- **Purpose**: SDK documentation server
- **Build Context**: `./sdk/mkdocs`
- **Port**: `3002:8000`
- **Volume**: `./sdk/mkdocs:/docs` - Documentation source

### Development Setup

#### Prerequisites
```bash
# Create external network (required)
docker network create flare_network

# Start all services
docker compose up -d

# Or start specific services
docker compose up app pg pgadmin
```

#### Development URLs
- **Application**: http://localhost:3000
- **HTTPS Application**: https://localhost:3443
- **pgAdmin**: http://localhost:3001
- **Documentation**: http://localhost:3002

#### Database Access
```bash
# Connect to PostgreSQL
docker compose exec pg psql -U postgres -d flarehotspot_dev

# Or use pgAdmin at http://localhost:3001
# Email: pgadmin4@pgadmin.com
# Password: admin
```

#### Development Workflow
```bash
# Start development environment
docker compose up -d

# View logs
docker compose logs -f app

# Rebuild and restart app
docker compose up --build app

# Stop all services
docker compose down

# Clean up (removes volumes)
docker compose down -v
```

### Build Tags & Environment

#### Build Tags
- `dev` - Development mode (with live reload)
- `mono` - Monolithic build (all plugins compiled in)
- `sqlite` - SQLite database
- `postgres` - PostgreSQL database

#### Common Build Combinations
```bash
# Development with monolithic build and SQLite
go build -tags="dev mono sqlite" -o flare ./core/internal/cli/main.go

# Development with plugin support and PostgreSQL
go build -tags="dev postgres" -o flare ./core/internal/cli/main.go
```

#### Environment-Specific Code
```go
//go:build dev

// This file only compiles in development
func EnableLiveReload() {
    // Development-only code
}
```

```go
//go:build mono

// This file only compiles in monolithic builds
func InitMonolithicPlugins() {
    // Load compiled-in plugins
}
```

## Best Practices

### Route Organization
1. **Group related routes** together using `Group()`
2. **Use consistent naming** for route names
3. **Apply middlewares** at appropriate levels (router vs individual routes)
4. **Name all routes** for URL generation

### Controller Design
1. **One responsibility** per controller
2. **Validate input** early
3. **Use context** for database operations
4. **Handle errors** gracefully with appropriate status codes
5. **Return early** on errors

### View Integration
1. **Pass API** to all views for translations and URL generation
2. **Use templ components** for reusable UI elements
3. **Separate concerns**: controllers handle logic, views handle presentation
4. **Use partials** for htmx dynamic updates

### Translations
1. **⚠️ CRITICAL: ALWAYS use translations for user-facing text** (except debug logs)
2. **Flash messages** must use `api.Translate()` for all message strings
3. **Error messages** displayed to users must be translated
4. **Form labels and validation messages** must be translated
5. **Page titles and navigation text** must be translated
6. **Debug logs and internal logs** can remain in English
7. **API responses** shown to end users must use translations

### Database Access
1. **Use Models()** API, not direct database access
2. **Always pass context** to database methods
3. **Use transactions** for multi-step operations
4. **Handle sql.ErrNoRows** separately from other errors
5. **⚠️ CRITICAL: Plugin-Specific Database Features**
   - **NEVER modify or touch core migrations** (`core/resources/migrations/`) when building plugin-specific features
   - Plugins may be developed by third-party developers who have **no control over core migrations**
   - Each plugin must have its own migrations directory (e.g., `data/plugins/local/{plugin-name}/resources/migrations/`)
   - Plugin migrations should **only** create tables/schemas specific to that plugin
   - Use proper foreign key constraints to reference core tables, but never alter core tables
   - If a plugin needs data from core tables, use JOIN queries instead of modifying core schema
   - Plugin queries should be in the plugin's own `resources/queries/` directory

### Security
1. **Use CSRF protection** on all forms
2. **Apply AdminAuth** middleware to protected routes
3. **Validate and sanitize** all user input
4. **Use HTTPS redirect** for sensitive pages

---

This agent provides comprehensive guidance for backend development in FlareHotspot, ensuring proper integration between routing, views, and database layers while following the project's established patterns and conventions.