# INotificationAPI

The `INotificationAPI` provides methods to manage user notifications in the Flare Hotspot system. Notifications can be used to inform users about important events, system status, or other relevant information.

To get an instance of `INotificationAPI`:

```go
notificationAPI := api.Notification()
fmt.Println(notificationAPI) // INotificationAPI
```

## INotificationAPI Methods

The following methods are available in `INotificationAPI`:

### AddNotification

Adds a new notification to the system.

```go
params := AddNotificationParams{
    Subject: "Payment Successful",
    Content: "Your payment of $10.00 has been processed successfully.",
    Type:    NotificationTypeSuccess,
}

err := api.Notification().AddNotification(r.Context(), params)
if err != nil {
    // handle error
}
```

### GetUnreadNotifications

Retrieves all unread notifications for the current user.

```go
notifications, err := api.Notification().GetUnreadNotifications(r.Context())
if err != nil {
    // handle error
}

for _, notification := range notifications {
    fmt.Printf("Subject: %s, Type: %s\n", notification.Subject, notification.Type)
}
```

### GetNotificationByID

Retrieves a specific notification by its ID.

```go
notification, err := api.Notification().GetNotificationByID(r.Context(), 123)
if err != nil {
    // handle error
}

fmt.Printf("Notification: %+v\n", notification)
```

### UpdateNotificationStatus

Updates the status of a notification (mark as read/unread).

```go
err := api.Notification().UpdateNotificationStatus(r.Context(), 123, NotificationStatusRead)
if err != nil {
    // handle error
}
```

## Types

### Notification

The `Notification` struct represents a notification in the system:

```go
type Notification struct {
    ID        int64              `json:"id"`
    Subject   string             `json:"subject"`
    Content   string             `json:"content"`
    Status    NotificationStatus `json:"status"`
    Type      NotificationType   `json:"type"`
    CreatedAt time.Time          `json:"created_at"`
    UpdatedAt time.Time          `json:"updated_at"`
}
```

### NotificationStatus

The `NotificationStatus` type represents the read status of a notification:

| Value | Description |
|-------|-------------|
| `0` | `NotificationStatusUnread` - Notification has not been read |
| `1` | `NotificationStatusRead` - Notification has been read |

### NotificationType

The `NotificationType` represents the type of notification:

| Value | Description |
|-------|-------------|
| `"success"` | `NotificationTypeSuccess` - Success notification |
| `"error"` | `NotificationTypeError` - Error notification |
| `"info"` | `NotificationTypeInfo` - Information notification |
| `"warn"` | `NotificationTypeWarn` - Warning notification |

### AddNotificationParams

The `AddNotificationParams` struct is used when creating new notifications:

```go
type AddNotificationParams struct {
    Subject string           // The notification subject/title
    Content string           // The notification content/body
    Type    NotificationType // The notification type
}
```

## Events

Notifications can trigger events that can be listened to in the browser using the `$flare.events` API:

```js
$flare.events.on("flare_notification", function(event) {
    console.log("New notification:", event.data);
});
```

The event data will contain the notification details in JSON format.</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/notification.md