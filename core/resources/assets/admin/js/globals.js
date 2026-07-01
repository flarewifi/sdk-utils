'use strict';

window.$flare = window.$flare || {};

var $ = require('../../lib/vendor/jquery-v3.7.1.js');
window.$ = $;
window.jQuery = $;

var htmx = require('../../lib/vendor/htmx-v2.0.10.js');
window.htmx = htmx;

// Bootstrap 5 JS bundle (Popper included) — provided globally by core so every
// admin page gets tabs/collapse/dropdowns/modals/tooltips regardless of the
// active theme. Themes must NOT vendor their own Bootstrap. Side-effect import:
// its data-api attaches to document, driving all data-bs-* interactions, and we
// expose window.bootstrap so themes can use the programmatic API (new bootstrap.Modal, etc.).
window.bootstrap = require('../../lib/vendor/bootstrap-bundle-v5.3.3.js');

require('../../lib/events.js');
require('../../lib/vendor/htmx-ext-loading-states-v2.0.2.js');
require('../../lib/vendor/htmx-ext-sse-v2.2.4.js');
require('./notify.js');
require('./flash.js');

import Alpine from '@flare/lib/vendor/alpinejs-v3.15.1.js';
window.Alpine = Alpine;

$(document).ready(function () {
  window.Alpine.start();
});
