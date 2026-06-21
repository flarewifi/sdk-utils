# IHttpCookie

The `IHttpCookie` provides methods to set and read cookie values.

## HttpCookieOpts

The `HttpCookieOpts` struct is used to configure optional cookie settings:

```go
type HttpCookieOpts struct {
    Path     string
    Expires  time.Time
    SameSite http.SameSite
}
```

- `Path string` - the URL path for which the cookie is valid
- `Expires time.Time` - the expiration time of the cookie
- `SameSite http.SameSite` - the SameSite attribute for the cookie (e.g., `http.SameSiteLaxMode`, `http.SameSiteStrictMode`, `http.SameSiteNoneMode`)

## IHttpCookie methods

Below are the methods available in `IHttpCookie`:

### SetCookie

Sets the cookie value for a given cookie name. The optional `opts` parameter allows configuring cookie settings.

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    cookieAPI := api.Http().Cookie()
    
    // Set cookie with default options (pass nil for opts)
    cookieAPI.SetCookie(w, "auth-token", "**token-string**", nil)
    
    // Or set cookie with custom options
    opts := &HttpCookieOpts{
        Path:     "/",
        Expires:  time.Now().Add(24 * time.Hour),
        SameSite: http.SameSiteLaxMode,
    }
    cookieAPI.SetCookie(w, "auth-token", "**token-string**", opts)
    
    http.WriteHeader(http.StatusOK)
}
```

### SetPlainCookie

Sets a plain (unencrypted) cookie value for a given cookie name. Unlike `SetCookie`, the value is stored as-is without any encoding.

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    cookieAPI := api.Http().Cookie()
    cookieAPI.SetPlainCookie(w, "lang", "en", nil)
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

### GetPlainCookie

Returns the plain (unencrypted) cookie value for a given cookie name.

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
    cookieAPI := api.Http().Cookie()
    lang, err := cookieAPI.GetPlainCookie(r, "lang")
    if err != nil {
        // handle error
    }
    fmt.Println(lang) // "en"
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
