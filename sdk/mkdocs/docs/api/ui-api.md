# IUIApi

The `IUIApi` provides reusable UI components for building consistent interfaces. These components use Bootstrap styling and are designed to work with the templ templating system.

To get an instance of `IUIApi`:

```go
uiAPI := api.UI()
fmt.Println(uiAPI) // IUIApi
```

## IUIApi Methods

### Pagination

Returns a templ component for displaying pagination controls. Use this method to add pagination to lists and tables.

```go
func (w http.ResponseWriter, r *http.Request) {
    opts := &sdkapi.UIPaginationOpts{
        PageURL:       api.Http().Helpers().UrlForRoute("admin:logs:index"),
        PerPage:       10,
        CurrentPage:   2,
        ItemsCount:    100,
        MaxPagerCount: 5,
        ExtraParams: map[string]string{
            "filter": "active",
        },
    }

    pagination := api.UI().Pagination(opts)

    // Use pagination in your templ template
    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: views.LogsList(logs, pagination),
    })
}
```

## Types

### UIPaginationOpts

The `UIPaginationOpts` struct configures the pagination component:

```go
type UIPaginationOpts struct {
    PageURL       string            // Base URL for pagination links
    PerPage       int               // Number of items per page
    CurrentPage   int               // Current active page (1-based)
    ItemsCount    int64             // Total number of items
    MaxPagerCount int               // Maximum number of page links to show
    ExtraParams   map[string]string // Additional query parameters to preserve
}
```

| Field | Type | Description |
| --- | --- | --- |
| `PageURL` | `string` | The base URL for pagination links. Use `api.Http().Helpers().UrlForRoute()` to generate this. |
| `PerPage` | `int` | Number of items to display per page. Common values: 10, 25, 50. |
| `CurrentPage` | `int` | The currently active page number (1-based index). |
| `ItemsCount` | `int64` | The total count of all items being paginated. |
| `MaxPagerCount` | `int` | Maximum number of page number links to display. If set to 5, shows something like "1 2 3 4 5 ..." |
| `ExtraParams` | `map[string]string` | Additional query parameters to include in pagination links (e.g., filters, search terms). |

## Usage Examples

### Basic Pagination

```go
func listDevices(w http.ResponseWriter, r *http.Request) {
    // Get current page from query params
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }

    perPage := 10
    offset := (page - 1) * perPage

    // Get total count and paginated results
    devices, err := api.Devices().GetPaginated(r.Context(), offset, perPage)
    if err != nil {
        // handle error
        return
    }

    totalCount, err := api.Devices().Count(r.Context())
    if err != nil {
        // handle error
        return
    }

    // Create pagination component
    pagination := api.UI().Pagination(&sdkapi.UIPaginationOpts{
        PageURL:       api.Http().Helpers().UrlForRoute("admin:devices:list"),
        PerPage:       perPage,
        CurrentPage:   page,
        ItemsCount:    totalCount,
        MaxPagerCount: 7,
    })

    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: views.DevicesList(devices, pagination),
    })
}
```

### Pagination with Filters

```go
func listSessions(w http.ResponseWriter, r *http.Request) {
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }

    // Get filter from query params
    status := r.URL.Query().Get("status")

    perPage := 25
    offset := (page - 1) * perPage

    // Get filtered and paginated results
    sessions, totalCount, err := api.Sessions().GetFiltered(r.Context(), status, offset, perPage)
    if err != nil {
        // handle error
        return
    }

    // Include filter in pagination links
    pagination := api.UI().Pagination(&sdkapi.UIPaginationOpts{
        PageURL:       api.Http().Helpers().UrlForRoute("admin:sessions:list"),
        PerPage:       perPage,
        CurrentPage:   page,
        ItemsCount:    totalCount,
        MaxPagerCount: 5,
        ExtraParams: map[string]string{
            "status": status, // Preserve filter across pages
        },
    })

    api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
        PageContent: views.SessionsList(sessions, pagination),
    })
}
```

### Using Pagination in Templ Templates

```templ
// views/devices_list.templ
package views

import "github.com/a-h/templ"

templ DevicesList(devices []Device, pagination templ.Component) {
    <div class="panel panel-default">
        <div class="panel-heading">
            <h3 class="panel-title">Devices</h3>
        </div>
        <div class="panel-body">
            <table class="table table-striped">
                <thead>
                    <tr>
                        <th>MAC Address</th>
                        <th>IP Address</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    for _, device := range devices {
                        <tr>
                            <td>{ device.MacAddress }</td>
                            <td>{ device.IpAddress }</td>
                            <td>{ device.Status }</td>
                        </tr>
                    }
                </tbody>
            </table>

            // Render the pagination component
            @pagination
        </div>
    </div>
}
```

## Best Practices

- **Always validate page numbers**: Ensure the current page is at least 1 and doesn't exceed the total pages
- **Use consistent `PerPage` values**: Common values are 10, 25, 50, or 100
- **Preserve filters in `ExtraParams`**: When users apply filters, include them in pagination links so filters persist
- **Set reasonable `MaxPagerCount`**: 5-7 is typical; too many page links can be overwhelming
- **Handle empty results gracefully**: Check if `ItemsCount` is 0 before rendering pagination
