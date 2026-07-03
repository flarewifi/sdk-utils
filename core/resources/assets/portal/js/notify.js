'use strict';

window.$flare = window.$flare || {};

// Use the same notification library as the admin surface (awesome-notifications)
// so portal/login toasts look and behave identically. The public
// window.$flare.notify API is unchanged, so callers and the server-driven
// #flash-message flash contract keep working as-is.
import AwesomeNotifications from '../../lib/vendor/awesome-notifications-v3.1.3.js';

var notifier = new AwesomeNotifications({
  icons: {
    enabled: false
  },
  durations: {
    global: 3000
  }
});

window.$flare.notify = {
  info: function (message) {
    notifier.info(message);
  },
  warn: function (message) {
    notifier.warning(message);
  },
  warning: function (message) {
    notifier.warning(message);
  },
  success: function (message) {
    notifier.success(message);
  },
  error: function (message) {
    notifier.alert(message);
  },
  failed: function (message) {
    notifier.alert(message);
  }
};

module.exports = window.$flare.notify;
