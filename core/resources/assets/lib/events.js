'use strict';

window.$flare = window.$flare || {};

var readyCallbacks = [];
var isConnected = false;

var events = {
  on: function (event, callback) {
    if (window.sse) {
      window.sse.addEventListener(event, callback);
    } else {
      console.warn(
        'SSE not initialized. Make sure you have an active EventSource connection.'
      );
    }
  },
  off: function (event, callback) {
    if (window.sse) {
      window.sse.removeEventListener(event, callback);
    } else {
      console.warn(
        'SSE not initialized. Make sure you have an active EventSource connection'
      );
    }
  },
  ready: function (callback) {
    readyCallbacks.push(callback);
  }
};

window.htmx.createEventSource = function (url) {
  var eventSource = new EventSource(url);

  eventSource.addEventListener('open', function () {
    if (isConnected) {
      window.location.href = '/';
    }
    isConnected = true;

    for (var i = 0; i < readyCallbacks.length; i++) {
      readyCallbacks[i]();
    }
    readyCallbacks = [];
  });

  // Bug: https://bugzilla.mozilla.org/show_bug.cgi?id=833462
  window.onbeforeunload = function () {
    eventSource.close();
  };

  window.sse = eventSource;
  return eventSource;
};

window.$flare.events = events;
module.exports = events;
