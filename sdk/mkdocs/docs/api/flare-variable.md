# The `$flare` global variable

The `$flare` variable is a global variable in the browser that contains helper functions to work with the Flare Hotspot API.

## 1. $flare.http {#flare-http}

### $flare.http.get {#flare-http-get}

The `$flare.http.get` method is used to perform a `GET` AJAX request. It accepts two arguments, the first argument is the URL to send the form data to, and the second argument is the query params.

```js
var queryParams = {amount: 100};

$flare.http.get('/path/to/handler', queryParams)
    .then(function(response){
        console.log(response);
    })
    .catch(function(error){
        console.log(error);
    });
```

### $flare.http.post {#flare-http-post}

The `$flare.http.post` method is used to perform a `POST` AJAX request. It accepts two arguments, the first argument is the URL to send the form data to, and the second argument is the form data.

```js
var formData = {amount: 100};

$flare.http.post('/path/to/handler', formData)
    .then(function(response){
        console.log(response);
    })
    .catch(function(error){
        console.log(error);
    });
```

!!!warning "Important"
    You must use [VueResponse](./vue-response.md) in the server side to perform http resposes for both the [$flare.http.get](#flare-http-get) and [$flare.http.post](#flare-http-post) methods.

## 2. $flare.vueLazyLoad #{flare-vuelazyload}

The `$flare.vueLazyLoad` method is used to lazy load vue components.

```js
var component = '<% .Helpers.VueComponentPath "sample-child.vue" %>';
var lazyComponent = $flare.vueLazyLoad(component);

var app = new Vue({
    el: '#app',
    components: {
        'sample-child': lazyComponent
    }
});
```

## 3. $flare.events {#flare-events}

The `$flare.events` is used to listen to events emitted by the server.

Below is an example of how to listen to an event:

```js
var listener = $flare.events.on("session:connected", function (data) {
    console.log("Session connected: ", data);
});
```

To unregister an event listener, use the `off` method.

```js
$flare.events.off("session:connected", listener);
```

See the user account events in the [AccountsApi](./accounts-api.md#events) documentation.

See the client device events in the [ClientDevice](./client-device.md#events) documentation.

## 4. $flare.notify {#flare-notify}

This is used to display a notification on the browser.
It has 4 methods: `success`, `info`, `warning` and `error`.

### Success

```js
$flare.notify.success('This is a success message.')
```

### Info
```js
$flare.notify.info('This is an info message.')
```

### Warning
```js
$flare.notify.warning('This is a warning message.')
```

### Error
```js
$flare.notify.error('This is an error message.')
```
