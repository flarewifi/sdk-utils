# The `$flare` global variable

The `$flare` variable is a global variable in the browser that contains helper functions to work with the Flarewifi API.

## 1. $flare.events {#flare-events}

The `$flare.events` object provides methods to listen to server-sent events (SSE) emitted by the Flarewifi backend.

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

## 3. $flare.ui {#flare-ui}

The `$flare.ui` namespace contains UI components available in admin views.

### $flare.ui.line_chart {#flare-ui-line-chart}

A lightweight (~4KB) SVG stacked area chart utility. No dependencies, ES5 compatible.

#### Availability

`$flare.ui.line_chart` is globally available in all admin views.

#### Basic Usage

```html
<div id="my-chart" class="fw-chart-container"></div>
```

```javascript
var container = document.getElementById('my-chart');

$flare.ui.line_chart.create(container, {
  data: [
    { label: 'Mon', values: { uploads: 120, downloads: 80 } },
    { label: 'Tue', values: { uploads: 150, downloads: 95 } },
    { label: 'Wed', values: { uploads: 180, downloads: 110 } }
  ],
  series: [
    { key: 'downloads', label: 'Downloads', color: '#3b82f6' },
    { key: 'uploads', label: 'Uploads', color: '#10b981' }
  ]
});
```

#### API

##### `$flare.ui.line_chart.create(container, options)`

Creates a new chart instance.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `container` | `HTMLElement` | The DOM element to render the chart into |
| `options` | `Object` | Chart configuration options |

**Returns:** `{ render: Function }` - Object with a `render()` method to manually re-render

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `data` | `Array` | `[]` | Array of data points |
| `series` | `Array` | `[]` | Array of series definitions |
| `tension` | `Number` | `0.4` | Curve smoothness (0 = straight lines, 1 = maximum smoothing) |
| `fillOpacity` | `Array` | `[0.8, 0.1]` | Gradient opacity [top, bottom] |
| `padding` | `Object` | `{ top: 10, right: 10, bottom: 30, left: 50 }` | Chart padding |
| `yAxis` | `Object` | `{}` | Y-axis configuration |
| `tooltipFormat` | `Function` | `null` | Custom tooltip formatter |

#### Data Format

Each data point should have:

```javascript
{
  label: 'Mon',           // X-axis label
  values: {               // Values for each series
    seriesKey1: 100,
    seriesKey2: 50
  }
}
```

#### Series Format

Each series should have:

```javascript
{
  key: 'downloads',       // Matches key in data.values
  label: 'Downloads',     // Display label for tooltips
  color: '#3b82f6'        // Line and fill color
}
```

#### Y-Axis Options

```javascript
yAxis: {
  min: 0,          // Minimum value (default: 0)
  max: 1000,       // Maximum value (default: auto-calculated)
  stepSize: 200    // Grid line interval (default: auto-calculated)
}
```

#### Custom Tooltip Formatter

```javascript
tooltipFormat: function(label, value) {
  return label + ': ' + value + ' MB';
}
```

#### CSS Classes

The chart uses these CSS classes which you can override:

| Class | Description |
|-------|-------------|
| `.fw-chart-container` | Container element (set height here) |
| `.fw-chart-svg` | The SVG element |
| `.fw-chart-grid` | Grid lines group |
| `.fw-chart-y-labels` | Y-axis labels |
| `.fw-chart-x-labels` | X-axis labels |
| `.fw-chart-indicator` | Vertical hover line |
| `.fw-chart-tooltip` | Tooltip container |
| `.fw-chart-tooltip-title` | Tooltip title |
| `.fw-chart-tooltip-row` | Tooltip data row |
| `.fw-chart-tooltip-dot` | Color dot in tooltip |
| `.fw-chart-hover-dot` | Dots at series intersection on hover |

#### Dark Mode

The line chart automatically supports dark mode via `[data-bs-theme="dark"]` CSS selectors.

#### Complete JavaScript Example

```javascript
var container = document.getElementById('bandwidth-chart');

$flare.ui.line_chart.create(container, {
  data: [
    { label: '00:00', values: { rx: 45, tx: 30 } },
    { label: '04:00', values: { rx: 20, tx: 15 } },
    { label: '08:00', values: { rx: 80, tx: 55 } },
    { label: '12:00', values: { rx: 150, tx: 90 } },
    { label: '16:00', values: { rx: 200, tx: 120 } },
    { label: '20:00', values: { rx: 180, tx: 100 } }
  ],
  series: [
    { key: 'rx', label: 'Download', color: '#3b82f6' },
    { key: 'tx', label: 'Upload', color: '#10b981' }
  ],
  tension: 0.4,
  fillOpacity: [0.6, 0.05],
  padding: { top: 20, right: 20, bottom: 40, left: 60 },
  yAxis: {
    min: 0,
    max: 250,
    stepSize: 50
  },
  tooltipFormat: function(label, value) {
    return label + ': ' + value + ' Mbps';
  }
});
```

