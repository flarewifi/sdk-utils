# Form Validator

The `FormValidator` enables developers to define custom validation rules for their forms in the backend.

See [Form Submission](./form-submission.md) for a complete usage example.

## Types

### FormWithValidator

The `FormWithValidator` struct defines the validation rules for a form.

```go
type FormWithValidator struct {
    FormName       string  
    FormValidators []FormValidator 
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `FormName` | The unique name of the form. This must match the form name used in [Errors](../api/http-forms-api.md#errors). |
| `FormValidators` | A collection of [FormValidator](#formvalidator) defining validation rules for each field in the form. |

---

### FormValidator

The `FormValidator` struct defines validation rules for a single form field.

```go
type FormValidator struct {
    FieldName  string  
    FieldLabel string  
    FieldType  string  
    FieldRules FormFieldRules
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `FieldName` | The unique name of the form field. This must match the `name` attribute in your HTML input. |
| `FieldLabel` | The label for the field. Used in validation error messages. |
| `FieldType` | The type of the field. Refer to [Field Types](../api/http-forms-api.md#field-types) for the available types. |
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
| `Minimum` | Minimum allowed value or minimum number of characters for string values. |
| `Maximum` | Maximum allowed value or maximum number of characters for string values. |

---

## Usage Example

Here's a complete example of defining and using form validators:

```go
func saveUserSettings(w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    
    // Define validation rules
    formValidator := sdkapi.FormWithValidator{
        FormName: "user-settings",
        FormValidators: []sdkapi.FormValidator{
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
    err := formsAPI.ParseFormWithValidator(w, r, formValidator)
    if err != nil {
        // Validation failed - redirect back to form
        api.Http().Response().Redirect(w, r, "user.settings")
        return
    }
    
    // Validation passed - get form values
    username := r.FormValue("username")
    email := r.FormValue("email")
    age := r.FormValue("age")
    
    // Process the validated data...
}
```

For more information, see:
- [IHttpFormsApi](../api/http-forms-api.md) - Complete API reference
- [Form Submission](./form-submission.md) - Complete guide with HTML form examples
- [Field Types](../api/http-forms-api.md#field-types) - Available field types for validation



