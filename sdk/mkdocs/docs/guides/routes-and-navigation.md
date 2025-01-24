# Routes and Navigation

Routes are used to map URL patterns to a functionality or components in your application.
When a user uses a browser to navigate to a specific URL, the router will match the URL to a registered `route` and executes the handler defined in that route.

## Types of Routes {#registering-routes}
There two (2) types of routes:

- `plugin routes` - Accessible to all users.
- `admin routes` - Accessible to authenticated admin accounts.

### Plugin Routes {#plugin-routes}

Below is an example on how to register a route to the [plugin router](../api/http-router-api.md#plugin-router).
Any route registered to the plugin router are categorized as `plugin route`.

```go title="plugins/local/com.mydomain.myplugin/main.go"
package main

import (
	"net/http"

	sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.PluginApi) {
    pluginRouter := api.Http().HttpRouter().PluginRouter()
    pluginRouter.Get("/welcome/{name}", func (w http.ResponseWriter, r *http.Request) {
        vars := api.Http().MuxVars(r)
        name := vars["name"]

        welcomePage := views.WelcomePage(name)
        api.Http().HttpResponse().PortalView(w, r, sdkapi.ViewPage{
            PageContent: welcomePage,
        })
    }).name("portal:welcome")
}
```

```templ title="plugins/local/com.mydomain.myplugin/resources/views/welcome.templ"
package views

templ WelcomePage(name string) {
    <p>Welcome, { name }</p>
}
```

In this example, we registered a plugin route named `portal:welcome` that executes when a user navigates to `/welcome/{name}`.
Then we extract the `name` URL parameter using [IHttpApi.MuxVars](../api/http-api.md#muxvars) method and display the `welcome.templ` [view template](../api/http-response.md#template-parsing).

The `plugin router` has additional methods aside from `Get`. See the [Router Instance](../api/http-router-api.md#router-instance) documentation.

### Admin Routes {#admin-routes}
Admin routes are very similar to [plugin routes](#plugin-routes), but are only accessible by authenticated admin accounts.

Below is an example on how to register an admin route. Any route registered to the [admin router](../api/http-router-api.md#admin-router) are categorized as `admin route`.

```go title="plugins/local/com.mydomain.myplugin/main.go"
package main

import (
	"net/http"

	sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.PluginApi) {
    adminRouter := api.Http().HttpRouter().AdminRouter()
    adminRouter.Get("/welcome/{name}", func (w http.ResponseWriter, r *http.Request) {
        vars := api.Http().MuxVars(r)
        name := vars["name"]

        welcomePage := views.WelcomePage(name)
        api.Http().HttpResponse().AdminView(w, r, sdkapi.ViewPage{
            PageContent: welcomePage,
        })
    }).name("admin:welcome")
}
```

## Navigation Menu

We have two (2) types of navigation menu:

- `portal menu` - Menu items that are available in the captive portal.
- `admin menu` - Menu items that are available in the admin web interface.

### Portal Menu Items

In the example above, we [registered a plugin route](#plugin-routes) to the plugin router. We named the route `portal:welcome`. Take note of the name of the route since we will use it as reference when registering a portal menu item.

To add a portal menu item that links to `portal:welcome` route, we will use the [INavsApi.PortalNavsFactory](../api/http-navs-api.md#portalnavsfactory) method.

```go title="plugins/local/com.mydomain.myplugin/main.go"
// rest of the init function code...

navsAPI := api.Http().Navs()
navsAPI.PortalNavsFactory(func(r *http.Request) []PortalNavItemOpt {
    return []sdkapi.AdminNavItemOpt{
        {
            Label:     "Welcome",                   // Menu display text
            RouteName: "portal:welcome",             // Link to the route
            IconUrl: api.Http().Helpers().ResourcePath("assets/images/some-image.jpg"),
            RouteParams: map[string]string{
                "name": "John",
            },
        },
    }
})
```

Now, visit [localhost:3000](http://localhost:3000) to see if the menu item appears.

### Admin Menu Items

In the example above, we also [registered an admin route](#admin-routes) to the admin router. We named the route `admin:welcome`. Take note of the name of the route since we will use it as reference when registering an admin menu item.

To add an admin menu item that links to `admin:welcome` route, we will use the [INavsApi.AdminNavsFactory](../api/http-navs-api.md#adminnavsfactory) method.

```go title="plugins/local/com.mydomain.myplugin/main.go"
// rest of the init function code...

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

Now, visit [localhost:3000/admin](http://localhost:3000/admin) to see if the admin menu item is present.

## Generating URLs {#generating-urls}

To generate URLs for a route, we will use the name of the route. In the example above, we named the route `portal:welcome` with a URL path `/welcome/{name}`.
To generate the URL for the route `portal:welcome`, we will use the [IHttpRouterApi.UrlForRoute](../api/http-router-api.md#urlforroute) method:

```go
url := api.Http().HttpRouter().UrlForRoute("portal:welcome", "name", "John")
fmt.Println(url)
// => /p/com.mydomain.myplugin/1.0.0/welcome/John
```

The example showed us how to generate a URL from a named route and how to set a value to a URL patameter (we replaced `{name}` parameter with the value `John`).

For multiple URL params, you can add more pairs to the `UrlForRoute` method. For example, if the route URL is `/welcome/:name/:gender/:age`:

```go
url := api.Http().HttpRouter().UrlForRoute("portal:welcome", "name", "John", "gender", "male", "age", "21")
fmt.Println(url)
// => /p/com.mydomain.myplugin/1.0.0/welcome/John/male/21
```

If you want to generate route for a third-party plugin or the core system, we will use the [IHttpRouterApi.UrlForPkgRoute](../api/http-router-api.md#urlforpkgroute) method:

```go
url := api.Http().HttpRouter().UrlForPkgRoute("com.flarego.core", "admin:index", "name", "John")
```

## URL Parameters {#url-params}

Routes can have path variables (URL parameters) that can be extracted from the URL. In the example above, the `name` parameter is extracted from the URL `/welcome/{name}` with:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    vars := api.Http().MuxVars(r)
    name := vars["name"]
}
```

If you just want to extract the query parameters from the URL, you can use the `r.URL.Query()` method in your handler function.
For example, if the URL is `/welcome?name=John` and you want to get the value of `name` query parameter:

```go
// handler
func (w http.ResponseWriter, r *http.Request) {
    queryParams := r.URL.Query()
    name := queryParams.Get("name")
    fmt.Println(name) // "John"
    // ...
}
```
