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

## 2. Translations With Variables

Let's say you want to display an amount in a label. You can use the  [IPluginApi.Translate](../api/plugin-api.md#translate) method with variables to achieve this. For example, if you have the following translation text with a vairable `amount`:

``` title="resources/translations/en/label/paid_amount.txt"
You paid <% .amount %>
```

Then you can substitue the variable `amount` with the actual value like this:
```go
txt := api.Translate("label", "paid_amount", "amount", 100)
```

Likewise, you can use the [IPluginApi.Translate](../api/plugin-api.md#translate) method in views to achieve the same result:
```templ
<p>{ api.Translate("label", "paid_amount", "amount", 100) }</p>
```
