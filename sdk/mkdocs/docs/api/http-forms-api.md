
# IHttpFormsApi

The `IHttpFormsApi` is used to build HTML forms. It is responsible for rendering, validating and parsing the HTML form and its values.


## IHttpFormsApi methods

Below are the methods available in `IHttpFormsApi`.

### RegisterForm

It registers a [HttpForm](#httpform) generator function into the plugin.

```go
formsAPI := api.Http().Forms()
if err := formsAPI.RegisterForm("my-form", func (r *http.Request) sdkapi.HttpForm {

    return sdkapi.HttpForm{
        // Define the form sections and fields
    }

}); err != nil {
    // handle error
}
```

### GetFormTemplate

It returns [templ.Component](../guides/rendering-views.md) of the HTML form.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    formComponent, err := formsAPI.GetFormTemplate("my-form", r)
    if err != nil {
        // handle error
    }

    // do something with formComponent
}
```

### ParseForm

It parses the input values from the HTTP request and returns a [IHttpForm](#ihttpform) object.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    form, err := formsAPI.ParseForm("my-form", r)
    if err != nil {
        // handle error
    }

    // do something with form
}
```

---

## HttpForm {#httpform}

The `HttpForm` struct defines the HTML form sections, fields, default input values, and validation rules.
It is composed of one or more sections. Each section contains various types of fields which include [text](#text-field), [decimal](#decimal-field), [integer](#integer-field), [boolean](#boolean-field), [list](#list-field) and [multi-field](#multi-field).

Below is an example of `HttpForm` definition:

```go
pluginConfigAPI := api.Config().Plugin()
formsAPI := api.Http().Forms()

formsAPI.RegisterForm("my-form", func (r *http.Request) sdkapi.HttpForm {

    // Define the form sections and fields
    sections := []sdkapi.FormSection{
        {
            {
                Name: "general_configuration",
                Label: "General Configuration",
                Fields: []sdkapi.IFormField{
                    // sdkapi.FormBooleanField,
                    // sdkapi.FormDecimalField,
                    // sdkapi.FormIntegerField,
                    // sdkapi.FormListField,
                    // sdkapi.FormMultiField,
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
                },
            },
        },
    }

    // Define the form callback and submit button
    form := sdkapi.HttpForm{
        CallbackRoute: "settings:save", // route to handle form submission
        SubmitLabel: "Submit", // submit button text
        Sections: sections, // assign the sections to the form
    }

    return form
})
```

Below are the attributes of the `HttpForm` struct:

### CallbackRoute

The [route](./http-router-api.md) to handle form submission.

### SubmitLabel

The text for the submit button.

### Sections

A slice of [FormSection](#formsection).

## FormSection {#formsection}

A `FormSection` is a collection of [fields](#form-fields) in a form.
It also has a `Name` and `Label` attributes.

```go
type FormSection struct {
	Name   string
	Label  string
	Fields []IFormField
}
```

## Form Fields {#form-fields}

Below are the available fields that can be used in the `HttpForm` definition.

#### Boolean Field

TODO: Add description

#### Decimal Field

TODO: Add description

#### Integer Field

#### List Field

#### Multi Field

#### Text Field

The `FormTextField` represents a text input field in a form. It provides methods to retrieve metadata and the field value dynamically.

##### Definition

```go
type FormTextField struct {
	Name    string
	Label   string
	ValueFn func() string
}
```

##### Methods

| Method | Description |
| ---- | ---- |
| `GetName() string` | Returns the unique name of the text field. |
| `GetLabel() string` | Returns the display label of the text field. |
| `GetType() string` | Returns the type of the field, which is "text". | 
| `GetValue() interface{}` | Returns the value of the text field. If ValueFn is defined, it calls the function;  otherwise, it returns an empty string. |

##### Example Usage

```go
// Create a FormTextField instance with a dynamic value function
	textField := sdkapi.FormTextField{
		Name:  "username",
		Label: "Username",
		ValueFn: func() string {
			return "john_doe"
		},
	}
```

---

## IHttpForm methods {#ihttpform}

### GetSections

Returns a slice of [FormSection](#formsection) in the form.

```go
formsAPI := api.Http().Forms()
form, _ := formsAPI.GetForm("my-form")
sections := form.GetSections()
```

### GetStringValue

Returns the string value of a field in the form.

```go
val, err := form.GetStringValue(section, "banner_text")
```

### GetStringValues

Returns a slice of strings for list or multi fields.

```go
vals, err := form.GetStringValues(section, "List Field")
```

### GetIntValue

Returns the integer value of a field in the form.

```go
val, err := form.GetIntValue(section, "Integer Field")
```

### GetIntValues

Returns a slice of integers for list or multi fields.

```go
vals, err := form.GetIntValues(section, "Integer List")
```

### GetFloatValue

Returns the float value of a decimal field in the form.

```go
val, err := form.GetFloatValue(section, "Decimal Field")
```

### GetFloatValues

Returns a slice of floats for list or multi fields.

```go
vals, err := form.GetFloatValues(section, "Decimal List")
```

### GetBoolValue

Returns the boolean value of a field in the form.
```go
val, err := form.GetBoolValue(section, "Boolean Field")
```

### GetBoolValues

Returns a slice of booleans for list or multi fields.

```go
vals, err := form.GetBoolValues(section, "Boolean List")
```

### GetMultiField

Returns a [IFormMultiField](#imultifield) instance of a multi field in the form.

```go
mf, err := form.GetMultiField(section, "Multi Field")
```

## IFormMultiField {#imultifield}
