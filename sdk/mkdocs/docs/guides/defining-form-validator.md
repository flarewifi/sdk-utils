# Form Validator

The `FormValidator` enables developers to define custom validation rules for their forms in the backend.

See [Form Submission](./form-submission.md) for a complete usage example.

## Types

### FormValidator

The `FormValidator` struct defines the validation rules for a form. Pass it to [`IHttpFormsApi.ParseForm`](../api/http-forms-api.md#parseform) to validate a form submission.

```go
type FormValidator struct {
    Name       string
    Validators []FormFieldValidator
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the form. This must match the form name used in [Errors](../api/http-forms-api.md#errors). |
| `Validators` | A slice of [FormFieldValidator](#formfieldvalidator) defining validation rules for each field. |

---

### FormFieldValidator

The `FormFieldValidator` struct defines validation rules for a single form field.

```go
type FormFieldValidator struct {
    FieldName  string
    FieldLabel string
    FieldType  FormFieldType
    FieldRules FormFieldRules
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `FieldName` | The unique name of the form field. This must match the `name` attribute in your HTML input. |
| `FieldLabel` | The label for the field. Used in validation error messages. |
| `FieldType` | The type of the field. See [Field Types](../api/http-forms-api.md#field-types) for available values. |
| `FieldRules` | The validation rules for the field. See [FormFieldRules](#formfieldrules). |

---

### FormFieldRules

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

#### Properties

| Property | Description |
| ---- | ----|
| `Required` | Indicates if the field is required. |
| `Email` | Indicates if the field must be a valid email address. |
| `Number` | Indicates if the field must be numeric. |
| `Minimum` | Minimum value for numbers, or minimum character length for strings. |
| `Maximum` | Maximum value for numbers, or maximum character length for strings. |
| `FileExt` | Allowed file extensions for file inputs, comma-separated (e.g. `"jpg,png,gif"`). |

---

## Usage Example

Here is a complete example of defining and using form validators:

```go
func saveUserSettings(w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()

    // Define validation rules
    formValidator := sdkapi.FormValidator{
        Name: "user-settings",
        Validators: []sdkapi.FormFieldValidator{
            {
                FieldName:  "username",
                FieldLabel: "Username",
                FieldType:  sdkapi.FormFieldTypeString,
                FieldRules: sdkapi.FormFieldRules{
                    Required: true,
                    Minimum:  "4",
                    Maximum:  "20",
                },
            },
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
                    Minimum:  "18",
                    Maximum:  "120",
                },
            },
        },
    }

    // Validate the form
    formValues, err := formsAPI.ParseForm(w, r, formValidator)
    if err != nil {
        // Validation failed - redirect back to form
        api.Http().Response().Redirect(w, r, "user.settings")
        return
    }

    // Validation passed - get form values
    username, _ := formValues.GetStringValue("username")
    email, _ := formValues.GetStringValue("email")
    age, _ := formValues.GetIntValue("age")

    _ = username
    _ = email
    _ = age

    // Process the validated data...
}
```

For more information, see:
- [IHttpFormsApi](../api/http-forms-api.md) - Complete API reference
- [Form Submission](./form-submission.md) - Complete guide with HTML form examples
- [Field Types](../api/http-forms-api.md#field-types) - Available field types for validation
