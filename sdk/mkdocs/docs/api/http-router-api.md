# IHttpRouterApi
The `IHttpRouterApi` is the backend for http routing in Flare Hotspot. There are two (2) kinds of http routers:

- `AdminRouter` - a router for the admin pages of the plugin that uses the [AdminAuth](./http-middlewares.md#admin-auth) middleware.
- `PluginRouter` - a router for general purpose routing within the plugin

## IHttpRouterApi methods {#http-router-api}

Below are the available methods in `IHttpRouterApi`:

### Admin Router {#admin-router}
This method returns the [admin router](#router-instance) for the admin routes. Routes generated from the admin router are prefixed with `/admin` and are only accessible to authenticated user [accounts](./accounts-api.md#account-instance). To get the admin router instance:

```go
adminRouter := api.Http().HttpRouter().AdminRouter()
```

### Plugin Router {#plugin-router}
This method returns the [plugin router](#router-instance) of the plugin. Routes generated from the plugin router are accessible to all users. To get the plugin router instance:

```go
pluginRouter := api.Http().HttpRouter().PluginRouter()
```

### Use {#use}

This method is used to add a global [middleware](#middlewares) to all routes. It accepts a list of [middlewares](./http-middlewares.md).
```go
middleware := func (next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // do something before the handler function
        next.ServeHTTP(w, r)
    })
}

api.Http().HttpRouter().Use(middleware)
```

### UrlForRoute

This method is used to generate the url for the given plugin route name. This method accepts two arguments, the first argument is the route name and the second argument is a map of route parameters. The route parameters are key-value pairs. The example below generates a URL for the route name `portal.welcome` with a route path `/welcome/:name`:

```go
url := api.Http().HttpRouter().UrlForRoute("portal.welcome", "name", "John")
```

### UrlForPkgRoute

This method is used to generate the url for third-party plugin route name. This method accepts three arguments, the first argument is the plugin package name (e.g `com.mydomain.myplugin`) in which the route name belongs to, the second argument is the route name and the third argument is the route parameters, similar to [UrlForRoute](#urlforroute) method.

```go
url := api.Http().HttpRouter().UrlForPkgRoute("com.mydomain.myplugin", "portal.welcome", "name", "John")
```

## IHttpRouterInstance {#router-instance}

`IHttpRouterInstance` is a router instance used to generate routes for the plugin. Routes can be generated using a [PluginRouter](#plugin-router) or an [AdminRouter](#admin-router). Below are the methods available in the router instance:

### Group

This method is used to create a group of routes with a common path prefix. This method accepts two arguments,
the first argument is the path prefix and the second argument is a function which accepts another router instance.
This can be used to nest more route groups. Take a look at the example below:

```go
// Get the router instance
router := api.Http().HttpRouter().PluginRouter()
router.Group("/payments", func (subrouter sdkhttp.HttpRouterInstance) {
    // Add route for /payments/received
    subrouter.Post("/receieved", func(w http.ResponseWriter, r *http.Request) {
        // Handle the request
    })
})
```

### Get

This method is used to create a route for the `GET` http method. It accepts tree (or more) arguments, the first argument is the route path, the second argument is the [handler function](../guides/form-submission.md#request-handler) and the third and subsequent arguments are the list of (optional) [middlewares](#middlewares). Take a look at the example below:

```go
router := api.Http().HttpRouter().PluginRouter()
router.Get("/payments/options", func(w http.ResponseWriter, r *http.Request) {
    // Handle the request
}).Name("payments.options") // set the route name
```

### Post

This method is used to create a route for the `POST` http method. It accepts three (or more) arguments, the first argument is the route path, the second argument is the [handler function](../guides/form-submission.md#request-handler) and the third and subsequent arguments are the list of (optional) [middlewares](#middlewares). Take a look at the example below:

```go
router := api.Http().HttpRouter().AdminRouter()
router.Post("/settings/save", func(w http.ResponseWriter, r *http.Request) {
    // Handle the request
}).Name("settings.save") // set the route name
```

### Use

This method is used to add a [middleware](#middlewares) to the router. It accepts a list of middlewares.
All routes defined after the `Use` method will use the middleware.

## Middlewares {#middlewares}

A [middleware](./http-middlewares.md) is a function of type `func(next http.Handler) http.Handler`. It is used to perform operations on the request before it reaches the handler function. Middlewares are functions that accept a http handler function and returns another http handler function.

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
api.Http().HttpRouter().PluginRouter().Group("/payments", func (subrouter sdkhttp.HttpRouterInstance) {
    subrouter.Use(middware)
})
```

Below is using a middleware for admin sub-router:

```go
api.Http().HttpRouter().AdminRouter().Group("/settings", func (subrouter sdkhttp.HttpRouterInstance) {
    subrouter.Use(middware)
})
```

In the examples above, the middleware is used to perform operations on the request before it reaches the handler function inside the sub-router. But it can also be used directly on the [PluginRouter](#plugin-router) or the [AdminRouter](#admin-router).
