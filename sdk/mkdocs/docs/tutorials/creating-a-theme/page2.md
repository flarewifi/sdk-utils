# Customizing the Admin Interface

This page covers advanced customization options for the admin theme, including navigation, notifications, and dashboard components.

## Navigation Customization

The admin theme allows you to customize how navigation items are displayed. The `LayoutBuilder` receives navigation data that you can use to create custom menus.

### Accessing Navigation Data

```go
navs := api.Http().Navs().GetAdminNavs(r)
var navItems []sdkapi.AdminNavItem
for _, nav := range navs {
    navItems = append(navItems, nav.Items...)
}
```

### Creating Custom Navigation

In your `layout.templ`, you can iterate through the navigation items:

```templ
templ AdminLayout(api sdkapi.IPluginApi, data AdminLayoutData) {
    // ... head ...
    <body>
        <nav class="navbar">
            <div class="nav-container">
                for _, nav := range data.Navs {
                    <div class="nav-section">
                        <h3>{ nav.Title }</h3>
                        <ul>
                            for _, item := range nav.Items {
                                <li>
                                    <a href={ item.Url } class={ item.Class }>
                                        { item.Label }
                                    </a>
                                </li>
                            }
                        </ul>
                    </div>
                }
            </div>
        </nav>
        // ... rest of layout ...
    </body>
}
```

## Notifications System

The admin theme includes a notifications system. You can customize how notifications are displayed and handled.

### Notification Data Structure

```go
notifs, err := api.Notification().GetUnreadNotifications(r.Context())
if err != nil {
    notifs = []sdkapi.Notification{}
}
```

Each notification has:
- `ID`: Unique identifier
- `Type`: Notification type (info, warning, error)
- `Message`: The notification text
- `CreatedAt`: Timestamp
- `Read`: Whether it's been read

### Custom Notification Display

```templ
templ NotificationDropdown(notifications []sdkapi.Notification) {
    <div class="notification-dropdown">
        <button class="notification-btn">
            <i class="icon-bell"></i>
            if len(notifications) > 0 {
                <span class="badge">{ len(notifications) }</span>
            }
        </button>
        <div class="notification-menu">
            if len(notifications) == 0 {
                <p>No new notifications</p>
            } else {
                for _, notif := range notifications {
                    <div class="notification-item" data-type={ notif.Type }>
                        <p>{ notif.Message }</p>
                        <small>{ notif.CreatedAt.Format("Jan 2, 15:04") }</small>
                    </div>
                }
            }
        </div>
    </div>
}
```

## Dashboard Customization

The admin index page can be fully customized to show relevant information and quick actions.

### Creating the Dashboard Page

Create a simple dashboard page template. See the [rendering views guide](../guides/rendering-views.md) for more information on templ syntax:

```templ
templ AdminIndexPage(api sdkapi.IPluginApi, data interface{}) {
    <h1>Admin Dashboard</h1>
}
```

## CSS Customization

The admin interface uses Bootstrap v5. The login and portal pages use Bootstrap v3 for compatibility with older captive portal browsers. Both interfaces use ES5 syntax only for maximum browser compatibility.

!!! important "Theme plugins must bundle their own Bootstrap"
    When creating a theme plugin, you must include your own copy of the required Bootstrap version in your theme's `vendor/` directory:

    - **Admin**: Bootstrap v5 CSS + JS + Bootstrap Icons
    - **Login/Portal**: Bootstrap v3 CSS (JS is not required)
    
    Reference these files in your manifest and stylesheets (see [Step 9: Add Assets](index.md#step-9-add-assets) in the main guide).

### Using Bootstrap 5

The admin interface always uses Bootstrap 5. You can use Bootstrap 5 classes in your custom styles:

```css
/* resources/assets/admin/css/theme.css */
.navbar-custom {
    background: linear-gradient(45deg, #667eea 0%, #764ba2 100%);
}

.dashboard {
    padding: 2rem;
}
```

## JavaScript Enhancements

You may use ES5-compatible JavaScript, and the admin interface supports jQuery, htmx, and Alpine.js for building interactive features.

### Theme JavaScript

```javascript
// resources/assets/admin/js/theme.js
$(function() {
    // Initialize theme-specific functionality

    // Notification handling
    $('.notification-btn').on('click', function() {
        $('.notification-menu').toggleClass('show');
    });
});
```

## Best Practices

1. **Responsive Design**: Ensure your theme works on mobile devices
2. **Accessibility**: Use proper ARIA labels and semantic HTML
3. **Performance**: Minimize CSS and JS, use efficient selectors
4. **Consistency**: Follow the existing design patterns
5. **Testing**: Test on different browsers and screen sizes

## Troubleshooting

### Common Issues

- **Assets not loading**: Check manifest files and file paths
- **Layout not rendering**: Verify templ syntax and data structures
- **Navigation not showing**: Ensure proper nav data handling
- **JavaScript errors**: Check for ES5 compatibility

### Debug Tips

- Use browser developer tools to inspect elements
- Check server logs for Go errors
- Verify API calls are working correctly
- Test with different user permissions

## Using Translations in Themes

To make your theme translatable, use the translation system in your templates:

```templ
<h3>{ api.Translate("label", "System Status") }</h3>
```

Create translation files in `resources/translations/[lang]/label/` for each language you want to support.

[← Back to Main Guide](index.md) | [Portal Customization →](page3.md)