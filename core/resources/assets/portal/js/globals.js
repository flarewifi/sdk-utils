'use strict';

window.$flare = window.$flare || {};

var jQuery = require('../../lib/vendor/jquery-v1.12.4.js')
window.$ = jQuery;
window.jQuery = jQuery;

var htmx = require('../../lib/vendor/htmx-v1.9.12.js');
window.htmx = htmx;

// Alpine.js for the portal. The portal asset bundle targets ES5 (for old
// captive-portal webviews that predate Proxy, e.g. Android 4.x WebView), so we
// use Alpine v2's IE11 build: it ships a defineProperty-based Proxy polyfill and
// is already ES5, so esbuild bundles it cleanly at the ES5 target (Alpine v3's
// Proxy reactivity can't be polyfilled). This is a side-effect require: the
// build self-assigns window.Alpine and auto-starts, deferring its DOM scan to
// DOMContentLoaded — so x-data factory functions defined by later end-of-body
// theme/page scripts are available before Alpine initializes. Admin stays on
// Alpine v3 (its bundle targets ES2017). NOTE: the v2 Proxy polyfill cannot add
// reactive properties after creation — declare all x-data properties up-front.
require('../../lib/vendor/alpinejs-ie11-v2.8.2.js');

require('../../lib/events.js');
require('../../lib/vendor/htmx-ext-sse-v1.19.12.js');
require('./notify.js');
require('./sessions.js');
require('./flash.js');
