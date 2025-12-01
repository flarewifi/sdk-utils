# INavsApi

The `INavsApi` is used to register navigation items for both portal and admin web interfaces.

To get an instance of `INavsApi`:

```go title="main.go"
navsAPI := api.Http().Navs()
fmt.Println(navsAPI) // INavsApi
```

## INavsApi methods

Below are the methods available in `INavsApi`:

### AdminNavsFactory

This method is used to generate the admin navigation items of the plugin. It takes function parameter that accepts an `*http.Request` and returns a slice of [AdminNavItemOpt](#admin-nav-item):

```go
func(r *http.Request) []AdminNavItemOpt
```

Below is an example of registering a function that generates admin navigation menu items:

```go
navsAPI := api.Http().Navs()
navsAPI.AdminNavsFactory(func(r *http.Request) []AdminNavItemOpt {
    return []sdkapi.AdminNavItemOpt{
        {
            Category:  sdkapi.NavCategorySystem,    // Category of the menu item
            Label:     "Welcome",                   // Menu display text
            RouteName: "admin:welcome",             // Link to the route
            Keywords: []string{"home", "welcome"},  // Used for searching menu items
        },
    }
})
```

### PortalNavsFactory

This method is used to add items to the portal menu items. It takes a function parameter that accepts `*http.Request` and returns a slice of [PortalNavItemOpt](#portal-nav-item):

```go
func(r *http.Request) []PortalNavItemOpt
```

Below is an example of registering a function that generates portal navigation menu items:

```go
navsAPI := api.Http().Navs()
navsAPI.PortalNavsFactory(func(r *http.Request) []PortalNavItemOpt {
    return []sdkapi.PortalNavItemOpt{
        {
            Label:     "Welcome",                       // Menu display text
            IconFile: "some-image.jpg",                 // File in the resources/images folder
            RouteName: "portal:welcome",                // Link to the route
            RouteParams: map[string]string{
                "name": "John",
            },
        },
    }
})
```

### GetAdminNavs

Returns the consolidated navigation list from all plugins for the admin dashboard. It accepts `*http.Request` as an argument and returns a slice of [AdminNavList](#admin-nav-list).

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    navsAPI := api.Http().Navs()
    adminNavs := navsAPI.GetAdminNavs(r)
    fmt.Println(adminNavs) // []AdminNavList
}
```

### GetPortalItems

Returns the consolidated navigation list from all plugins for the portal. It accepts `*http.Request` as an argument and returns a slice of [PortalNavItem](#portal-nav-item).

```go
func (w http.ResponseWriter, r *http.Request) {
    navsAPI := api.Http().Navs()
    portalItems := navsAPI.GetPortalItems(r)
    fmt.Println(portalItems) // []PortalNavItem
}
```

## AdminNavItemOpt {#admin-nav-item}

The `AdminNavItemOpt` is a Go struct that represents an admin navigation menu item:

```go
type AdminNavItemOpt struct {
	Category    INavCategory
	Label       string
	RouteName   string
	RouteParams map[string]string
	ExtraAttrs  map[string]any
	Keywords    []string // Used for admin nav search indexing
}
```

The available `INavCategory` options are:

- `sdkapi.NavCategoryQuickAccess`
- `sdkapi.NavCategorySystem`
- `sdkapi.NavCategoryPayments`
- `sdkapi.NavCategoryThemes`
- `sdkapi.NavCategoryNetwork`
- `sdkapi.NavCategoryTools`

## PortalNavItemOpt {#portal-nav-item-opt}

The `PortalNavItemOpt` is a Go struct that represents a portal navigation menu item option:

```go
type PortalNavItemOpt struct {
	Label       string
	IconFile    string
	RouteName   string
	RouteParams map[string]string
	ExtraAttrs  map[string]any
	Metadata    any
}
```

## AdminNavList {#admin-nav-list}

The `AdminNavList` contains a list of admin navigation items which can be used by admin theme plugins to display the navigation menu.

```go
type AdminNavList struct {
	Category INavCategory
	Label    string
	Items    []AdminNavItem
}
```

## AdminNavItem {#admin-nav-item-result}

The `AdminNavItem` is a Go struct that represents an individual admin menu item returned by `GetAdminNavs()`.

```go
type AdminNavItem struct {
	Label     string
	RouteUrl  string
	IsCurrent bool     // true if current active route
	Keywords  []string // Used for admin nav search indexing
}
```

## PortalNavItem {#portal-nav-item}

The `PortalNavItem` is a Go struct that represents an individual portal menu item returned by `GetPortalItems()`.

```go
type PortalNavItem struct {
	ID         string
	Label      string
	IconUrl    string
	RouteUrl   string
	ExtraAttrs map[string]any
}
```
