# Form Submission

Form submission is a way of sending data from the browser to the server using [HTML forms](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/form).
To build HTML forms, we can use basic HTML elements like `input`, `textarea`, `select`, and `button`.

To have a consistent look and feel throughout the application, we must use the [IHttpFormApi](../api/http-forms-api.md) to build forms.
This API provides a convenient way of building HTML forms with built-in input validation and error handling.

## Building a form {#build-form}

To build a form, first we need to define the form's [sections and fields](../api/http-forms-api.md#httpform), then register the form using the [IHttpFormsApi.RegisterForms](../api/http-forms-api.md#registerforms) method:

```go
formsAPI := api.Http().Forms()

// Define the form sections
sections := []sdkapi.FormSection{
    {
        {
            Name: "general_configuration",
            Label: "General Configuration",
            Fields: []sdkapi.IFormField{
                // Boolean Field
                sdkapi.FormBooleanField{
                    Name:  "accept_terms",
                    Label: "Accept Terms and Conditions",
                    ValueFn: func() bool {
                        // your custom specific boolean logic
                        return true
                    },
                },
                // Decimal Field
                priceField := FormDecimalField{
                    Name:      "price",
                    Label:     "Product Price",
                    Step:      0.01,
                    Precision: 2,
                    ValueFn: func() float64 {
                        // your custom specific decimal form logic
                        return 99.99
                    },
                }
                // Integer Field,
                ageField := sdkapi.FormIntegerField{
                    Name:  "age",
                    Label: "User Age",
                    ValueFn: func() int64 {
                        // your custom integer specific logic
                        return 25
                    },
                }
                // List Field,
                listField := sdkapi.FormListField{
                    Name:  "experience_level",
                    Label: "Select Experience Level",
                    Type:  "int", // Specifies that the values are integers
                    OptionsFn: func() []sdkapi.FormListFieldOption {
                        return []sdkapi.FormListFieldOption{
                            {Label: "Beginner", Value: 1},
                            {Label: "Intermediate", Value: 2},
                            {Label: "Advanced", Value: 3},
                        }
                    },
                    ValueFn: func() interface{} {
                        // Your custom list specific logic
                        return 2 // Default selected value (Intermediate)
                    },
                }
                // Multi Field,
                sdkapi.FormMultiField{
                    Name:  "items",
                    Label: "Order Items",
                    Columns: func() []sdkapi.FormMultiFieldCol {
                        return []sdkapi.FormMultiFieldCol{
                            {Name: "item_name", Label: "Item Name", Type: "string"},
                            {Name: "quantity", Label: "Quantity", Type: "int"},
                            {Name: "price", Label: "Price", Type: "float"},
                        }
                    },
                    ValueFn: func() [][]sdkapi.FormFieldData {
                        // Your custom multi field logic
                        return [][]sdkapi.FormFieldData{
                            {{"item_name", "Apple"}, {"quantity", 3}, {"price", 1.99}},
                            {{"item_name", "Banana"}, {"quantity", 2}, {"price", 0.99}},
                        }
                    },
                },
                // Text Field
                sdkapi.FormTextField{
                    Name:  "banner_text",
                    Label: "Banner Text",
                    ValueFn: func() string {
                        b, err := pluginConfigAPI.Read("banner_text")
                        if err != nil {
                            return "This is the default banner text!"
                        }
                        return string(b)
                    },
                },
                // String Field
                sdkapi.FormStringField{
                    Name:  "Username",
                    Label: "Username",
                    IsReadOnly: true,
                    Extractable: true,
                    IsPassword: false,
                    ValueFn: func() string {
                        username := ""
                        // Your custom specific string logic
                        return username
                    },
                },
            },
        },
    },
}


// Register the form
if err := formsAPI.RegisterForm("my-form", func (r *http.Request) sdkapi.HttpForm {
    // Define the form
    return sdkapi.HttpForm{
        CallbackRoute: "settings:save", // route to handle form submission
        SubmitLabel: "Submit", // submit button text
        Sections: sections, // add sections
    }
}); err != nil {
    // handle error
}
```

See [IHttpFormsApi](../api/http-forms-api.md) documentation to know more.

## Rendering the form

To render the form in our views, first we need to get the form's template using the [IHttpForm.GetTemplate](../api/http-forms-api.md#gettemplate) method then use one of the [IHttpResponse](../api/http-response.md) methods to render the form template:

```go
// handler
func handler(w http.ResponseWriter, r *http.Request) {
    // Get the form template
    formTpl, err := api.Http().Forms().GetFormTemplate("my-form", r)
    if err != nil {
        // handle error
    }

    // render form to the admin view
    api.Http().HttpResponse().AdminView(w, r, sdkapi.ViewPage{
        PageContent: formTpl,
    })
}
```

## The callback route

When we defined the [HTML form](#build-form) in our example above, we have set the [CallbackRoute](../api/http-forms-api.md#callbackroute) to `settings:save`.
This means that when a user clicks the `Submit` button on the HTML form, it will submit the form data to the `settings:save` route using `POST` HTTP method. Thus, one must register the callback route to a [Router Instance](../api/http-router-api.md).

```go
pluginRouter := api.Http().HttpRouter().PluginRouter()

pluginRouter.Post("/settings/save", func (w http.ResponseWriter, r *http.Request) {

    // Handle the form data...

}).Name("settings:save") // Set the route name to "settings:save"
```

## Handling the form data

To get the form data, first we need to get the registered from using the [IHttpFormsApi.GetForm](../api/http-forms-api.md#getform) method.
Then we can use the [IHttpForm](../api/http-forms-api.md#ihttpform) methods to retrieve the values from the form input fields.
For example, if we want to retrieve the input value in the form section named `general_configuration`  and the input field named `banner_text`:

```go
pluginRouter := api.Http().HttpRouter().PluginRouter()
cfgAPI := api.Config().Plugin()

pluginRouter.Post("/settings/save", func (w http.ResponseWriter, r *http.Request) {
    // Handle the form data...

    // Parse and validate the form input values
    form, ok := api.Http().Forms().ParseForm("my-form", r)
    if !ok {
        // handle error
    }

    // If form is valid
    // Get the "banner_text" input value
    val, err := form.GetStringValue("general_configuration", "banner_text")
    if err != nil {
        // handle error
    }

    // Save the data using the plugin config API
    if err := cfgAPI.Write("banner_text", []byte(val)); err != nil {
        // handle error
    }

}).Name("settings:save") // Set the route name to "settings:save"
```

!!!note
    Read the [Error Handling](./error-handling.md) and the [Saving Data](./saving-data.md) guides.
