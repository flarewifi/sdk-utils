'use strict';

window.$flare = window.$flare || {};

import Alpine from 'alpinejs'
Alpine.start()
window.Alpine = Alpine

var htmx = require('htmx.org');
window.htmx = htmx;

require('../../lib/events.js');
require('../../lib/vendor/htmx-ext-loading-states-v1.19.12.js');
require('../../lib/vendor/htmx-ext-sse-v1.19.12.js');
require('./notify.js');
require('./flash.js');

