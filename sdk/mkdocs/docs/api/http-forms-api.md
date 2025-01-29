
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

The `IFormField` interface defines a generic form field structure. It provides methods for retrieving essential field properties such as name, label, type, and value.

### Definition

```go
type IFormField interface {
    GetName() string
    GetLabel() string
    GetType() string
    GetValue() interface{}
}
```

### Available Fields
Below are the available fields that can be used in the `HttpForm` definition which implements the `IFormField` interface.

#### Boolean Field

TODO: Add description

#### Decimal Field

The `FormDecimalField` represents a decimal number input field in an HTTP form.

##### Definition

```go
type FormDecimalField struct {
    Name      string
    Label     string
    Step      float64   // controls the increment/decrement value of the field
    Precision int       // controls the precision of the decimal value or how many decimal places it accepts
    ValueFn   func() float64
}
```

##### Methods

| Method | Description |
| ---- | ---- |
| `GetName() string` | Returns the decimal field name. |
| `GetLabel() string` | Returns the decimal field label. |
| `GetType() string` | Returns the field type ("decimal"). |
| `GetValue() interface{}` | Returns the decimal value of the field. Uses `ValueFn` if set, otherwise returns 0.0. |

##### Usage Example

```go
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
```

#### Integer Field

The `FormIntegerField` represents an integer input field in an HTTP form.

##### Definition

```go
type FormIntegerField struct {
	Name    string
	Label   string
	ValueFn func() int64
}
```

##### Methods

| Method | Description |
| ---- | ---- |
| `GetName() string` | Returns the integer field name. |
| `GetLabel() string` | Returns the integer field label. |
| `GetType() string` | Returns the field type ("int"). |
| `GetValue() interface{}` | Returns the integer value of the field. Uses `ValueFn()` if set, otherwise returns 0. |

##### Usage Example

```go
ageField := FormIntegerField{
    Name:  "age",
    Label: "User Age",
    ValueFn: func() int64 {
        // your custom integer specific logic
        return 25
    },
}
```

#### List Field

#### Multi Field

The `FormMultiField` represents a structured form field that consists of multiple rows and columns. Each column defines a specific type of data, and each row contains values for those columns.

##### Interface: `IFormMultiField`

The IFormMultiField interface provides methods to retrieve values from a multi-row form field.

**`IFormMultiField` Methods**

| Method | Description |
| ---- | ---- |
| `NumRows() int` | Returns the number of rows in the multi-field form. |
| `GetStringValue(row int, name string) (string, error)` | Retrieves a string value from the specified row and column name. |
| `GetIntValue(row int, name string) (int64, error)` | Retrieves an integer value from the specified row and column name. |
| `GetFloatValue(row int, name string) (float64, error)` | Retrieves a float value from the specified row and column  name. |
| `GetBoolValue(row int, name string) (bool, error)` | Retrieves a boolean value from the specified row and column name. |

##### Struct: `FormMultiFieldCol`

Represents a column in the multi-field form.

**`FormMultiFieldCol` Fields**

| Field | Type | Description |
| ---- | ---- | ---- |
| `Name` | string | The name of the column. |
| `Label` | string | The label displayed for the column. |
| `Type` | string | The data type of the column (e.g., "string", "int", "float", "bool"). | 
| `ValueFn` | `func() interface{}` | Function that returns the value for the column. |

**`FormMultiFieldCol` Methods**

| Method | Description |
| ---- | ---- |
| `GetName()`  string` | Returns the column name. |
| `GetLabel() string` | Returns the column label. |
| `GetType() string` | Returns the column data type. |
| `GetValue() interface{}` | Returns the column value using ValueFn if set, otherwise nil. |

##### Struct: `FormMultiField`

Represents a multi-field form containing multiple rows and columns.

**`FormMultiField` Fields**

| Field | Type | Description |
| ---- | ---- | ---- |
| `Name` | string | The name of the multi-field form. |
| `Label` | string | The label displayed for the multi-field form. |
| `Columns` | `func() []FormMultiFieldCol` | Function returning a list of column definitions. |
| `ValueFn` | `func() [][]FormFieldData` | Function returning the values for each row and column. |

**`FormMultiField` Methods**

| Method | Description |
| ---- | ----- |
| `GetName() string` | Returns the field name. |
| `GetLabel() string` | Returns the field label. |
| `GetType() string` | Returns the field type ("multi"). |
| `GetValue() interface{}` | Returns the field value using ValueFn if set, otherwise returns an empty slice of  `[][]FormFieldData{}`. |

##### Usage Example

```go
field := sdkapi.FormMultiField{
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
        return [][]sdkapi.FormFieldData{
            {{"item_name", "Apple"}, {"quantity", 3}, {"price", 1.99}},
            {{"item_name", "Banana"}, {"quantity", 2}, {"price", 0.99}},
        }
    },
}

// Accessing field details
fmt.Println(field.GetName())  // "items"
fmt.Println(field.GetLabel()) // "Order Items"
fmt.Println(field.GetType())  // "multi"

// Accessing columns
columns := field.Columns()
for _, col := range columns {
    fmt.Printf("Column: %s (%s)\n", col.Label, col.Type)
}

// Accessing values
values := field.GetValue().([][]FormFieldData)
for rowIdx, row := range values {
    fmt.Printf("Row %d:\n", rowIdx+1)
    for _, fieldData := range row {
        fmt.Printf("  %s: %v\n", fieldData.Name, fieldData.Value)
    }
}
```

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
