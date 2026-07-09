# IHttpRouterApi
The `IHttpRouterApi` is the backend for http routing in Flarewifi. There are two (2) router methods, each configured with an options struct:

- `AdminRouter(opts *AdminRouterOpts)` - a router for the admin pages of the plugin that requires admin authentication and is always served over HTTPS.
- `HttpRouter(opts *HttpRouterOpts)` - a general-purpose router for the plugin, with no authentication.

The options structs select a variant of each router:

```go
type AdminRouterOpts struct {
    Static bool // Routes persist across plugin version updates (/admin/static/{package}/{path}).
}

type HttpRouterOpts struct {
    HttpsOnly bool // Routes are served only over HTTPS (plain-HTTP is redirected to HTTPS).
    Static    bool // Routes persist across plugin version updates (/p/static/{package}/{path}).
}
```

Passing `nil` (or a zero-value struct) returns the default variant. For `HttpRouter`, `Static` takes precedence over `HttpsOnly` when both are set.

> **Migration from the old API:** the former `PluginRouter()`, `HttpsRouter()`, `StaticPluginRouter()`, and `StaticAdminRouter()` methods have been removed in favor of the two opts-based methods. Update your calls as follows:
>
> | Old call | New call |
> |----------|----------|
> | `AdminRouter()` | `AdminRouter(nil)` |
> | `PluginRouter()` / `HttpRouter()` | `HttpRouter(nil)` |
> | `HttpsRouter()` | `HttpRouter(&sdkapi.HttpRouterOpts{HttpsOnly: true})` |
> | `StaticPluginRouter()` | `HttpRouter(&sdkapi.HttpRouterOpts{Static: true})` |
> | `StaticAdminRouter()` | `AdminRouter(&sdkapi.AdminRouterOpts{Static: true})` |

---

## IHttpRouterApi methods {#http-router-api}

Below are the available methods in `IHttpRouterApi`:

