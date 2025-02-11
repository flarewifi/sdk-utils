'use strict';

window.$flare = window.$flare || {};

var toastr = require('../../lib/vendor/toastr.js');
toastr.options = {
  "closeButton": false,
  "debug": false,
  "newestOnTop": false,
  "progressBar": false,
  "positionClass": "toast-bottom-right",
  "preventDuplicates": true,
  "showDuration": 300,
  "hideDuration": 1000,
  "timeOut": 5000,
  "extendedTimeOut": 1000,
  "showEasing": "swing",
  "hideEasing": "linear",
  "showMethod": "fadeIn",
  "hideMethod": "fadeOut"
};

window.toastr = toastr;

window.$flare.notify = {
  info: function(message) {
    toastr.info(message, "Info");
  },
  success: function(message) {
    toastr.success(message, "Success");
  },
  warning: function (message) {
    toastr.warning(message, "Warning");
  },
  error: function(message) {
    toastr.error(message, "Error");
  }
};

module.exports = window.$flare.notify;
