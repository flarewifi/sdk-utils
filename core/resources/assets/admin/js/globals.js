'use strict';

window.$flare = window.$flare || {};

var $ = require('../../lib/vendor/jquery-v1.12.4.js');
window.$ = $;

var htmx = require('../../lib/vendor/htmx-v1.9.12.js');
window.htmx = htmx;

require('../../lib/events.js');
require('../../lib/vendor/htmx-ext-loading-states-v1.19.12.js');
require('../../lib/vendor/htmx-ext-sse-v1.19.12.js');
require('./notify.js');
require('./flash.js');

import Alpine from 'alpinejs';
window.Alpine = Alpine;

$(document).ready(function () {
  window.Alpine.start();
});
