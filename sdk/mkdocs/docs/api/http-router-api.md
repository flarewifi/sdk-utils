# IHttpRouterApi
The `IHttpRouterApi` is the backend for http routing in Flarewifi. There are four (4) kinds of http routers:

- `AdminRouter` - a router for the admin pages of the plugin that requires admin authentication.
- `PluginRouter` - a router for general purpose routing within the plugin.
- `StaticAdminRouter` - a version-independent admin router whose paths persist across plugin version updates.
- `StaticPluginRouter` - a version-independent plugin router whose paths persist across plugin version updates.

---

## IHttpRouterApi methods {#http-router-api}

Below are the available methods in `IHttpRouterApi`:

### Admin Router {#admin-router}
This method returns the [admin router](#router-instance) for the admin routes. Routes generated from the admin router are prefixed with `/admin` and are only accessible to authenticated user [accounts](./accounts-api.md#account-instance). To get the admin router instance:

```go
adminRouter := api.Http().Router().AdminRouter()
```

### Plugin Router {#plugin-router}
This method returns the [plugin router](#router-instance) of the plugin. Routes generated from the plugin router are accessible to all users. To get the plugin router instance:

```go
pluginRouter := api.Http().Router().PluginRouter()
```

### Use {#use}

This method is used to add a global middleware to all routes. It accepts a list of middlewares.
```go
middleware := func (next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // do something before the handler function
        next.ServeHTTP(w, r)
    })
}

api.Http().Router().Use(middleware)
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
    sdkapi "github.com/flarehotspot/sdk-api"
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

### Static Admin Router {#static-admin-router}

This method returns the [static admin router](#router-instance) for the plugin. Routes registered here are accessible at `/admin/static/{package}/{path}` and **persist across plugin version updates**. Like the regular admin router, these routes require admin authentication. Use static routes for URLs you want to remain stable even after bumping the plugin version — for example, webhook endpoints or external integrations.

```go
staticAdminRouter := api.Http().Router().StaticAdminRouter()
```

### Static Plugin Router {#static-plugin-router}

This method returns the [static plugin router](#router-instance) for the plugin. Routes registered here are accessible at `/p/static/{package}/{path}` and **persist across plugin version updates**. Use static routes for public-facing URLs that must remain stable, such as payment callbacks or QR code links.

```go
staticPluginRouter := api.Http().Router().StaticPluginRouter()
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

`IHttpRouterInstance` is a router instance used to generate routes for the plugin. Routes can be generated using a [PluginRouter](#plugin-router), an [AdminRouter](#admin-router), a [StaticPluginRouter](#static-plugin-router), or a [StaticAdminRouter](#static-admin-router). Below are the methods available in the router instance:

### Group

This method is used to create a group of routes with a common path prefix. This method accepts two arguments,
the first argument is the path prefix and the second argument is a function which accepts another router instance.
This can be used to nest more route groups. Take a look at the example below:

```go
// Get the router instance
router := api.Http().Router().PluginRouter()
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
router := api.Http().Router().PluginRouter()
router.Get("/payments/options", func(w http.ResponseWriter, r *http.Request) {
    // Handle the request
}).Name("payments.options") // set the route name
```

### Post

This method is used to create a route for the `POST` http method. It accepts three (or more) arguments, the first argument is the route path, the second argument is the [handler function](#handler-function) and the third and subsequent arguments are the list of (optional) [middlewares](#middlewares). Take a look at the example below:

```go
router := api.Http().Router().AdminRouter()
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
router := api.Http().Router().PluginRouter()
router.Get("/purchase/wifi", handler).Name("purchase:wifi")

adminRouter := api.Http().Router().AdminRouter()
adminRouter.Get("/sessions", handler).Name("admin:sessions:list")

// Static route — same Name() method
staticRouter := api.Http().Router().StaticPluginRouter()
staticRouter.Get("/purchase/wifi", handler).Name("purchase:wifi")
// accessible at: /p/static/com.mydomain.myplugin/purchase/wifi

staticAdminRouter := api.Http().Router().StaticAdminRouter()
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
pluginRouter := api.Http().Router().PluginRouter()
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
api.Http().Router().PluginRouter().Group("/payments", func (subrouter IHttpRouterInstance) {
    subrouter.Use(middleware)
})
```

Below is using a middleware for admin sub-router:

```go
api.Http().Router().AdminRouter().Group("/settings", func (subrouter IHttpRouterInstance) {
    subrouter.Use(middleware)
})
```

In the examples above, the middleware is used to perform operations on the request before it reaches the handler function inside the sub-router. But it can also be used directly on the [PluginRouter](#plugin-router) or the [AdminRouter](#admin-router).

---

## Static Routers {#static-routers}

Static routers provide version-independent URLs that persist across plugin version updates. Use them for any URL that must remain stable over time — payment callbacks, QR code links, webhook endpoints, or bookmarkable admin pages.

### Path structure

| Router | Path pattern |
|--------|-------------|
| `PluginRouter` | `/p/{package}/{version}/{path}` |
| `AdminRouter` | `/admin/{package}/{version}/{path}` |
| `StaticPluginRouter` | `/p/static/{package}/{path}` |
| `StaticAdminRouter` | `/admin/static/{package}/{path}` |

### Example — stable payment callback

```go
func Init(api sdkapi.IPluginApi) error {
    // This URL stays the same even after a plugin version bump:
    // /p/static/com.mydomain.myplugin/payments/callback
    api.Http().Router().StaticPluginRouter().
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
    api.Http().Router().StaticAdminRouter().
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
