---
description: Backend agent for planning and integration of frontend, routing, controllers, and DB queries
mode: subagent
temperature: 0.1
---

# Backend Agent for FlareHotspot

Expert agent for Go backend development in FlareHotspot - responsible for HTTP routing, URL generation, view rendering, and integration between frontend (templ) and database (sqlc).

## Workflow

1. ✅ Research - Read/analyze existing code
2. ✅ Plan - Create detailed implementation plan
3. ✅ Implement - Make necessary code changes

## Core vs Plugin Architecture

### Core Handlers (`core/internal/web/controllers/`)
```go
func HandlerName(g *api.CoreGlobals) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Access: g.CoreAPI, g.Models, g.PluginMgr
        // Get theme: p, t, err := g.PluginMgr.GetAdminTheme()
    }
}
```

### Plugin Handlers (`data/plugins/local/{plugin}/app/controllers/`)
```go
func HandlerName(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Access: api.Http(), api.Models(), api.Translate()
    }
}
```

**Key Differences:**

| Aspect | Core | Plugin |
|--------|------|--------|
| Parameter | `g *api.CoreGlobals` | `api sdkapi.IPluginApi` |
| HTTP API | `g.CoreAPI.HttpAPI` | `api.Http()` |
| Database | `g.Models` | `api.Models()` |
| Translation | `g.CoreAPI.Translate()` | `api.Translate()` |
| Theme Access | `g.PluginMgr.GetAdminTheme()` | N/A |

## Routing Patterns

### ⚠️ ALWAYS Use Group() for Route Organization

```go
// Core routes
func RegisterRoutes(g *api.CoreGlobals) {
    adminR := g.CoreAPI.HttpAPI.Router().AdminRouter()
    portalR := g.CoreAPI.HttpAPI.Router().PluginRouter()

    adminR.Group("/devices", func(subrouter sdkapi.IHttpRouterInstance) {
        subrouter.Get("/", ctrl.DevicesListCtrl(g)).Name("admin:devices:index")
        subrouter.Get("/{id}", ctrl.DeviceShowCtrl(g)).Name("admin:devices:show")
        subrouter.Post("/delete/{id}", ctrl.DeviceDeleteCtrl(g)).Name("admin:devices:delete")
    })

    portalR.Group("/payments", func(subrouter sdkapi.IHttpRouterInstance) {
        subrouter.Get("/options", ctrl.PaymentOptionsCtrl(g)).Name("payments:options")
        subrouter.Post("/process", ctrl.ProcessPaymentCtrl(g)).Name("payments:process")
    })
}
```

```go
// Plugin routes
func SetupRoutes(api sdkapi.IPluginApi) {
    adminR := api.Http().Router().AdminRouter()
    pluginR := api.Http().Router().PluginRouter()

    adminR.Group("/inventory", func(subrouter sdkapi.IHttpRouterInstance) {
        subrouter.Get("/", controllers.InventoryListCtrl(api)).Name("admin:inventory:index")
        subrouter.Get("/form", controllers.InventoryFormCtrl(api)).Name("admin:inventory:form")
        subrouter.Post("/save", controllers.InventorySaveCtrl(api)).Name("admin:inventory:save")
    })
}
```

### Route Naming Convention
- **Admin routes**: `admin:` prefix → `admin:sessions:index`, `admin:vouchers:create`
- **Portal routes**: No `admin:` prefix → `portal:sse`, `payments:options`
- **Auth routes**: `auth:` prefix → `auth:login`, `admin:auth:logout`
- **Internal format**: `{plugin-package}#{route-name}` → `com.example.plugin#admin:sessions:index`

### Built-in Middlewares
```go
api.Http().Middlewares().AdminAuth()       // Require authentication
api.Http().Middlewares().Device()          // Track client device
api.Http().Middlewares().HTTPSRedirect()   // Force HTTPS
api.Http().Middlewares().TrackNav()        // Track navigation
api.Http().Middlewares().CacheResponse(days)
api.Http().Middlewares().WebhookAuth()     // Verify JWT
api.Http().Middlewares().PendingPurchase()
```

## URL Generation

```go
// Plugin - within same plugin
url := api.Http().Router().UrlForRoute("admin:sessions:index")
url := api.Http().Router().UrlForRoute("admin:sessions:show", "id", "123")

// Plugin - cross-plugin
url := api.Http().Router().UrlForPkgRoute("com.other.plugin", "admin:feature:index")

// Core
url := g.CoreAPI.HttpAPI.Router().UrlForRoute("admin:dashboard")
url := g.CoreAPI.HttpAPI.Router().UrlForRoute("admin:devices:show", "id", "123")

// Legacy (still works)
url := api.Http().Helpers().UrlForRoute("admin:sessions:index")
```

