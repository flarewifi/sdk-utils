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
            Icon:      "<i class='bi bi-house'></i>", // Icon HTML tag
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
	Icon        string
	RouteName   string
	RouteParams map[string]string
	ExtraAttrs  map[string]any
	Keywords    []string // Used for admin nav search indexing
	Order       int      // Sort order within category (lower numbers appear first, default: 5000)
}
```

### Fields

- **Category** - The navigation category where this item will appear
- **Label** - The display text for the menu item
- **Icon** - An HTML tag string for the menu item icon (e.g., `"<i class='bi bi-gear'></i>"`)
- **RouteName** - The named route this menu item links to
- **RouteParams** - Optional route parameters as key-value pairs
- **ExtraAttrs** - Optional HTML attributes for the menu item element
  - Used to add custom CSS classes, data attributes, or other HTML attributes
  - Example: `map[string]any{"class": "highlight", "data-id": "123"}`
  - Theme plugins can use these attributes for custom styling or behavior
- **Keywords** - Search keywords for the admin navigation search feature
- **Order** - Sort order within the category (default: 5000 if not specified)
  - Lower numbers appear first (e.g., 1000 appears before 5000)
  - Higher numbers appear last (e.g., 9999 appears at the end)
  - Use this to control the position of your menu items

### Navigation Categories

The available `INavCategory` options are:

- `sdkapi.NavCategoryQuickAccess` - Quick access items (top 5 most visited)
- `sdkapi.NavCategorySystem` - System settings and configuration
- `sdkapi.NavCategoryPayments` - Payment and billing related items
- `sdkapi.NavCategoryThemes` - Theme selection and customization
- `sdkapi.NavCategoryNetwork` - Network configuration
- `sdkapi.NavCategoryTools` - Utility tools and features

### Ordering Examples

```go
// Item appears first in System category
{
    Category:  sdkapi.NavCategorySystem,
    Label:     "General Settings",
    Icon:      "<i class='bi bi-gear'></i>",
    RouteName: "admin:general",
    Order:     1000,
}

// Item appears in middle (default position)
{
    Category:  sdkapi.NavCategorySystem,
    Label:     "Bandwidth Settings",
    Icon:      "<i class='bi bi-speedometer2'></i>",
    RouteName: "admin:bandwidth:settings",
    Order:     4000,
}

// Item appears last in System category
{
    Category:  sdkapi.NavCategorySystem,
    Label:     "Shutdown",
    Icon:      "<i class='bi bi-power'></i>",
    RouteName: "admin:power:shutdown",
    Order:     9999,
}
```

### ExtraAttrs Examples

The `ExtraAttrs` field allows you to add custom HTML attributes to navigation menu items. This is useful for:

- Adding custom CSS classes for styling
- Adding data attributes for JavaScript interactions
- Adding accessibility attributes
- Any other HTML attributes supported by the theme

```go
// Add custom CSS class
{
    Category:   sdkapi.NavCategorySystem,
    Label:      "Important Settings",
    RouteName:  "admin:important",
    ExtraAttrs: map[string]any{
        "class": "nav-highlight",
    },
}

// Add multiple attributes
{
    Category:   sdkapi.NavCategoryTools,
    Label:      "External Tool",
    RouteName:  "admin:external-tool",
    ExtraAttrs: map[string]any{
        "class":       "external-link",
        "target":      "_blank",
        "data-id":     "tool-123",
        "data-source": "plugin",
    },
}

// Add accessibility attributes
{
    Category:   sdkapi.NavCategorySystem,
    Label:      "Advanced",
    RouteName:  "admin:advanced",
    ExtraAttrs: map[string]any{
        "aria-label": "Advanced system settings",
        "role":       "menuitem",
    },
}
```

**Note:** The actual rendering of `ExtraAttrs` depends on the theme plugin. Not all themes may support all attributes.

## PortalNavItemOpt {#portal-nav-item-opt}

The `PortalNavItemOpt` is a Go struct that represents a portal navigation menu item option:

```go
type PortalNavItemOpt struct {
	Label       string
	IconFile    string
	RouteName   string
	RouteParams map[string]string
	ExtraAttrs  map[string]any // HTML attributes for the menu item element
	Metadata    any            // Custom metadata for theme plugins
}
```

### Fields

- **Label** - The display text for the menu item
- **IconFile** - Path to the icon file (relative to `resources/assets/images/`)
- **RouteName** - The named route this menu item links to
- **RouteParams** - Optional route parameters as key-value pairs
- **ExtraAttrs** - Optional HTML attributes for the menu item element
  - Used to add custom CSS classes, data attributes, or other HTML attributes
  - Example: `map[string]any{"class": "featured", "target": "_blank"}`
- **Metadata** - Custom metadata that theme plugins can use for additional functionality

### Example

```go
navsAPI.PortalNavsFactory(func(r *http.Request) []PortalNavItemOpt {
    return []sdkapi.PortalNavItemOpt{
        {
            Label:     "WiFi Access",
            IconFile:  "wifi-icon.png",
            RouteName: "portal:wifi",
            ExtraAttrs: map[string]any{
                "class":    "featured-item",
                "data-id":  "wifi-access",
            },
            Metadata: map[string]any{
                "priority": "high",
                "category": "network",
            },
        },
    }
})
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
	Label      string
	Icon       string
	RouteUrl   string
	IsCurrent  bool               // true if current active route
	Keywords   []string           // Used for admin nav search indexing
	ExtraAttrs map[string]any     // HTML attributes (passed from AdminNavItemOpt)
	Order      int                // Sort order within category
}
```

### Fields

- **Label** - The display text for the menu item
- **Icon** - An HTML tag string for the menu item icon (passed from `AdminNavItemOpt.Icon`)
- **RouteUrl** - The full URL for the menu item (generated from RouteName)
- **IsCurrent** - `true` if this is the currently active route
- **Keywords** - Search keywords for the admin navigation search feature
- **ExtraAttrs** - HTML attributes passed from `AdminNavItemOpt.ExtraAttrs`
  - Theme plugins use these to render custom attributes on menu elements
- **Order** - Sort order value (used for sorting, already applied)

**Note:** Items returned by `GetAdminNavs()` are already sorted by the `Order` field within each category.

## PortalNavItem {#portal-nav-item}

The `PortalNavItem` is a Go struct that represents an individual portal menu item returned by `GetPortalItems()`.

```go
type PortalNavItem struct {
	ID         string
	Label      string
	IconUrl    string
	RouteUrl   string
	ExtraAttrs map[string]any // HTML attributes (passed from PortalNavItemOpt)
}
```

### Fields

- **ID** - Unique identifier for the menu item
- **Label** - The display text for the menu item
- **IconUrl** - The full URL to the icon image
- **RouteUrl** - The full URL for the menu item (generated from RouteName)
- **ExtraAttrs** - HTML attributes passed from `PortalNavItemOpt.ExtraAttrs`
  - Theme plugins use these to render custom attributes on menu elements
