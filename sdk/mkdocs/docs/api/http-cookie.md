# IHttpCookie

The `IHttpCookie` provides methods to set and read cookie values.

## IHttpCookie methods

Below are the methods available in `IHttpCookie`:

### SetCookie

Sets the cookie value for a given cookie name

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    cookieAPI := api.Http().Cookie()
    cookieAPI.SetCookie(w, "auth-token", "**token-string**")
    http.WriteHeader(http.StatusOK)
}
```

### GetCookie

Returns the string value of the cookie

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    cookieAPI := api.Http().Cookie()
    token, err := cookieAPI.GetCookie(r, "auth-token")
    if err != nil {
        // handle error
    }
    fmt.Println(token) // auth token
}
```

### DeleteCookie

Deletes the cookie value for a given cookie name
```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    cookieAPI := api.Http().Cookie()
    cookieAPI.DeleteCookie(w, "auth-token")
    http.WriteHeader(http.StatusOK)
}
```