**In templ views:**
```templ
<a href={ templ.URL(api.Http().Router().UrlForRoute("admin:sessions:show", "id", fmt.Sprint(session.ID))) }>
    { api.Translate("label", "View Session") }
</a>
```

## View Rendering

### 1. AdminView - Full admin layout
```go
func AdminCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        content := views.AdminPage(api, data)
        page := sdkapi.ViewPage{
            PageContent: content,
            PageCss:     "admin.css",  // Optional
            PageJs:      "admin.js",   // Optional
        }
        api.Http().Response().AdminView(w, r, page)
    }
}
```

### 2. PortalView - Captive portal layout
```go
api.Http().Response().PortalView(w, r, page)
```

### 3. View - Partial HTML (for htmx)
```go
api.Http().Response().View(w, r, page)
```

### 4. JSON Response
```go
api.Http().Response().Json(w, r, data, http.StatusOK)
```

### Flash Messages & Redirects
```go
// Flash messages (types: FlashMsgSuccess, FlashMsgError, FlashMsgWarning, FlashMsgInfo)
api.Http().Response().FlashMsg(w, r, api.Translate("error", "Failed to save"), sdkapi.FlashMsgError)

// Redirects (auto-detects htmx and sets HX-Redirect header)
api.Http().Response().Redirect(w, r, "admin:sessions:index")
api.Http().Response().Redirect(w, r, "admin:sessions:show", "id", "123")
api.Http().Response().RedirectToPortal(w, r)
```

## Controller Examples

### Core Controller with Theme
```go
func AdminDashboardCtrl(g *api.CoreGlobals) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Get theme
        p, t, err := g.PluginMgr.GetAdminTheme()
        if err != nil {
            errMsg := g.CoreAPI.Translate("error", "Unable to Get Admin Theme")
            g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(errMsg), http.StatusInternalServerError)
            return
        }

        // Fetch data
        ctx := r.Context()
        sessions, _ := g.Models.Session().ListActive(ctx)
        devices, _ := g.Models.Device().List(ctx)

        // Render
        page := t.AdminTheme.DashboardPageFactory(w, r, sessions, devices)
        p.Http().Response().AdminView(w, r, page)
    }
}
```

### Plugin Controller with Form Processing
```go
func CreateResourceCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Parse form
        if err := r.ParseForm(); err != nil {
            api.Http().Response().Error(w, r, err, http.StatusBadRequest)
            return
        }

        // Validate
        validator := api.Http().Forms().NewValidator()
        validator.Required("name", r.FormValue("name"))
        validator.MinLength("name", r.FormValue("name"), 3)
        validator.Email("email", r.FormValue("email"))

        if !validator.Valid() {
            content := views.ResourceForm(api, validator.Errors(), r.Form)
            api.Http().Response().AdminView(w, r, sdkapi.ViewPage{PageContent: content})
            return
        }

        // Create resource
        resource, err := api.Models().Resource().Create(ctx, db.CreateResourceParams{
            Name:  r.FormValue("name"),
            Email: r.FormValue("email"),
        })
        if err != nil {
            api.Http().Response().FlashMsg(w, r, api.Translate("error", "Failed to create"), sdkapi.FlashMsgError)
            api.Http().Response().Redirect(w, r, "admin:resources:new")
            return
        }

        api.Http().Response().FlashMsg(w, r, api.Translate("success", "Created successfully"), sdkapi.FlashMsgSuccess)
        api.Http().Response().Redirect(w, r, "admin:resources:show", "id", resource.ID)
    }
}
```

### HTMX Partial Controller
```go
func SessionSummaryCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        client, err := api.Http().GetClientDevice(r)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }

        session, err := api.Models().Session().FindActiveByDevice(ctx, client.Id())
        if err != nil {
            content := views.NoActiveSession(api)
            api.Http().Response().View(w, r, sdkapi.ViewPage{PageContent: content})
            return
        }

        content := views.SessionSummary(api, session)
        api.Http().Response().View(w, r, sdkapi.ViewPage{PageContent: content})
    }
}
```

## Database Integration