### Admin Router {#admin-router}
This method returns the [admin router](#router-instance) for the admin routes. Routes generated from the admin router are prefixed with `/admin` and are only accessible to authenticated user [accounts](./accounts-api.md#account-instance). Admin routes are always served over HTTPS. To get the admin router instance:

```go
adminRouter := api.Http().Router().AdminRouter(nil)
```

To get a **version-independent** admin router whose paths persist across plugin version updates (accessible at `/admin/static/{package}/{path}`), pass `Static: true`:

```go
staticAdminRouter := api.Http().Router().AdminRouter(&sdkapi.AdminRouterOpts{Static: true})
```

### Http Router {#plugin-router}
This method returns the [plugin router](#router-instance) of the plugin. Routes generated from the http router are accessible to all users over either scheme (HTTP or HTTPS) with no authentication middleware. To get the http router instance:

```go
httpRouter := api.Http().Router().HttpRouter(nil)
```

The `HttpRouterOpts` select a variant of the same router:

```go
// HTTPS-only — plain-HTTP requests are redirected to HTTPS. Use for public
// endpoints that must run over TLS (secure callbacks, webhooks, login posts).
// Accessible at /p/https/{package}/{version}/{path}.
httpsRouter := api.Http().Router().HttpRouter(&sdkapi.HttpRouterOpts{HttpsOnly: true})

// Static — routes persist across plugin version updates. Use for public-facing
// URLs that must remain stable, such as payment callbacks or QR code links.
// Accessible at /p/static/{package}/{path}.
staticRouter := api.Http().Router().HttpRouter(&sdkapi.HttpRouterOpts{Static: true})
```

### UseForPortal {#use-for-portal}

This method is used to register middlewares that will wrap the captive portal index page handler (`/portal/index`). This is useful for plugins that need to track portal page views, add custom authentication, implement rate limiting, or perform other operations before the portal page is rendered.

**Important Notes:**

- Middlewares registered with `UseForPortal` must be registered during plugin initialization (in the `Init()` function)
- These middlewares only apply to the `/portal/index` route, not other portal routes
- Multiple plugins can register middlewares, and they will all be executed in the order they were registered
- Plugin middlewares execute **before** core middlewares (HTTPRedirect, PendingPurchase)

**Middleware Execution Order:**

1. Plugin-registered middlewares (via `UseForPortal`) - **FIRST**
2. HTTPRedirect middleware (redirects HTTPS to HTTP)
3. PendingPurchase middleware (checks for pending purchases)
4. Portal index page handler - **LAST**

**Example - Analytics Tracking:**

```go
package main

import (
    "net/http"
    sdkapi "github.com/flarewifi/sdk-api"
)

var Api sdkapi.IPluginApi

func Init(api sdkapi.IPluginApi) error {
    Api = api
    
    // Register middleware for portal index page
    api.Http().Router().UseForPortal(analyticsMiddleware)
    
    return nil
}

func analyticsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Track portal page view
        Api.Logger().Info("Portal index page accessed from: " + r.RemoteAddr)
        
        // Continue to next middleware/handler
        next.ServeHTTP(w, r)
    })
}
```

**Example - Custom Rate Limiting:**

```go
func Init(api sdkapi.IPluginApi) error {
    Api = api
    
    // Register rate limiting middleware
    api.Http().Router().UseForPortal(rateLimitMiddleware)
    
    return nil
}

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check rate limit
        if isRateLimited(r.RemoteAddr) {
            http.Error(w, "Too many requests", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### ClaimPortalTraffic {#claim-portal-traffic}

This method registers *claim middlewares* that the core web stack runs on **every top-level page navigation**, before any HTTPS forcing or captive-portal funnel routing. A claim uses the standard middleware signature `func(next http.Handler) http.Handler`: it inspects the request — typically the client IP from `r.RemoteAddr` — and either writes the response itself, taking **full ownership** of the navigation for any path and Host, or calls `next` to leave the request to the normal funnel/portal flow.

Use this for clients that cannot go through the normal captive-portal device registration at all — for example routed PPPoE subscribers, whose IPs are not on the machine's L2 segment, so they have no ARP/MAC visibility and can never register a client device. `UseForPortal` cannot reach such clients (it wraps only `/portal/index`, behind the device-registration gate); `ClaimPortalTraffic` runs before everything.

**Important Notes:**

- Register claims once, during plugin initialization (in the `Init()` function).
- Claims run on every page navigation, so the pass-through decision must be **fast** — decide from in-memory state (a cached IP set/map), not per-request database or network lookups. Do the heavier data loading only after deciding the request is yours.
- Sub-resource requests (assets, XHR/EventSource, favicons) are **never** claimed — a claimed page can reference core-served CSS/JS by absolute path and they load normally.
- A claimed request bypasses HTTPS forcing, the captive funnel, and device registration entirely; your middleware owns the response.
- When several plugins register claims, they nest in registration order (first registered runs outermost); a claim that does not call `next` ends the chain.
- Claims behave identically on dev and production builds, so the flow is fully testable in the dev container.

**Example - Redirecting suspended subscribers to a plugin page:**

```go
func Init(api sdkapi.IPluginApi) error {
    api.Http().Router().ClaimPortalTraffic(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip, _, err := net.SplitHostPort(r.RemoteAddr)
            if err != nil {
                ip = r.RemoteAddr
            }

            // Fast in-memory lookup — populated elsewhere by the plugin
            client, suspended := lookupSuspendedIP(ip)
            if !suspended {
                next.ServeHTTP(w, r) // not ours — normal portal/funnel flow continues
                return
            }

            page := views.SuspendedPage(Api, client)
            Api.Http().Response().PortalView(w, r, sdkapi.ViewPage{PageContent: page})
        })
    })

    return nil
}
```

### UrlForRoute

This method generates the URL for a route by name. It automatically resolves the correct URL whether the route was registered on a versioned or static router — no separate method needed for static routes.

```go
// Works for both versioned and static routes:
url := api.Http().Router().UrlForRoute("purchase:wifi", "plan", "basic")
```

### UrlForPkgRoute

This method generates the URL for a route registered in a different plugin. It accepts the target plugin's package name, the route name, and optional route parameters. Like `UrlForRoute`, it automatically resolves versioned or static routes.

```go
url := api.Http().Router().UrlForPkgRoute("com.mydomain.myplugin", "portal.welcome", "name", "John")
```

---

## IHttpRouterInstance {#router-instance}

`IHttpRouterInstance` is a router instance used to generate routes for the plugin. The same instance type is returned by `AdminRouter(...)` and `HttpRouter(...)` regardless of the options passed. Below are the methods available in the router instance:

### Group

This method is used to create a group of routes with a common path prefix. This method accepts two arguments,
the first argument is the path prefix and the second argument is a function which accepts another router instance.
This can be used to nest more route groups. Take a look at the example below:

```go
// Get the router instance
router := api.Http().Router().HttpRouter(nil)
router.Group("/payments", func (subrouter sdkhttp.HttpRouterInstance) {
    // Add route for /payments/received
    subrouter.Post("/receieved", func(w http.ResponseWriter, r *http.Request) {
        // Handle the request
    })
})
```

### Get

This method is used to create a route for the `GET` http method. It accepts tree (or more) arguments, the first argument is the route path, the second argument is the [handler function](#handler-function) and the third and subsequent arguments are the list of (optional) [middlewares](#middlewares). Take a look at the example below:

```go
router := api.Http().Router().HttpRouter(nil)
router.Get("/payments/options", func(w http.ResponseWriter, r *http.Request) {
    // Handle the request
}).Name("payments.options") // set the route name
```

### Post

This method is used to create a route for the `POST` http method. It accepts three (or more) arguments, the first argument is the route path, the second argument is the [handler function](#handler-function) and the third and subsequent arguments are the list of (optional) [middlewares](#middlewares). Take a look at the example below:

```go
router := api.Http().Router().AdminRouter(nil)
router.Post("/settings/save", func(w http.ResponseWriter, r *http.Request) {
    // Handle the request
}).Name("settings.save") // set the route name
```

### Use

This method is used to add a [middleware](#middlewares) to the router. It accepts a list of middlewares.
All routes defined after the `Use` method will use the middleware.

---

## IHttpRoute {#http-route}

`IHttpRoute` is returned by `Get()` and `Post()` and allows you to configure the route after registration.

### Name {#name}

Sets the name of the route. Route names for admin routes must be prefixed with `admin:`. Named routes can be resolved to URLs using [UrlForRoute](#urlforroute) regardless of whether they are versioned or static.

The same `Name()` method is used for both versioned and static routes — the correct URL namespace is applied automatically based on whether the route was registered on a static or versioned router.

```go
// Versioned route
router := api.Http().Router().HttpRouter(nil)
router.Get("/purchase/wifi", handler).Name("purchase:wifi")

