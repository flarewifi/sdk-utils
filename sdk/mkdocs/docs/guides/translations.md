# Translations

## 1. Basic Translations

To translate texts, we will use the [Translate](../api/plugin-api.md#translate) method from [PluginApi](../api/plugin-api.md).

The [Translate](../api/plugin-api.md#translate) method receives a **message type** and the **English source text** and returns the translated text. The method looks the text up in a single per-language JSON catalog in your plugin:

```
resources/translations/<lang>.json
```

The `<lang>` placeholder is the language code set in the [application config](../api/config-api.md#application), e.g. `en` for English. There is **one JSON file per language** (`en.json`, `es.json`, `fr.json`, …) — not a tree of per-string `.txt` files.

Each catalog is a JSON object keyed first by message type, then by the English source text, whose value is the translation:

``` json title="resources/translations/es.json"
{
  "label": {
    "Save": "Guardar",
    "Cancel": "Cancelar"
  },
  "error": {
    "Invalid input": "Entrada no válida"
  }
}
```

The six supported message types (the top-level keys of every catalog) are: `label`, `error`, `success`, `info`, `warning`, and `type`.

**The lookup key is the English source text itself** — you pass the full, natural-language message, not a short symbolic key:

```go
saveText := api.Translate("label", "Save")
```

Here `Translate` looks in the active language's catalog for `["label"]["Save"]`. The translate method can also be called within views using the same helper:

```html
<h1>{ api.Translate("label", "Save") }</h1>
```

### The English catalog is the registry

`en.json` is the source-of-truth **registry** of every translatable string. In it, each value is identical to its key (`"Save": "Save"`) — it exists to enumerate the strings your plugin uses, and its presence marks a component as migrated.

Other languages are intentionally **sparse**: a translator only fills in the strings that have been translated. When a key is **absent** (or the whole `<lang>.json` is missing), `Translate` falls back to the **English source text you passed in** — it is never written back to disk as English, and it never errors. This means:

- You can call `Translate` with a brand-new string and it will render correctly (as English) before any catalog entry exists.
- Untranslated strings are simply omitted from non-English files, keeping them small and easy to review.

### Populating catalogs (tooling, not runtime)

Unlike the old system, the runtime **does not auto-generate** translation files. Missing keys are added to `en.json` by the `translations-mcp` tooling, whose `sync` command scans your plugin's `.go` and `.templ` source for `Translate("type", "text")` calls and appends any missing keys (with `value == key`) to `en.json`. Translators then fill in each `<lang>.json`. The `check` / `find_untranslated` commands report coverage gaps.

In development (`GO_ENV=dev`) catalogs are re-read from disk on every call, so edits to a `<lang>.json` show up live without a rebuild; production caches parsed catalogs and the build minifies them into compact single-line JSON.

## 2. Translations With Variables

Let's say you want to display an amount in a label. You can use the  [IPluginApi.Translate](../api/plugin-api.md#translate) method with variables to achieve this. Placeholders use the `<% .name %>` delimiters (a Go `text/template` with custom `<% %>` delims — **not** `{{ }}`, which would print literally).

The English source text — placeholders and all — is the catalog key. So a translated entry looks like this:

``` json title="resources/translations/es.json"
{
  "label": {
    "You paid <% .currency %> <% .amount %>": "Pagaste <% .currency %> <% .amount %>"
  }
}
```

Then you substitute the variables `currency` and `amount` with actual values by passing **key/value pairs** after the text:
```go
txt := api.Translate("label", "You paid <% .currency %> <% .amount %>", "currency", "PHP", "amount", 100)
```

Likewise, you can use the [IPluginApi.Translate](../api/plugin-api.md#translate) method in views to achieve the same result:
```templ
<p>{ api.Translate("label", "You paid <% .currency %> <% .amount %>", "currency", "PHP", "amount", 100) }</p>
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

For the good example above, the English source text becomes a key under the `error` type in `en.json`, and each language file supplies its own translation using the same `<% %>` placeholders:

``` json title="resources/translations/en.json (registry — value == key)"
{
  "error": {
    "Input value does not meet the required minimum characters": "Input value does not meet the required minimum characters"
  }
}
```

``` json title="resources/translations/es.json (translated, with placeholders)"
{
  "error": {
    "Input value does not meet the required minimum characters": "<% .label %> debe tener al menos <% .min %> caracteres"
  }
}
```

### Common Generic Translation Keys

For form validation, use these generic keys:

- `"Input field is required"` → `<% .label %> is required`
- `"Input value must be a valid integer"` → `<% .label %> must be a valid integer`
- `"Input value does not meet the required minimum"` → `<% .label %> must be at least <% .min %>`
- `"Input value exceeds the maximum allowed"` → `<% .label %> must not exceed <% .max %>`
- `"File upload is required"` → `<% .label %> is required`
- `"Invalid file extension uploaded"` → `Invalid file extension for <% .label %>. Allowed extensions: <% .extensions %>`

