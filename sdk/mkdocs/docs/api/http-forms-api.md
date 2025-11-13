
# IHttpFormsApi

The `IHttpFormsApi` provides form validation functionality for custom HTML forms. It validates form input values based on defined rules and manages validation errors.

See [Form Submission](../guides/form-submission.md) documentation for usage example.

## IHttpFormsApi

### Definition

```go
type IHttpFormsApi interface {
    ParseFormWithValidator(w http.ResponseWriter, r *http.Request, form FormWithValidator) error
	Errors(w http.ResponseWriter, r *http.Request, formName string) map[string]string
}
```

### Methods

#### ParseFormWithValidator

Parses and validates the input values from the HTTP request based on the provided validation rules in the [Form Validator](#form-validator).

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    formValidator := sdkapi.FormWithValidator{
        FormName: "my-form",
        FormValidators: []sdkapi.FormValidator{
            {
                FieldName:  "username",
                FieldLabel: "Username",
                FieldType:  sdkapi.FormFieldTypeText,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  4,
                    Maximum:  20,
                },
            },
        },
    }
        
    err := formsAPI.ParseFormWithValidator(w, r, formValidator)
    if err != nil {
        // handle error - validation failed
        // redirect back to form with errors
        return
    }

    // validation passed - parse request values
    username := r.FormValue("username")
}
```

#### Errors

Retrieves the validation errors from `ParseFormWithValidator` and returns them as a map, with the field name as the keys.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    
    formName := "my-form"
    errMap := formsAPI.Errors(w, r, formName)
    
    // Pass the error map to your custom views to display validation errors
    // Example: views.MyForm(api, errMap)
}
```

See [Form Submission](../guides/form-submission.md) for a complete example of handling form data with validation.

---

## FormWithValidator {#form-validator}

The `FormWithValidator` struct defines the validation rules for a form.

### Definition

```go
type FormWithValidator struct {
    FormName       string
    FormValidators []FormValidator
}
```

### Properties

| Property | Description |
| ---- | ----|
| `FormName` | The unique name of the form. |
| `FormValidators` | A collection of [FormValidator](#formvalidator) defining validation rules for each field. |

### Usage Example

```go
formValidator := sdkapi.FormWithValidator{
    FormName: "registration-form",
    FormValidators: []sdkapi.FormValidator{
        {
            FieldName:  "email",
            FieldLabel: "Email Address",
            FieldType:  sdkapi.FormFieldTypeText,
            FieldRules: sdkapi.FormFieldRules{
                Required: true,
                Minimum:  5,
                Maximum:  100,
            },
        },
        {
            FieldName:  "age",
            FieldLabel: "Age",
            FieldType:  sdkapi.FormFieldTypeInteger,
            FieldRules: sdkapi.FormFieldRules{
                Required: true,
                Minimum:  18,
                Maximum:  120,
            },
        },
    },
}
```

---

## FormValidator {#formvalidator}

The `FormValidator` struct defines validation rules for a single form field.

### Definition

```go
type FormValidator struct {
    FieldName  string
    FieldLabel string
    FieldType  string
    FieldRules FormFieldRules
}
```

### Properties

| Property | Description |
| ---- | ----|
| `FieldName` | The unique name of the form field. This must match the `name` attribute in your HTML input. |
| `FieldLabel` | The label for the field. Used in validation error messages. |
| `FieldType` | The type of the field. See [Field Types](#field-types) for available types. |
| `FieldRules` | The validation rules for the field. See [FormFieldRules](#formfieldrules). |

---

## FormFieldRules {#formfieldrules}

The `FormFieldRules` struct defines validation constraints for a form field.

### Definition

```go
type FormFieldRules struct {
    Required bool
    Minimum  int
    Maximum  int
}
```

### Properties

| Property | Description |
| ---- | ----|
| `Required` | Indicates if the field is required. |
| `Minimum` | Minimum allowed value or minimum number of characters for string values. |
| `Maximum` | Maximum allowed value or maximum number of characters for string values. |

### Validation Behavior by Field Type

| Field Type | Minimum | Maximum |
| ---- | ---- | ---- |
| `FormFieldTypeText` | Minimum string length | Maximum string length |
| `FormFieldTypeInteger` | Minimum numeric value | Maximum numeric value |
| `FormFieldTypeDecimal` | Minimum numeric value | Maximum numeric value |
| `FormFieldTypeBoolean` | Not applicable | Not applicable |

---

## Field Types {#field-types}

The following field types are supported for form validation:

| Field Type | Data Type | Description |
| ---- | ---- | ---- |
| `sdkapi.FormFieldTypeText` | `string` | Text input fields (including textarea) |
| `sdkapi.FormFieldTypeInteger` | `int64` | Integer number input fields |
| `sdkapi.FormFieldTypeDecimal` | `float64` | Decimal number input fields |
| `sdkapi.FormFieldTypeBoolean` | `bool` | Boolean/checkbox input fields |

### Usage Example

```go
// Text field validation
{
    FieldName:  "description",
    FieldLabel: "Description",
    FieldType:  sdkapi.FormFieldTypeText,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Minimum:  10,   // minimum 10 characters
        Maximum:  500,  // maximum 500 characters
    },
}

// Integer field validation
{
    FieldName:  "quantity",
    FieldLabel: "Quantity",
    FieldType:  sdkapi.FormFieldTypeInteger,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Minimum:  1,    // minimum value of 1
        Maximum:  100,  // maximum value of 100
    },
}

// Decimal field validation
{
    FieldName:  "price",
    FieldLabel: "Price",
    FieldType:  sdkapi.FormFieldTypeDecimal,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Minimum:  0,      // minimum value of 0
        Maximum:  10000,  // maximum value of 10000
    },
}

// Boolean field validation
{
    FieldName:  "accept_terms",
    FieldLabel: "Accept Terms",
    FieldType:  sdkapi.FormFieldTypeBoolean,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,  // user must check the checkbox
    },
}
```

---

## Complete Example

Here's a complete example of using the form validator with a custom HTML form:

### Define the Form Validator

```go
func validateRegistrationForm(w http.ResponseWriter, r *http.Request) error {
    formsAPI := api.Http().Forms()
    
    formValidator := sdkapi.FormWithValidator{
        FormName: "registration-form",
        FormValidators: []sdkapi.FormValidator{
            {
                FieldName:  "username",
                FieldLabel: "Username",
                FieldType:  sdkapi.FormFieldTypeText,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  4,
                    Maximum:  20,
                },
            },
            {
                FieldName:  "email",
                FieldLabel: "Email",
                FieldType:  sdkapi.FormFieldTypeText,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  5,
                    Maximum:  100,
                },
            },
            {
                FieldName:  "age",
                FieldLabel: "Age",
                FieldType:  sdkapi.FormFieldTypeInteger,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  18,
                    Maximum:  120,
                },
            },
            {
                FieldName:  "accept_terms",
                FieldLabel: "Terms and Conditions",
                FieldType:  sdkapi.FormFieldTypeBoolean,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                },
            },
        },
    }
    
    return formsAPI.ParseFormWithValidator(w, r, formValidator)
}
```

### Display Validation Errors

```go
func showForm(w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    
    // Get validation errors (if any)
    errMap := formsAPI.Errors(w, r, "registration-form")
    
    // Render your custom form template with errors
    formView := views.RegistrationForm(api, errMap)
    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: formView,
    })
}
```

See [Form Submission](../guides/form-submission.md) and [Form Validator](../guides/defining-form-validator.md) guides for more details.
