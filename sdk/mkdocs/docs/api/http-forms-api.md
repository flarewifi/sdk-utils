
# IHttpFormsApi

The `IHttpFormsApi` provides form validation functionality for custom HTML forms. It validates form input values based on defined rules and manages validation errors.

See [Form Submission](../guides/form-submission.md) documentation for usage example.

## IHttpFormsApi

### Definition

```go
type IHttpFormsApi interface {
    // Parses and validates form data from the http request based on the provided validators.
    ParseForm(w http.ResponseWriter, r *http.Request, validator FormValidator) (IFormValues, error)

    // Retrieves form validation errors
    Errors(w http.ResponseWriter, r *http.Request, validatorName string) FormErrors
}
```

### Methods

#### ParseForm

Parses and validates the input values from the HTTP request based on the provided validation rules in the [Form Validator](#form-validator).

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    formValidator := sdkapi.FormValidator{
        Name: "my-form",
        Validators: []sdkapi.FormFieldValidator{
            {
                FieldName:  "username",
                FieldLabel: "Username",
                FieldType:  sdkapi.FormFieldTypeString,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  "5",
                },
            },
        },
    }

    formValues, err := formsAPI.ParseForm(w, r, formValidator)
    if err != nil {
        // handle error - validation failed
        // redirect back to form with errors
        return
    }

    // validation passed - parse request values
    username, _ := formValues.GetStringValue("username")
}
```

#### Errors

Retrieves the validation errors from `ParseForm` and returns them as a FormErrors interface.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()

    validatorName := "my-form"
    formErrors := formsAPI.Errors(w, r, validatorName)

    // Check for errors
    if formErrors.HasError("username") {
        errorMsg := formErrors.GetError("username")
        // handle error
    }
}
```

See [Form Submission](../guides/form-submission.md) for a complete example of handling form data with validation.

---

## IFormValues

The `IFormValues` interface provides methods to retrieve parsed form values.

### Definition

```go
type IFormValues interface {
    // Returns the input field value as a string
    GetStringValue(name string) (string, error)

    // Returns the input field value as a int64
    GetIntValue(name string) (int64, error)

    // Returns the input field value as a float64
    GetFloatValue(name string) (float64, error)

    // Returns the input field value as a bool
    GetBoolValue(name string) (bool, error)

    // Returns the temp filepath of the uploaded file
    GetFilePath(name string) (string, error)
}
```

### Usage Example

```go
formValues, err := formsAPI.ParseForm(w, r, formValidator)
if err != nil {
    // handle error
}

username, _ := formValues.GetStringValue("username")
age, _ := formValues.GetIntValue("age")
price, _ := formValues.GetFloatValue("price")
accepted, _ := formValues.GetBoolValue("accept_terms")
filePath, _ := formValues.GetFilePath("document")
```

---

## FormErrors

The `FormErrors` interface provides methods to check and retrieve validation errors.

### Definition

```go
type FormErrors interface {
    // Returns true if there is an error for the given field name
    HasError(name string) bool

    // Returns the error message for the given field name
    GetError(name string) string
}
```

### Usage Example

```go
formErrors := formsAPI.Errors(w, r, "my-form")

if formErrors.HasError("username") {
    errorMsg := formErrors.GetError("username")
    // display error
}
```

---

## FormValidator {#form-validator}

The `FormValidator` struct defines the validation rules for a form.

### Definition

```go
type FormValidator struct {
    Name       string
    Validators []FormFieldValidator
}
```

### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the form validator. |
| `Validators` | A collection of [FormFieldValidator](#formfieldvalidator) defining validation rules for each field. |

### Usage Example

```go
formValidator := sdkapi.FormValidator{
    Name: "registration-form",
    Validators: []sdkapi.FormFieldValidator{
        {
            FieldName:  "email",
            FieldLabel: "Email Address",
            FieldType:  sdkapi.FormFieldTypeString,
            FieldRules: sdkapi.FormFieldRules{
                Required: true,
                Email:    true,
                Minimum:  "5",
                Maximum:  "100",
            },
        },
        {
            FieldName:  "age",
            FieldLabel: "Age",
            FieldType:  sdkapi.FormFieldTypeInteger,
            FieldRules: sdkapi.FormFieldRules{
                Required: true,
                Number:   true,
                Minimum:  "18",
                Maximum:  "120",
            },
        },
    },
}
```

---

## FormFieldValidator {#formfieldvalidator}

The `FormFieldValidator` struct defines validation rules for a single form field.

### Definition

```go
type FormFieldValidator struct {
    FieldName  string
    FieldLabel string
    FieldType  FormFieldType
    FieldRules FormFieldRules
}
```

### Properties

| Property | Description |
| ---- | ----|
| `FieldName` | The unique name of the form field. This must match the `name` attribute in your HTML input. |
| `FieldLabel` | The label for the field. Used in validation error messages. |
| `FieldType` | The type of the field. See [Field Types](#field-types) for available types. |
| `FieldRules` | The validation rules for the field. See [FormFieldRule](#formfieldrule). |

---

## FormFieldRules {#formfieldrules}

The `FormFieldRules` struct defines validation constraints for a form field.

```go
type FormFieldRules struct {
    Required bool   // value must be provided
    Email    bool   // value must be a valid email
    Number   bool   // value must be a number (int or float)
    Minimum  string // parsable minimum value (for number/float) or length (for string)
    Maximum  string // parsable maximum value (for number/float) or length (for string)
    FileExt  string // allowed file extensions separated by comma (if file input)
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Required` | `bool` | Indicates if the field is required. |
| `Email` | `bool` | Validates that the field contains a valid email address. |
| `Number` | `bool` | Validates that the field contains a numeric value. |
| `Minimum` | `string` | Minimum allowed value (for numbers) or minimum number of characters (for strings). |
| `Maximum` | `string` | Maximum allowed value (for numbers) or maximum number of characters (for strings). |
| `FileExt` | `string` | Allowed file extensions separated by comma (for file inputs). |

---

## Field Types {#field-types}

The following field types are supported for form validation:

| Field Type | Data Type | Description |
| ---- | ---- | ---- |
| `sdkapi.FormFieldTypeString` | `string` | Text input fields (including textarea) |
| `sdkapi.FormFieldTypeInteger` | `int64` | Integer number input fields |
| `sdkapi.FormFieldTypeDecimal` | `float64` | Decimal number input fields |
| `sdkapi.FormFieldTypeBoolean` | `bool` | Boolean/checkbox input fields |
| `sdkapi.FormFieldTypeFile` | `string` | File upload input fields |

### Usage Example

```go
// String field validation
{
    FieldName:  "description",
    FieldLabel: "Description",
    FieldType:  sdkapi.FormFieldTypeString,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Minimum:  "10",
        Maximum:  "500",
    },
}

// Integer field validation
{
    FieldName:  "quantity",
    FieldLabel: "Quantity",
    FieldType:  sdkapi.FormFieldTypeInteger,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Number:   true,
        Minimum:  "1",
        Maximum:  "1000",
    },
}

// Decimal field validation
{
    FieldName:  "price",
    FieldLabel: "Price",
    FieldType:  sdkapi.FormFieldTypeDecimal,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Number:   true,
        Minimum:  "0.01",
        Maximum:  "9999.99",
    },
}

// Boolean field validation
{
    FieldName:  "accept_terms",
    FieldLabel: "Accept Terms",
    FieldType:  sdkapi.FormFieldTypeBoolean,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
    },
}

// File field validation
{
    FieldName:  "document",
    FieldLabel: "Document",
    FieldType:  sdkapi.FormFieldTypeFile,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        FileExt:  "pdf,doc,docx",
    },
}
```

---

## Complete Example

Here's a complete example of using the form validator with a custom HTML form:

### Define the Form Validator

```go
func validateRegistrationForm(w http.ResponseWriter, r *http.Request) (sdkapi.IFormValues, error) {
    formsAPI := api.Http().Forms()

    formValidator := sdkapi.FormValidator{
        Name: "registration-form",
        Validators: []sdkapi.FormFieldValidator{
            {
                FieldName:  "username",
                FieldLabel: "Username",
                FieldType:  sdkapi.FormFieldTypeString,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  "3",
                    Maximum:  "50",
                },
            },
            {
                FieldName:  "email",
                FieldLabel: "Email",
                FieldType:  sdkapi.FormFieldTypeString,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Email:    true,
        Minimum:  "5",
        Maximum:  "100",
    },
            },
            {
                FieldName:  "age",
                FieldLabel: "Age",
                FieldType:  sdkapi.FormFieldTypeInteger,
    FieldRules: sdkapi.FormFieldRules{
        Required: true,
        Number:   true,
        Minimum:  "1",
        Maximum:  "1000",
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

    return formsAPI.ParseForm(w, r, formValidator)
}
```

### Display Validation Errors

```go
func showForm(w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()

    // Get validation errors (if any)
    formErrors := formsAPI.Errors(w, r, "registration-form")

    // Render your custom form template with errors
    formView := views.RegistrationForm(api, formErrors)
    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: formView,
    })
}
```

See [Form Submission](../guides/form-submission.md) and [Form Validator](../guides/defining-form-validator.md) guides for more details.
