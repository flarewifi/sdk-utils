# Form Validator
The `FormValidor` enables devs to define their own custom rules in the backend.

```go
    type FormWithValidator struct {
        FormName       string  
        FormValidators []FormValidator 
    }

    type FormValidator struct {
        FieldName  string  
        FieldLabel string  
        FieldType  string  
        FieldRules FormFieldRules
    }

    type FormFieldRules struct {
        Required bool
        Minimum  int
        Maximum  int
    }
```

## FormWithValidator Properties

| Property | Description |
| ---- | ----|
| `FormName` | The unique name of the form. |
| `FormValidators` | Set of custom rules for each fields in the form. |

## FormValidator Properties

| Property | Description |
| ---- | ----|
| `FieldName` | The unique name of the form field. |
| `FieldLabel` | Form field label. This is optional, but helpful for constructing validation errors. |
| `FieldType` | Form field type. Refer to [Field Type](../api/http-forms-api.md#field-types) for the available types. |
| `FieldRules` | Set of rules for the field |

## FormFieldRules Properties

| Property | Description |
| ---- | ----|
| `Required` | Indicator is the field is required. |
| `Minimum` | Minimum allowed value or minimum number of characters for string values. |
| `Maximum` | Maximum allowed value or maximum number of characters for string values. |



