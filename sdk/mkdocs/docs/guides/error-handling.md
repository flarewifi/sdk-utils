# Error Handling

There are two (2) ways to handle HTTP errors:

- Showing an error page
- Showing a flash message

## Error Page

To show an error page to users, we will use the [IHttpResponse.Error](../api/http-response.md#error) method:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    err := errors.New("Some error")
    api.Http().HttpResponse().Error(w, r, err, http.StatusInternalServerError)
}
```

## Flash Message

To show error message as flash message, we will use the [IHttpResponse.FlashMsg](../api/http-response.md#flashmsg) method:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    msg := "Some flash message"
    // Show the flash message
    api.Http().HttpResponse().FlashMsg(w, r, msg, sdkapi.FlashMsgSuccess)

    // Redirect back to the form page
    api.Http().HttpResponse().Redirect(w, r, "settings:form")
}
```
