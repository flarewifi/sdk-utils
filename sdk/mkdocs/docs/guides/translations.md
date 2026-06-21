# Translations

## 1. Basic Translations

To translate texts, we will use the [Translate](../api/plugin-api.md#translate) method from [PluginApi](../api/plugin-api.md).

The [Translate](../api/plugin-api.md#translate) method receives a message type and message key string and returns the translated text. The method will look for the translation in the `resources/translations/[lang]/[type]/[file].txt` file in your plugin.

The `[lang]` placeholder is the language code set in the [application config](../api/config-api.md#application), e.g. `en` for English.

The `[type]` placeholder is the type of the translation message, e.g. `label` for labels and button texts. Other types are `info` and `error`.

The `[file]` placeholder is the message key of the translation message.

For example, to translate message key "save" to the target language, we can use the following code:
```go
saveText := api.Translate("label", "save")
```

In this example, the [IPluginApi.Translate](../api/plugin-api.md#translate) method will look for the file `resources/translations/en/label/save.txt`. The contents of the file will be used as the template for the translated text. For more advanced translations, see [PluginApi.Translate](../api/plugin-api.md#translate) method documentation.

The translate method can also be called within views using the `{ api.Translate("label", "save") }` helper method. For example:

```html
<h1>{ api.Translate("label", "save") }</h1>
```

When a translation file is missing, the running code will automatically generate it for each supported language. You can edit the translation files once they are generated.

## 2. Translations With Variables

Let's say you want to display an amount in a label. You can use the  [IPluginApi.Translate](../api/plugin-api.md#translate) method with variables to achieve this. For example, if you have the following translation text with a vairable `amount`:

``` title="resources/translations/en/label/paid_amount.txt"
You paid <% .currency %> <% .amount %>
```

Then you can substitue the variable `amount` with the actual value following like this:
```go
txt := api.Translate("label", "paid_amount", "currency", "PHP", "amount", 100)
```

Likewise, you can use the [IPluginApi.Translate](../api/plugin-api.md#translate) method in views to achieve the same result:
```templ
<p>{ api.Translate("label", "paid_amount", "amount", 100) }</p>
```

## 3. Best Practices for Translation Keys

### Use Generic, Professional Wording

**Avoid ellipsis (...), multiple exclamation marks (!!), and informal language in translation messages.** Use clear, professional wording that is appropriate for enterprise software.

#### ❌ Bad Examples:
```go
// Avoid ellipsis - too informal
api.Translate("info", "Loading data...")

// Avoid multiple exclamation marks - too excited
api.Translate("success", "Data saved successfully!!")

// Avoid casual language
api.Translate("error", "Oops! Something went wrong")
```

#### ✅ Good Examples:
```go
// Use clear, professional wording
api.Translate("info", "Loading data")

// Use single punctuation appropriately
api.Translate("success", "Data saved successfully")

// Use professional error messages
api.Translate("error", "An error occurred while processing your request")
```

### Use Generic Phrases, Not Concatenated Variables

When creating translation keys, avoid concatenating variables directly in the key. Instead, use generic descriptive phrases and pass variables as parameters. This approach provides better maintainability and allows translators to understand the context properly.

#### ❌ Bad Example (Concatenating Variables):
```go
// Don't do this - concatenating variables in translation keys
errStr = validtr.api.Translate("error", fieldLabel+" must be at least "+fmt.Sprint(min)+" characters")
```

#### ✅ Good Example (Generic Keys with Placeholders):
```go
// Do this - use generic keys with variables passed separately
errStr = validtr.api.Translate("error", "Input value does not meet the required minimum characters", "label", fieldLabel, "min", min)
```

### Translation File Structure

For the good example above, create a translation file at:
```
resources/translations/en/error/Input value does not meet the required minimum characters.txt
```

With content using Go template placeholders:
```
<% .label %> must be at least <% .min %> characters
```

### Common Generic Translation Keys

For form validation, use these generic keys:

- `"Input field is required"` → `<% .label %> is required`
- `"Input value must be a valid integer"` → `<% .label %> must be a valid integer`
- `"Input value does not meet the required minimum"` → `<% .label %> must be at least <% .min %>`
- `"Input value exceeds the maximum allowed"` → `<% .label %> must not exceed <% .max %>`
- `"File upload is required"` → `<% .label %> is required`
- `"Invalid file extension uploaded"` → `Invalid file extension for <% .label %>. Allowed extensions: <% .extensions %>`

