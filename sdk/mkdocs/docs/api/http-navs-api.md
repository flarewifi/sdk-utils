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
}
```

The available `INavCategory` options are:

- `sdkapi.NavCategorySystem`
- `sdkapi.NavCategoryPayments`
- `sdkapi.NavCategoryThemes`
- `sdkapi.NavCategoryNetwork`
- `sdkapi.NavCategoryTools`

## PortalNavItemOpt {#portal-nav-item}

The `PortalNavItemOpt` is a Go struct the represents a portal navigation menu item.

```go
type PortalNavItemOpt struct {
	Label       string
	IconUrl     string
	RouteName   string
	RouteParams map[string]string
}
```

## AdminNavList {#admin-nav-list}

The `AdminNavList` contains a list of admin navigation items which can be used by admin theme plugins to display the navigation menu.

```go
type AdminNavList struct {
	Category string
	Items    []AdminNavItem
}

type AdminNavItem struct {
	Label    string
	RouteUrl string
}
```

## PortalNavItem {#portal-nav-item}

The `PortalNavItem` is a Go struct that represents an invidual portal menu item.

```go
type PortalNavItem struct {
	Label    string
	IconUrl  string
	RouteUrl string
}
```
