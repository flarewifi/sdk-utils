# IHttpAuth

The `IHttpAuth` is used to authenticate and authorize admin users.

## IHttpAuth methods

The following are the available methods in `IHttpAuth`.

### CurrentAcct

It returns the current admin user [IAccount](./accounts-api.md#account-instance) instance from http request and an `error` if any. This method is only applicable on handlers registered on the [AdminRouter](./http-router-api.md#admin-router).

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    acct, err := api.Http().Auth().CurrentAcct(r)
    if err != nil {
        // handle error
    }
    fmt.Sprintf("Admin: %s", acct.Username) // IAccount
}
```

### IsAuthenticated

Checks if the user is authenticated. This will perform cookie checks and does not rely on the AdminAuth middleware.

### Authenticate

It authenticates an account using a username and password.
It returns an [IAccount](./accounts-api.md#account-instance) instance and an `error` if any.
This method is only applicable on handlers registered on the [HttpRouter](./http-router-api.md#plugin-router), otherwise the request is blocked by the authentication middleware.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    username := r.PostFormValue("username")
    password := r.PostFormValue("password")
    acct, err := api.Http().Auth().Authenticate(username, password)
    if err != nil {
        // handle error
    }
    // proceed to api.Http().Auth().SignIn()
}
```

### SignIn

It signs in an account with an [IAccount](./accounts-api.md#account-instance) instance by setting a cookie in the http response header.
It returns an `error` if any.
This method is only applicable on handlers registered on the [HttpRouter](./http-router-api.md#plugin-router), otherwise the request is blocked by the authentication middleware.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    acct, err := api.Http().Auth().Authenticate("admin", "admin")
    if err != nil {
        // handle error
    }

    // set cookie header in the http response
    err = api.Http().Auth().SignIn(w, acct)
    if err != nil {
        // handle error
    }
    w.WriteHeader(http.StatusOK)
}
```

### SignOut

It signs out an [Account](./accounts-api.md#account-instance) by removing the cookie from the http response header.
It returns an `error` if any. This method works on [HttpRouter](./http-router-api.md#plugin-router) and [AdminRouter](./http-router-api.md#admin-router).

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    err := api.Http().Auth().SignOut(w)
    if err != nil {
        // handle error
    }
    w.WriteHeader(http.StatusOK)
}
```