#### Go API

The line chart can also be rendered using the Go SDK via `api.UI().LineChart()`. This is the recommended approach for server-rendered charts.

##### Usage in Templ

```templ
package views

import sdkapi "sdk/api"

templ BandwidthChart(api sdkapi.IPluginApi, data []sdkapi.LineChartDataPoint) {
    @api.UI().LineChart(&sdkapi.LineChartOpts{
        Data: data,
        Series: []sdkapi.LineChartSeries{
            {Key: "rx", Label: "Download", Color: "#3b82f6"},
            {Key: "tx", Label: "Upload", Color: "#10b981"},
        },
        Height: "400px",
        TooltipTemplate: "{label}: {value} Mbps",
        TooltipDecimals: intPtr(1),
    })
}

func intPtr(i int) *int { return &i }
```

##### Go Types

```go
// LineChartDataPoint represents a single data point
type LineChartDataPoint struct {
    Label  string             `json:"label"`
    Values map[string]float64 `json:"values"`
}

// LineChartSeries defines a data series
type LineChartSeries struct {
    Key   string `json:"key"`
    Label string `json:"label"`
    Color string `json:"color"`
}

// LineChartYAxis configures the Y-axis
type LineChartYAxis struct {
    Min      *float64 `json:"min,omitempty"`
    Max      *float64 `json:"max,omitempty"`
    StepSize *float64 `json:"stepSize,omitempty"`
}

// LineChartPadding configures chart padding
type LineChartPadding struct {
    Top    int `json:"top,omitempty"`
    Right  int `json:"right,omitempty"`
    Bottom int `json:"bottom,omitempty"`
    Left   int `json:"left,omitempty"`
}

// LineChartOpts configures the line chart component
type LineChartOpts struct {
    ID              string               // HTML element ID (auto-generated if empty)
    Data            []LineChartDataPoint // Chart data points
    Series          []LineChartSeries    // Series definitions
    Tension         *float64             // Curve smoothness (0-1, default 0.4)
    FillOpacity     []float64            // Gradient opacity [top, bottom]
    Padding         *LineChartPadding    // Chart padding
    YAxis           *LineChartYAxis      // Y-axis configuration
    Height          string               // Container height (default "350px")
    Class           string               // Additional CSS classes
    TooltipTemplate string               // e.g., "{label}: {value} MB"
    TooltipPrefix   string               // Prefix before value, e.g., "₱"
    TooltipDecimals *int                 // Decimal places (default: 2)
}
```

##### Complete Go Example

```go
// In a controller
func (c *DashboardController) Index(w http.ResponseWriter, r *http.Request) {
    // Prepare chart data
    chartData := []sdkapi.LineChartDataPoint{
        {Label: "Mon", Values: map[string]float64{"revenue": 1200, "expenses": 800}},
        {Label: "Tue", Values: map[string]float64{"revenue": 1500, "expenses": 950}},
        {Label: "Wed", Values: map[string]float64{"revenue": 1800, "expenses": 1100}},
        {Label: "Thu", Values: map[string]float64{"revenue": 1400, "expenses": 900}},
        {Label: "Fri", Values: map[string]float64{"revenue": 2200, "expenses": 1400}},
    }

    // Render view with chart
    api.Http().HttpResponse().AdminView(w, r, sdkapi.ViewPage{
        PageContent: views.Dashboard(api, chartData),
    })
}
```

```templ
// In views/dashboard.templ
package views

import sdkapi "sdk/api"

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int { return &i }

templ Dashboard(api sdkapi.IPluginApi, chartData []sdkapi.LineChartDataPoint) {
    <div class="card">
        <div class="card-header">
            <h5>{ api.Translate("label", "Weekly Overview") }</h5>
        </div>
        <div class="card-body">
            @api.UI().LineChart(&sdkapi.LineChartOpts{
                Data: chartData,
                Series: []sdkapi.LineChartSeries{
                    {Key: "revenue", Label: "Revenue", Color: "#10b981"},
                    {Key: "expenses", Label: "Expenses", Color: "#ef4444"},
                },
                Height: "350px",
                Tension: floatPtr(0.4),
                FillOpacity: []float64{0.6, 0.05},
                Padding: &sdkapi.LineChartPadding{Right: 30},
                YAxis: &sdkapi.LineChartYAxis{Min: floatPtr(0)},
                TooltipTemplate: "{label}: {value}",
                TooltipPrefix: "₱",
                TooltipDecimals: intPtr(2),
            })
        </div>
    </div>
}
