'use strict'

window.$flare = window.$flare || {};

// Alpine is not a commonjs module
require('../../lib/vendor/alpinejs-v3.14.3.js');

var htmx = require('../../lib/vendor/htmx-v1.9.12.js')
window.htmx = htmx;

require('../../lib/events.js')
require('../../lib/vendor/htmx-ext-sse-v1.19.12.js')
require('./notify.js')
require('./flash.js')

