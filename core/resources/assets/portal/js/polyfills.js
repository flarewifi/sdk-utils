'use strict';

import { SSE } from '@flarehotspot/lib/vendor/event-source-polyfill';
window.EventSource = window.EventSource || SSE;
