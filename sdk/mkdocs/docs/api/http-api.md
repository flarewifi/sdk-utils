# HttpApi

The `HttpApi` is used to access various HTTP server functionalities including authentication, routing, and http responses.

## 1. HttpApi methods {#httpapi-methods}

The following are the available methods in `HttpApi`:

### GetClientDevice

Get the [IClientDevice](./client-device.md) from the http request:

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    clnt, err := api.Http().GetClientDevice(r)
    fmt.Println(clnt) // IClientDevice
}
```

### Auth

It returns an instance of the [IHttpAuth](./http-auth.md).

```go
auth := api.Http().Auth()
```

### Cookie

It returns an instance of [IHttpCookie](./http-cookie.md).

### Middlewares

It returns an instance of [IHttpMiddlewares](./http-router-api.md#middlewares) that contains the built-in middlewares.

```go
middlewares := api.Http().Middlewares()
```

### Helpers

It returns an instance of the [IHttpHelpers](./http-helpers.md).

```go
helpers := api.Http().Helpers()
```

### Router

It returns an instance of [IHttpRouterApi](./http-router-api.md).

```go
httpRouter := api.Http().Router()
```

### Response

Returns an instance of [IHttpResponse](./http-response.md).

```go
httpResponse := api.Http().Response()
```

### MuxVars

Returns a `map[string]string` of variables from the request path.

Below is an example to get the value if `id` in the route path `/sessions/:id`

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    // other logic...
    vars := api.Http().MuxVars(r) // map[string]string
    id := vars["id"]
    fmt.Println(id) // "1"
}
```

### Navs

Returns an instance of [INavsApi](./http-navs-api.md).

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    navsAPI := api.Http().Navs()
    fmt.Println(navsAPI) // INavsApi
}
```

### Forms

Returns an instance of [IHttpFormsApi](./http-forms-api.md)

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    fmt.Println(formsAPI) // IHttpFormsApi
}
```
