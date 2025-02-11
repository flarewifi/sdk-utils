'use strict';

window.$flare = window.$flare || {};

var jQuery = require('@flarehotspot/lib/vendor/jquery-v1.12.4.js')
window.$ = jQuery;
window.jQuery = jQuery;

var htmx = require('@flarehotspot/lib/vendor/htmx-v1.9.12.js');
window.htmx = htmx;

require('@flarehotspot/lib/events.js');
require('@flarehotspot/lib/vendor/htmx-ext-sse-v1.19.12.js');
require('./notify.js');
require('./sessions.js');
require('./flash.js');

