'use strict'

var events = require('@flarehotspot/lib/events')
var notify = require('./notify.js')

events.ready(function() {
  events.on('session:connected', function(e) {
    notify.success(e.data);
  });
  events.on('session:disconnected', function(e) {
    notify.warning(e.data);
  });
});
