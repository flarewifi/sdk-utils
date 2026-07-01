'use strict';

window.$flare = window.$flare || {};

var jQuery = require('../../lib/vendor/jquery-v3.7.1.js')
window.$ = jQuery;
window.jQuery = jQuery;

var htmx = require('../../lib/vendor/htmx-v2.0.10.js');
window.htmx = htmx;

// Bootstrap 5 JS bundle (Popper included) — provided globally by core so every
// portal page gets Bootstrap components regardless of the active theme. Themes
// must NOT vendor their own Bootstrap. Side-effect import: its data-api attaches
// to document, driving all data-bs-* interactions, and we expose window.bootstrap
// so themes can use the programmatic API (new bootstrap.Modal, etc.).
window.bootstrap = require('../../lib/vendor/bootstrap-bundle-v5.3.3.js');

// Alpine.js v3 for the portal (same version as admin). The portal bundle now
// targets ES2017 for modern browsers only — the old ES5/IE11 Alpine v2 build is
// gone. Import (hoisted by esbuild) self-assigns window.Alpine and auto-starts,
// deferring its DOM scan to DOMContentLoaded, so x-data factories defined by
// later end-of-body theme/page scripts are available before Alpine initializes.
import Alpine from '@flare/lib/vendor/alpinejs-v3.15.1.js';
window.Alpine = Alpine;

require('../../lib/events.js');
require('../../lib/vendor/htmx-ext-sse-v2.2.4.js');
require('./notify.js');
require('./sessions.js');
require('./flash.js');
