
# IHttpFormsApi

The `IHttpFormsApi` is used to build HTML forms. It is responsible for rendering, validating and parsing the HTML form and its input values.

See [Form Submission](../guides/form-submission.md) documentation for usage example.

## IHttpFormsApi

### Definition

```go
type IHttpFormsApi interface {
	RegisterForm(name string, factory func(r *http.Request) HttpForm) error
	GetFormTemplate(name string, r *http.Request) (templ.Component, error)
	ParseForm(name string, r *http.Request) (IHttpForm, error)
}
```

### Methods

#### RegisterForm

Register a function that must return an [HttpForm](#httpform).

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

#### GetFormTemplate

It returns a [templ component](https://templ.guide) that contains the HTML form.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    formComponent, err := formsAPI.GetFormTemplate("my-form", r)
    if err != nil {
        // handle error
    }
    // render the formComponent (templ component)
}
```

See [Rendering Views](../guides/rendering-views.md) to learn how to render a `templ` component.

#### ParseForm

It parses the input values from the HTTP request and returns a [IHttpForm](#ihttpform) object.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    form, err := formsAPI.ParseForm("my-form", r)
    if err != nil {
        // handle error
    }
    // save the form data
}
```

See [Saving Data](../guides/saving-data.md) to lean how to save data from the form.

---

## HttpForm {#httpform}

The `HttpForm` struct defines the HTML form sections, fields, default input values, and validation rules.
It is composed of one or more sections. Each section contains various types of fields which include [string](#string-field), [text](#text-field), [decimal](#decimal-field), [integer](#integer-field), [boolean](#boolean-field), [list](#list-field) and [multi-field](#multi-field).

### Definition

```go
type HttpForm struct {
	CallbackRoute string
	SubmitLabel   string
	Sections      []FormSection
}
```

### Properties

| Property | Description |
|--- | --- |
| CallbackRoute | The [route](../guides/routes-and-navigation.md) to handle form submission. |
| SubmitLabel | The submit button text. |
| Sections | A collection of [form sections](#form-section). |

### Usage Example

```go
pluginConfigAPI := api.Config().Plugin()
formsAPI := api.Http().Forms()

formsAPI.RegisterForm("my-form", func (r *http.Request) sdkapi.HttpForm {

    // Define the form callback and submit button
    form := sdkapi.HttpForm{
        CallbackRoute: "settings:save", // route to handle form submission
        SubmitLabel: "Submit", // submit button text
        Sections: []sdkapi.FormSection{
            {
                Name: "general_configuration",
                Label: "General Configuration",
                Fields: []sdkapi.IFormField{
                    // Field properties are left out for brevity.
                    // To see the field properties, refer to the specific field type below.
                    sdkapi.FormBooleanField{},
                    sdkapi.FormDecimalField{},
                    sdkapi.FormIntegerField{},
                    sdkapi.FormListField{},
                    sdkapi.FormMultiField{},
                    sdkapi.FormStringField{},
                    sdkapi.FormTextField{},
                },
            },
        }
    }

    return form
})
```

---

## FormSection {#form-section}

A `FormSection` is a section in a form contains a collection of [form fields](#form-fields).

### Definition

```go
type FormSection struct {
	Name   string
	Label  string
	Fields []IFormField
}
```

### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the input field within the section scope. |
| `Label` | The label for the input field. |
| `Fields` | The collection of fields inside the form section. See [Form Fields](#form-fields) for the available fields. |

---

## Form Fields {#form-fields}

Below are the available fields that can be used in the `HttpForm` definition which implements the `IFormField` interface.

### Field Types

| Field Type | Data Type | Used In | Description
| ---- | ---- | ---- | ---
| `sdkapi.FormFieldTypeBoolean` | `bool` | [FormListField](#list-field), [FormMultiField](#multi-field) | Represents a boolean field.
| `sdkapi.FormFieldTypeDecimal` | `float64` | [FormListField](#list-field), [FormMultiField](#multi-field) | Represents a decimal field.
| `sdkapi.FormFieldTypeInteger` | `int64` | [FormListField](#list-field), [FormMultiField](#multi-field) | Represents an integer field.
| `sdkapi.FormFieldTypeList` | `[]any` | `N/A` | Represents a [list field](#list-field).
| `sdkapi.FormFieldTypeMulti` | `[][]any` | `N/A` | Represents a tabulated [multi-field](#multi-field).
| `sdkapi.FormFieldTypeString` | `string` | [FormListField](#list-field), [FormMultiField](#multi-field) | Represents a text or password input field.
| `sdkapi.FormFieldTypeText` | `string` | [FormListField](#list-field), [FormMultiField](#multi-field) | Represents a large text field.

### Boolean Field

The `FormBooleanField` represents a boolean field in an HTML form.

#### Definition

```go
type FormBooleanField struct {
    Name    string
    Label   string
    ValueFn func() bool
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the field within the section scope. |
| `Label` | The label for the input. |
| `ValueFn` | This function should return the current value of the input field. |

#### Usage Example

```go
termsField := FormBooleanField{
    Name:  "accept_terms",
    Label: "Accept Terms and Conditions",
    ValueFn: func() bool {
        return true
    },
}
```

### Decimal Field

The `FormDecimalField` represents a decimal number input field in an HTML form.

#### Definition

```go
type FormDecimalField struct {
    Name      string
    Label     string
    Step      float64   // controls the increment/decrement value of the input field
    Precision int       // controls the precision of the decimal value or how many decimal places it accepts
    ValueFn   func() float64
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the field within the section scope. |
| `Label` | The label for the input. |
| `Step` | The increment/decrement value of the input field. |
| `Precision` | The number of decimal fields for the input value. |
| `ValueFn` | This function should return the current value of the input field. |

#### Usage Example

```go
priceField := FormDecimalField{
    Name:      "price",
    Label:     "Product Price",
    Step:      0.01,
    Precision: 2,
    ValueFn: func() float64 {
        return 99.99
    },
}
```

### Integer Field

The `FormIntegerField` represents an integer input field in an HTML form.

#### Definition

```go
type FormIntegerField struct {
	Name    string
	Label   string
	ValueFn func() int64
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the field within the section scope. |
| `Label` | The label for the input. |
| `ValueFn` | This function should return the current value of the input field. |

#### Usage Example

```go
ageField := FormIntegerField{
    Name:  "age",
    Label: "User Age",
    ValueFn: func() int64 {
        return 25
    },
}
```

### List Field {#list-field}

The `FormListField`  represents a list selection field in an HTML form, allowing users to choose from a predefined set of options which is based on `FormListFieldOption`. It supports both single and multiple selections.

#### Definition

```go
type FormListFieldOption struct {
	Label string
	Value interface{}
}

type FormListField struct {
	Name     string
	Label    string
	Type     string     // type of the list options
	Multiple bool
	Options  func() []FormListFieldOption
	ValueFn  func() interface{}
}
```

#### Properties

| Property | Description |
| ---- | ----|
| `Name` | The unique name of the field within the section scope. |
| `Label` | The label for the input. |
| `Type` | The type of the input fields. See [Field Types](#field-types) for the available types. |
| `Multiple` | Indicates whether the field allows multiple selections. |
| `Options` | A function that returns a list of options for the field. See [List Field Options](#list-field-options) |
| `ValueFn` | This function should return the current value of the input field. |

#### List Field Options {#list-field-options}

The `FormListFieldOption` represents an option in a list field.

##### Definition
```go
type FormListFieldOption struct {
    Label string
    Value interface{}
}
```

##### Properties

| Property | Description
|--- | ---
| `Label` | The display label of the option.
| `Value` | The value of the option. It must be of the same type as the [Type](#field-types) of the list field.

##### Usage Example

```go
countryField := sdkapi.FormListField{
    Name:     "country",
    Label:    "Select Country",
    Type:     "string",
    Multiple: false,
    Options: func() []sdkapi.FormListFieldOption {
        return []sdkapi.FormListFieldOption{
            {Label: "Philippines", Value: "PH"},
            {Label: "Canada", Value: "CA"},
            {Label: "United Kingdom", Value: "UK"},
        }
    },
    ValueFn: func() interface{} {
        // your custom list logic
        return "PH"
    },
}

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
```

### Multi Field {#multi-field}

The `FormMultiField` represents a structured form field that consists of multiple rows and columns. Each column defines a specific type of data, and each row contains values for those columns.

#### Definition

```go
type FormMultiField struct {
	Name    string
	Label   string
	Columns func() []FormMultiFieldCol
	ValueFn func() [][]FormFieldData
}
```

#### Properties

| Field | Description |
| ---- | ---- |
| `Name` | The name of the multi-field form. |
| `Label` | The label displayed for the multi-field form. |
| `Columns` | Function returning a list of [column definitions](#multi-field-column). |
| `ValueFn` | Function returning the values for each row and column. |

#### Usage Example

```go
sdkapi.FormMultiField{
    Name:  "wifi_rates",
    Label: "WiFi Rates",
    Columns: func() []sdkapi.FormMultiFieldCol {
        return []sdkapi.FormMultiFieldCol{
            {
                Name:  "amount",
                Label: "Amount",
                Type:  sdkapi.FormFieldTypeDecimal,
                ValueFn: func() interface{} {
                    return float64(0.0) // Default value
                },
            },
            {
                Name:  "wifi_time_seconds",
                Label: "WiFi Time (in seconds)",
                Type:  sdkapi.FormFieldTypeInteger,
                ValueFn: func() interface{} {
                    return 0 // Default value
                },
            },
            {
                Name:  "wifi_data_mb",
                Label: "Consumable Data (in megabytes)",
                Type:  sdkapi.FormFieldTypeInteger,
                ValueFn: func() interface{} {
                    return 0 // Default value
                },
            },
        }
    },
}
```


### String Field

The `FormStringField` represents a text or password input field in a form.

#### Definition

```go
type FormTextField struct {
	Name    string
	Label   string
	ValueFn func() string
	IsReadOnly bool // indicates if the field is read-only
	IsPassword bool // indicates if the field is a password field
}
```

#### Properties

| Field | Description |
|--- | --- |
| `Name`  | The name of the input field. |
| `Label` | The label displayed for the input field. |
| `IsReadOnly` | Indicates if the field is read-only. |
| `IsPassword` | Indicates if the field is a password field. |
| `ValueFn` | Function that returns the value for the input field. |

#### Usage Example

```go
sdkapi.FormStringField{
    Name: "fname",
    Label: "First Name",
    IsReadOnly: true,
    IsPassword: true,
    ValueFn: func () string {
        return "John Doe"
    }
}
```

### Text Field

The `FormTextField` represents a textarea field in a form.

#### Definition

```go
type FormTextField struct {
	Name    string
	Label   string
	ValueFn func() string
}
```

#### Properties

| Field | Description |
|--- | --- |
| `Name`  | The name of the input field. |
| `Label` | The label displayed for the input field. |
| `ValueFn` | Function that returns the value for the input field. |

#### Usage Example

```go
sdkapi.FormTextField{
    Name: "item_desc",
    Label: "Item Description",
    ValueFn: func () string {
        return "Lorem ipsum dolor sit amet..."
    }
}
```

## FormMultiFieldCol {#multi-field-column}

Represents a column in the [multi-field](#multi-field) form.

### Definition

```go
type FormMultiFieldCol struct {
	Name    string
	Label   string
	Type    string
	ValueFn func() interface{}
}
```

### Properties

| Field  | Description |
| ----  | ---- |
| `Name`  | The name of the column. |
| `Label` | The label displayed for the column. |
| `Type` | The [data type](#field-types) of the column. |
| `ValueFn` | Function that returns the default value for the column. |

---

## IHttpForm {#ihttpform}

The `IHttpForm` is primarily used for retrieving data from the HTTP form.

See [Saving Data](../guides/saving-data.md) for an example.

### Definition

```go
type IHttpForm interface {
	GetSection(section string) (sec IFormSection, ok bool)
	GetSections() []IFormSection

	GetStringValue(section string, name string) (string, error)
	GetStringValues(section string, name string) ([]string, error)

	GetIntValue(section string, name string) (int64, error)
	GetIntValues(section string, name string) ([]int64, error)

	GetFloatValue(section string, name string) (float64, error)
	GetFloatValues(section string, name string) ([]float64, error)

	GetBoolValue(section string, name string) (bool, error)
	GetBoolValues(section string, name string) ([]bool, error)

	GetMultiField(section string, name string) (IFormMultiField, error)
}
```

### Methods

#### GetSection

It returns a [IFormSection](#iformsection) of the form identified by the [section](#form-section)'s `Name` property.
A `IFormSection` can also be used to retrieve data from a form's section aside from the `IHttpForm`'s own methods.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    form, _ := formsAPI.GetForm("my-form", r)
    section := form.GetSection("settings")
}
```

#### GetSections

Returns all [IFormSection](#iformsection) in the form.

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    formsAPI := api.Http().Forms()
    form, _ := formsAPI.GetForm("my-form", r)
    sections := form.GetSections()
}
```

#### GetBoolValue

Returns the `boolean` value of a [boolean field](#boolean-field) in the form.

```go
val, err := form.GetBoolValue("section_name", "boolean_field_name")
```

#### GetBoolValues

Returns `[]boolean` for [list fields](#list-field) of type `sdkapi.FormFieldTypeBoolean`.

```go
vals, err := form.GetBoolValues("section_name", "boolean_list_field_name")
```

#### GetFloatValue

Returns the `float64` value of a [decimal field](#decimal-field) in the form.

```go
val, err := form.GetFloatValue("section_name", "decimal_field_name")
```

#### GetFloatValues

Returns `[]float64` for [list fields](#list-field) of type `sdkapi.FormFieldTypeDecimal`.

```go
vals, err := form.GetFloatValues("section_name", "decimal_list_field_name")
```


#### GetIntValue

Returns the `int64` value of an [integer field](#integer-field) in the form.

```go
val, err := form.GetIntValue("section_name", "integer_field_name")
```

#### GetIntValues

Returns `[]int64` value for [list fields](#list-field) of type `sdkapi.FormFieldTypeInteger`.

```go
vals, err := form.GetIntValues("section_name", "integer_list_Field_name")
```

#### GetMultiField

Returns a [IFormMultiField](#imultifield) instance of a multi field in the form.

```go
mf, err := form.GetMultiField("section_name", "multi_field_name")
```

#### GetStringValue

Returns the string value of a text (or textarea) field in the form.

```go
val, err := form.GetStringValue("section_name", "banner_text")
```

#### GetStringValues

Returns a slice of strings for [list fields](#list-field).

```go
vals, err := form.GetStringValues("section_name", "string_list_field_name")
```

---

## IFormSection

An `IFormSection` represents a section of the HTTP form. It can be used to retrieve the form's input values.

### Definition

```go
type IFormSection interface {
	GetName() string
	GetLabel() string
	GetFields() []IFormField

	GetStringValue(name string) (string, error)
	GetStringValues(name string) ([]string, error)

	GetIntValue(name string) (int64, error)
	GetIntValues(name string) ([]int64, error)

	GetFloatValue(name string) (float64, error)
	GetFloatValues(name string) ([]float64, error)

	GetBoolValue(name string) (bool, error)
	GetBoolValues(name string) ([]bool, error)

	GetMultiField(name string) (IFormMultiField, error)
}
```

### Usage Example

```go
func (w http.ResponseWriter, r *http.Request) {
    httpForm, err := api.Http().Forms().ParseForm("my-form", r)
    if err != nil {
        // handle error
    }

    section := httpForm.GetSection("section_name")
    fmt.Println(section) // IFormSection
}
```

### Methods

#### GetName

Returns the `Name` property of a form [section](#form-section).

#### GetLabel

Returns the `Label` property of a form [section](#form-section).

#### GetFields

Returns a collection of [IFormField](#iformfield).

#### GetBoolValue

Returns the `boolean` value of a [boolean field](#boolean-field) in the form.

```go
val, err := form.GetBoolValue("boolean_field_name")
```

#### GetBoolValues

Returns `[]boolean` for [list fields](#list-field) of type `sdkapi.FormFieldTypeBoolean`.

```go
vals, err := form.GetBoolValues("boolean_list_field_name")
```

#### GetFloatValue

Returns the `float64` value of a [decimal field](#decimal-field) in the form.

```go
val, err := form.GetFloatValue("decimal_field_name")
```

#### GetFloatValues

Returns `[]float64` for [list fields](#list-field) of type `sdkapi.FormFieldTypeDecimal`.

```go
vals, err := form.GetFloatValues("decimal_list_field_name")
```


#### GetIntValue

Returns the `int64` value of an [integer field](#integer-field) in the form.

```go
val, err := form.GetIntValue("integer_field_name")
```

#### GetIntValues

Returns `[]int64` value for [list fields](#list-field) of type `sdkapi.FormFieldTypeInteger`.

```go
vals, err := form.GetIntValues("integer_list_Field_name")
```

#### GetMultiField

Returns a [IFormMultiField](#imultifield) instance of a multi field in the form.

```go
mf, err := form.GetMultiField("multi_field_name")
```

#### GetStringValue

Returns the string value of a text (or textarea) field in the form.

```go
val, err := form.GetStringValue("banner_text")
```

#### GetStringValues

Returns a slice of strings for [list fields](#list-field).

```go
vals, err := form.GetStringValues("string_list_field_name")
```

---

## IFormMultiField

An `IFormMultiField` contains values of a [FormMultiField](#multi-field).
A multi-field can be obtained using [IHttpForm.GetMultiField](#getmultifield).

### Definition

```go
type IFormMultiField interface {
	NumRows() int
	GetStringValue(row int, name string) (string, error)
	GetIntValue(row int, name string) (int64, error)
	GetFloatValue(row int, name string) (float64, error)
	GetBoolValue(row int, name string) (bool, error)
}
```

### Usage Example

```go
func (w http.ResponseWriter, r *http.Request) {
    httpForm, err := api.Http().Forms().ParseForm("my-form", r)
    if err != nil {
        // handle error
    }
    multiField := httpForm.GetMultiField("section_name", "multi_field_name")
}
```

### Methods

#### NumRows

Returns the number of rows in a multi-field.

```go
rows := multiField.NumRows()
```

#### GetStringValue

Returns a `string` value of a [string](#string-field) or [text](#text-field) field.

```go
row := 1
col := "column_name"
value, err := multiField.GetStringValue(row, col)
```

#### GetIntValue

Returns a `int64` value of an [integer](#integer-field) field.

```go
row := 1
col := "column_name"
value, err := multiField.GetIntValue(row, col)
```

#### GetFloatValue

Returns a `float64` value of a [decimal](#decimal-field) field.

```go
row := 1
col := "column_name"
value, err := multiField.GetFloatValue(row, col)
```

#### GetBoolValue

Returns a `boolean` value of a [boolean](#boolean-field) field.

```go
row := 1
col := "column_name"
value, err := multiField.GetBoolValue(row, col)
```

---

## IFormField

The `IFormField` represents an input field in a section within an HTTP form.

### Definition

```go
type IFormField interface {
	GetName() string
	GetLabel() string
	GetType() string
	GetValue() interface{}
}
```

### Methods

| Method | Description
| --- | ---
| `GetName() string` | Returns the `Name` property of an input field.
| `GetLabel() string` | Returns the `Label` property of an input field.
| `GetType() string` | Returns the [Type](#field-types) of an input field.
| `GetValue() interface{}` | Returns the value of an input field.
