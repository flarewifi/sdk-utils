# IHttpRouterApi
The `IHttpRouterApi` is the backend for http routing in Flare Hotspot. There are two (2) kinds of http routers:

- `AdminRouter` - a router for the admin pages of the plugin that requires admin authentication.
- `PluginRouter` - a router for general purpose routing within the plugin

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

### UrlForRoute

This method is used to generate the url for the given plugin route name. This method accepts two arguments, the first argument is the route name and the second argument is a map of route parameters. The route parameters are key-value pairs. The example below generates a URL for the route name `portal.welcome` with a route path `/welcome/{name}`:

```go
url := api.Http().Router().UrlForRoute("portal.welcome", "name", "John")
```

### UrlForPkgRoute

This method is used to generate the url for third-party plugin route name. This method accepts three arguments, the first argument is the plugin package name (e.g `com.mydomain.myplugin`) in which the route name belongs to, the second argument is the route name and the third argument is the route parameters, similar to [UrlForRoute](#urlforroute) method.

```go
url := api.Http().Router().UrlForPkgRoute("com.mydomain.myplugin", "portal.welcome", "name", "John")
```

---

## IHttpRouterInstance {#router-instance}

`IHttpRouterInstance` is a router instance used to generate routes for the plugin. Routes can be generated using a [PluginRouter](#plugin-router) or an [AdminRouter](#admin-router). Below are the methods available in the router instance:

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
    w.Write([]byte("Welcome to Flare Hotspot!"))
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
