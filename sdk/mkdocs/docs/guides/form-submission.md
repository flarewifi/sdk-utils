# Form Submission

Form submission is a way of sending data from the browser to the server using [HTML forms](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/form).
To build HTML forms, we can use basic HTML elements like `input`, `textarea`, `select`, and `button`.

## Building a form {#build-form}

To build a form, first we need to define the form's section and fields using [HTML forms](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/form).

```templ

const (
    SampleForm = "my-sample-form"
    StringField = "my-string-field"
    IntegerField = "my-integer-field"
)

templ SampleView(api sdkapi.IPluginApi, formErrors sdkapi.FormErrors) {
    <div class="container">
        <div class="row mb-2">
            <div class="col">
                Sample Form
            </div>
        </div>
        <div class="row">
            <div class="col-md-10">
                <div class="row">
                    <div class="col-7 border rounded ms-2 me-2 mb-4">
                        <form id={ SampleForm } 
                            method="POST" 
                            action={ templ.SafeURL(api.Http().Helpers().UrlForRoute("admin.sample.save")) }
                        >
                             {{
                                 stringFldClass := "form-control"
                                 intFldClass := "form-control"

                                 if formErrors.HasError(StringField) {
                                     stringFldClass += " is-invalid"
                                 }

                                 if formErrors.HasError(IntegerField) {
                                     intFldClass += " is-invalid"
                                 }
                             }}

                            <div class="pb-3">
                                <label for={ StringField } class={ "form-label" }>Sample String Field:</label>
                                <input 
                                    type="text" 
                                    class={ stringFldClass } 
                                    id={ StringField } 
                                    name={ StringField } 
                                />
                                 if formErrors.HasError(StringField) {
                                     <div class="invalid-feedback">{ formErrors.GetError(StringField) }</div>
                                 }
                            </div>

                            <div class="pb-3">
                                <label for={ IntegerField } class={ "form-label" }>Sample Integer Field:</label>
                                <input 
                                    type="number" 
                                    class={ intFldClass } 
                                    id={ IntegerField } 
                                    name={ IntegerField } 
                                />
                                 if formErrors.HasError(IntegerField) {
                                     <div class="invalid-feedback">{ formErrors.GetError(IntegerField) }</div>
                                 }
                            </div>

                            <div class="mb-3 align-items-center">
                                <button class="btn btn-primary" type="submit">
                                    { api.Translate("label", "save") }
                                </button>
                            </div>
                         </form>
                     </div>
                 </div>
             </div>
         </div>
     </div>
 }

```

## Rendering the form

To render the form in our views, we need to pass the validation errors from the form API's [Errors](../api/http-forms-api.md#errors) method to our template:

```go
// handler
func handler(w http.ResponseWriter, r *http.Request) {
    res := api.Http().Response()

    // Form errors from form's API to use for validation.
    formErrors := api.Http().Forms().Errors(w, r, views.SampleForm)

    // Retrieve our custom form template.
    sampleViewForm := views.SampleView(api, formErrors)

    // Render form to the admin view
    res.AdminView(w, r, sdkapi.ViewPage{PageContent: sampleViewForm})
}
```
In the example above, the form errors are used to handle form validations. It’s important that the form and field names match, so that each validation error is displayed with the correct input field.

## The form action route

When we defined the [HTML form](#build-form) in our example above, we have set the form action to `admin.sample.save`.
This means that when a user clicks the `Submit` button on the HTML form, it will submit the form data to the `admin.sample.save` route using `POST` HTTP method. Thus, one must register the route to a [Router Instance](../api/http-router-api.md).

```go
    pluginRouter := api.Http().Router().PluginRouter()

    pluginRouter.Post("/save", func (w http.ResponseWriter, r *http.Request) {

        // Handle the form data...

    }).Name("admin.sample.save") // Set the route name to "admin.sample.save"
```

## Handling the form data

To get the form data, first we need to parse our form with our custom-defined form validators using [ParseForm](../api/http-forms-api.md#parseform) method.

Then we can use the [IFormValues](../api/http-forms-api.md#iformvalues) methods to retrieve the values from the form input fields.


```go
    pluginRouter := api.Http().Router().PluginRouter()
    cfgAPI := api.Config().Plugin()

    pluginRouter.Post("/save", func (w http.ResponseWriter, r *http.Request) {
        res := api.Http().Response()

        // Define your custom form rules.
        formValidators := []sdkapi.FormFieldValidator{
            {
                FieldName:  views.StringField, // Input field name. Important that it matches with the field name.
                FieldLabel: "Sample String Field",
                FieldType:  sdkapi.FormFieldTypeString,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  "5",
                    Maximum:  "100",
                },
            },
            {
                FieldName:  views.IntegerField,
                FieldLabel: "Sample Integer Field",
                FieldType:  sdkapi.FormFieldTypeInteger,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Number:   true,
                    Minimum:  "1",
                    Maximum:  "100",
                },
            },
        }

        formValidator := sdkapi.FormValidator{
            Name:       views.SampleForm,
            Validators: formValidators,
        }

        // Parse form with the custom form validator.
        formValues, err := api.Http().Forms().ParseForm(w, r, formValidator)
        if err != nil {
            res.FlashMsg(w, r, "parsing error", sdkapi.FlashMsgError)

            // Redirect back to your form view if there's parsing error.
            res.Redirect(w, r, "admin.sample")

            return
        }

        // Read the form values.
        stringValue, _ := formValues.GetStringValue(views.StringField)
        intValue, _ := formValues.GetIntValue(views.IntegerField)

        // Do something with the parsed values.

    }).Name("admin.sample.save") // Set the route name to "admin.sample.save"
```
For additional information about the form rules and validators, refer to [Form Validator](../guides/defining-form-validator.md#form-validator).

!!!note
    Read the [Error Handling](./error-handling.md) and the [Saving Data](./saving-data.md) guides.
