'use strict'

var htmx = require('../../lib/vendor/htmx-v1.9.12.js')
window.htmx = htmx;

htmx.createEventSource = function(url) {
  var eventSource = new EventSource(url);

  // Bug: https://bugzilla.mozilla.org/show_bug.cgi?id=833462
  window.onbeforeunload = function() {
    eventSource.close();
  };

  return eventSource;
};

require('../../lib/vendor/htmx-ext-sse-v1.19.12.js')

