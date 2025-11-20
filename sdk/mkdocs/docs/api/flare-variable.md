# The `$flare` global variable

The `$flare` variable is a global variable in the browser that contains helper functions to work with the Flare Hotspot API.



## 1. $flare.events {#flare-events}

The `$flare.events` object provides methods to listen to server-sent events (SSE) emitted by the Flare Hotspot backend.

### $flare.events.on {#flare-events-on}

Registers an event listener for server-sent events.

```js
// Listen for session events
var listener = $flare.events.on("session:connected", function (event) {
    console.log("Session connected:", event.data);
});

// Listen for client device events
$flare.events.on("client:connected", function (event) {
    console.log("Client connected:", event.data);
});

// Listen for notifications
$flare.events.on("flare_notification", function (event) {
    console.log("New notification:", event.data);
});

// Listen for plugin installation events (admin only)
$flare.events.on("install:progress", function (event) {
    console.log("Installation progress:", event.data);
});
```

**Parameters:**
- `event` (string): The event name to listen for
- `callback` (function): The callback function that receives the event data

**Available Events:**

| Event | Description |
|-------|-------------|
| `session:connected` | Emitted when a client session is connected |
| `session:disconnected` | Emitted when a client session is disconnected |
| `session:expired` | Emitted when a client session expires |
| `session:updated` | Emitted when session data is updated |
| `client:created` | Emitted when a new client device is created |
| `client:updated` | Emitted when client device data is updated |
| `client:connected` | Emitted when a client device connects |
| `client:disconnected` | Emitted when a client device disconnects |
| `flare_notification` | Emitted when a notification is sent to the user |
| `install:progress` | Emitted during plugin installation (admin interface only) |

### $flare.events.off {#flare-events-off}

Unregisters an event listener.

```js
// Remove a specific listener
$flare.events.off("session:connected", listener);

// Remove all listeners for an event
$flare.events.off("session:connected");
```

**Parameters:**
- `event` (string): The event name
- `callback` (function, optional): The specific callback to remove. If omitted, all listeners for the event are removed.

### $flare.events.ready {#flare-events-ready}

Registers a callback to be executed when the SSE connection is established.

```js
$flare.events.ready(function() {
    console.log("SSE connection established");
    // Now safe to register event listeners
});
```

**Parameters:**
- `callback` (function): The callback function to execute when ready

See the user account events in the [AccountsApi](./accounts-api.md#events) documentation.

See the client device events in the [ClientDevice](./client-device.md#events) documentation.

## 2. $flare.notify {#flare-notify}

The `$flare.notify` object provides methods to display toast notifications in the browser. The implementation differs between admin and portal interfaces:

- **Admin Interface**: Uses Awesome Notifications library
- **Portal Interface**: Uses Toastr library

Both implementations provide the same core methods.

### $flare.notify.success {#flare-notify-success}

Displays a success notification.

```js
$flare.notify.success('Payment processed successfully!');
$flare.notify.success('Plugin installed successfully!');
```

### $flare.notify.info {#flare-notify-info}

Displays an informational notification.

```js
$flare.notify.info('Session will expire in 5 minutes.');
$flare.notify.info('New update available.');
```

### $flare.notify.warning {#flare-notify-warning}

Displays a warning notification.

```js
$flare.notify.warning('Session disconnected due to inactivity.');
$flare.notify.warning('Low disk space detected.');
```

### $flare.notify.error {#flare-notify-error}

Displays an error notification.

```js
$flare.notify.error('Failed to save settings.');
$flare.notify.error('Network connection lost.');
```

### Admin Interface Additional Methods

The admin interface provides additional notification methods:

```js
// Alternative warning method
$flare.notify.warn('This is a warning message.');

// Failed notification (alias for error)
$flare.notify.failed('Operation failed.');
```

### Automatic Flash Message Handling

Flash messages set by the server are automatically displayed using the appropriate notification type:

```html
<!-- Server sets flash message -->
<div id="flash-message"
     data-flash-type="success"
     data-flash-message="Settings saved successfully">
</div>
```

The flash message will be automatically displayed as a success notification when the page loads.
