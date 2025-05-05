# UIApi

The `UIApi` is used to get reusable UI template using bootstrap.

## 1. UIApi methods {#httpapi-methods}

The following are the available methods in `UIApi`:

### Pagination

Utilize the `Pagination` method to get the pagination template:

```go
// http handler
func (w http.ResponseWriter, r *http.Request) {
	opts := sdkapi.UIPaginationOpts{
		PageURL:     api.Helpers().UrlForRoute("admin:logs:index"),
		PerPage:     10,
		CurrentPage: 2,
		ItemsCount:  100,
		ExtraParams: map[string]string{
			"params_key":     "params_value",
		},
	}

 	pagination := api.UI().Pagination(opts)

  // Use `pagination` template here.
}
```
