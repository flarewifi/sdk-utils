# Error Handling

There are three (3) ways to handle HTTP errors:

- Showing an error page
- Showing a flash message
- Displaying form validation errors

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

To show error message as flash message, we will use the [IHttpResponse.FlashMsg](../api/http-response.md#flashmsg) method. After showing the message, redirect the user with [IHttpResponse.Redirect](../api/http-response.md#redirect):

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

## Form Validation Errors

To display form validation errors in your views, use the [IHttpFormsApi.Errors](../api/http-forms-api.md#errors) method to retrieve validation errors from a failed form submission. These errors can then be passed to your template for display.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    // Get form validation errors
    formErrors := api.Http().Forms().Errors(w, r, "my-form")

    // Render the form view with errors
    formView := views.MyForm(api, formErrors)
    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: formView,
    })
}
```

In your template, check for errors and display them:

```templ
templ MyForm(api sdkapi.IPluginApi, formErrors sdkapi.FormErrors) {
    <form>
        <div class="mb-3">
            <label for="username">Username:</label>
            <input
                type="text"
                id="username"
                name="username"
                class={ templ.Ternary(formErrors.HasError("username"), "form-control is-invalid", "form-control") }
            />
            if formErrors.HasError("username") {
                <div class="invalid-feedback">
                    { formErrors.GetError("username") }
                </div>
            }
        </div>
        <button type="submit" class="btn btn-primary">Submit</button>
    </form>
}
```

When form validation fails, use [IHttpFormsApi.ParseForm](../api/http-forms-api.md#parseform) to validate and redirect back to the form with errors preserved:

```go
// handler for form submission
func (w http.ResponseWriter, r *http.Request) {
    formValues, err := api.Http().Forms().ParseForm(w, r, formValidator)
    if err != nil {
        // Validation failed - redirect back to form
        api.Http().Response().Redirect(w, r, "my-form-route")
        return
    }

    // Process valid form data...
}
```

---

## Related

- [IHttpResponse](../api/http-response.md) — `Error`, `FlashMsg`, and `Redirect` methods
- [IHttpFormsApi](../api/http-forms-api.md) — `Errors` and `ParseForm` for form validation
- [Form Submission](./form-submission.md) — Complete guide for building and handling forms
- [Form Validator](./defining-form-validator.md) — Defining validation rules with `FormValidator`