adminRouter := api.Http().Router().AdminRouter(nil)
adminRouter.Get("/sessions", handler).Name("admin:sessions:list")

// Static route — same Name() method
staticRouter := api.Http().Router().HttpRouter(&sdkapi.HttpRouterOpts{Static: true})
staticRouter.Get("/purchase/wifi", handler).Name("purchase:wifi")
// accessible at: /p/static/com.mydomain.myplugin/purchase/wifi

staticAdminRouter := api.Http().Router().AdminRouter(&sdkapi.AdminRouterOpts{Static: true})
staticAdminRouter.Get("/sessions/export", handler).Name("admin:sessions:export")
// accessible at: /admin/static/com.mydomain.myplugin/sessions/export
```

### Queries

Adds URL query parameter constraints to the route. Only requests that include the specified query parameters will match this route.

```go
router.Get("/search", handler).
    Queries("q", "{query}").
    Name("portal:search")
```

---

## Handler Function {#handler-function}

A handler function is a function that executes when a URL pattern in the HTTP request is matched.
Below is an example of a handler function:

```go
func(w http.ResponseWriter, r *http.Request) {
    // Handler function code here...
}
```

### Using a handler function

Handler functions are used when you register a route to a router. An example below is a handler function that gets executed when a user navigates to `/welcome` URL.

```go
pluginRouter := api.Http().Router().HttpRouter(nil)
pluginRouter.Get("/welcome", func (w http.ResponseWriter, r *http.Request) {
    // Handler function code here...
    w.Write([]byte("Welcome to Flarewifi!"))
}).Name("portal:welcome")
```

---

## Middlewares {#middlewares}

A middleware is used to perform operations on the HTTP request before it reaches the [handler function](#handler-function). Middlewares are functions that accept a http handler function and returns another http handler function: `func(next http.Handler) http.Handler`

### Declaring a middleware

Below is an example of a middleware:

```go
middleware := func (next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // do something before the handler function
        next.ServeHTTP(w, r)
    })
}
```

### Using a middleware

Below is using a middleware for plugin sub-router:

```go
api.Http().Router().HttpRouter(nil).Group("/payments", func (subrouter IHttpRouterInstance) {
    subrouter.Use(middleware)
})
```

Below is using a middleware for admin sub-router:

```go
api.Http().Router().AdminRouter(nil).Group("/settings", func (subrouter IHttpRouterInstance) {
    subrouter.Use(middleware)
})
```

In the examples above, the middleware is used to perform operations on the request before it reaches the handler function inside the sub-router. But it can also be used directly on the [HttpRouter](#plugin-router) or the [AdminRouter](#admin-router).

---

## Static Routers {#static-routers}

A **static** router (selected with `Static: true`) provides version-independent URLs that persist across plugin version updates. Use them for any URL that must remain stable over time — payment callbacks, QR code links, webhook endpoints, or bookmarkable admin pages.

### Path structure

| Router call | Path pattern |
|-------------|-------------|
| `HttpRouter(nil)` | `/p/{package}/{version}/{path}` |
| `HttpRouter(&HttpRouterOpts{HttpsOnly: true})` | `/p/https/{package}/{version}/{path}` |
| `HttpRouter(&HttpRouterOpts{Static: true})` | `/p/static/{package}/{path}` |
| `AdminRouter(nil)` | `/admin/{package}/{version}/{path}` |
| `AdminRouter(&AdminRouterOpts{Static: true})` | `/admin/static/{package}/{path}` |

### Example — stable payment callback

```go
func Init(api sdkapi.IPluginApi) error {
    // This URL stays the same even after a plugin version bump:
    // /p/static/com.mydomain.myplugin/payments/callback
    api.Http().Router().HttpRouter(&sdkapi.HttpRouterOpts{Static: true}).
        Post("/payments/callback", handlePaymentCallback).
        Name("payments:callback")

    return nil
}

func handlePaymentCallback(w http.ResponseWriter, r *http.Request) {
    // Handle payment gateway POST notification
}
```

### Example — stable admin export endpoint

```go
func Init(api sdkapi.IPluginApi) error {
    // /admin/static/com.mydomain.myplugin/sessions/export
    api.Http().Router().AdminRouter(&sdkapi.AdminRouterOpts{Static: true}).
        Get("/sessions/export", handleExport).
        Name("admin:sessions:export")

    return nil
}
```

### Generating static URLs

Use the same `UrlForRoute` / `UrlForPkgRoute` you already use for versioned routes — the correct static URL is resolved automatically:

```go
// In a handler:
url := api.Http().Router().UrlForRoute("payments:callback")
// → /p/static/com.mydomain.myplugin/payments/callback

// In a templ template:
templ.SafeURL(api.Http().Helpers().UrlForRoute("payments:callback"))

// Linking to another plugin's static route:
templ.SafeURL(api.Http().Helpers().UrlForPkgRoute("com.other.plugin", "payments:callback"))
```