### Using Models
```go
ctx := r.Context()
models := api.Models()

// Create
session, err := models.Session().Create(ctx, queries.CreateSessionParams{...})

// Read
session, err := models.Session().FindByID(ctx, id)
sessions, err := models.Session().List(ctx)

// Update
err := models.Session().Update(ctx, id, params)

// Delete
err := models.Session().Delete(ctx, id)
```

### Database Model Methods
```go
// core/db/models/session-model.go
func (m *SessionModel) FindExpiringSoon(ctx context.Context, days int) ([]*queries.Session, error) {
    // SQLite date handling
    query := `SELECT * FROM sessions WHERE datetime('now') > datetime(started_at, '+' || (exp_days - ?) || ' days')`
    // Implementation details...
}
```

## HTTP Helpers API

```go
helpers := api.Http().Helpers()

// Translation (ALWAYS use for user-facing text)
text := helpers.Translate("msgtype", "Welcome Message")
errorText := helpers.Translate("error", "Invalid Input", "field", "email")

// CSRF protection
csrfField := helpers.CsrfHtmlTag(r) // <input type="hidden" name="csrf_token" value="...">

// Asset paths
adminCss := helpers.AdminAssetPath("styles.css")
portalJs := helpers.PortalAssetPath("portal.js")
publicImg := helpers.PublicPath("logo.png")

// URL generation
url := helpers.UrlForRoute("admin:sessions:index")
crossPluginUrl := helpers.UrlForPkgRoute("com.other.plugin", "feature:index")

// Plugin manager
pluginMgr := helpers.PluginMgr()
theme, _ := pluginMgr.GetAdminTheme()
```

## Authentication & Other APIs

```go
// Authentication
acct, err := api.Http().Auth().IsAuthenticated(r)
acct, err := api.Http().Auth().CurrentAccount(r)
err := api.Http().Auth().SignIn(w, username, password)
err := api.Http().Auth().SignOut(w)

// Route parameters
vars := api.Http().MuxVars(r)
id := vars["id"]

// Client device
client, err := api.Http().GetClientDevice(r)

// Cookie management
api.Http().Cookie().SetCookie(w, "name", "value")
value, err := api.Http().Cookie().GetCookie(r, "name")
api.Http().Cookie().DeleteCookie(w, "name")

// Form validation
validator := api.Http().Forms().NewValidator()
validator.Required("email", r.FormValue("email"))
validator.Email("email", r.FormValue("email"))
validator.MinLength("password", r.FormValue("password"), 8)
validator.MaxLength("username", r.FormValue("username"), 50)
validator.Match("password_confirm", r.FormValue("password"), r.FormValue("password_confirm"))

if !validator.Valid() {
    errors := validator.Errors() // map[string]string
}
```

## Best Practices

### Route Organization
- ✅ **ALWAYS use `Group()`** to organize routes by feature
- ✅ Use naming pattern: `section:resource:action`
- ✅ Name all routes for URL generation
- ✅ Apply middlewares at router level when possible

### Controller Design
- ✅ One responsibility per controller
- ✅ Validate input early
- ✅ Use `r.Context()` for database operations
- ✅ Handle errors gracefully with appropriate status codes
- ✅ Return early on errors

### Translations
- ⚠️ **CRITICAL: ALWAYS use translations for user-facing text** (except debug logs)
- ✅ Flash messages: `api.Translate("error", "Failed to save")`
- ✅ Error messages: `api.Translate("error", "Resource Not Found")`
- ✅ Form labels: `api.Translate("label", "Username")`
- ✅ Debug logs can remain in English (not user-facing)

### Database Access
- ✅ Use `api.Models()`, not direct database access
- ✅ Always pass `r.Context()` to database methods
- ✅ Use transactions for multi-step operations
- ✅ Handle `sql.ErrNoRows` separately
- ⚠️ **CRITICAL: NEVER modify core migrations** (`core/resources/migrations/`)
  - Plugins must have own migrations: `data/plugins/local/{plugin}/resources/migrations/`
  - Plugin migrations only create plugin-specific tables
  - Use foreign keys to reference core tables, never alter them
  - Use JOIN queries instead of modifying core schema

### Security
- ✅ Use CSRF protection: `helpers.CsrfHtmlTag(r)`
- ✅ Apply `AdminAuth()` middleware to protected routes
- ✅ Validate and sanitize all user input
- ✅ Use HTTPS redirect for sensitive pages

---

**Remember:** This agent plans first, then implements after user confirmation. Always use existing patterns from the codebase.
