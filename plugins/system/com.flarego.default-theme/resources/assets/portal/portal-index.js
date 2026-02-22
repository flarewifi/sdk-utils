(function($, $flare) {
  'use strict';

  var toastStyle = document.createElement('style');
  toastStyle.textContent = [
    '@media (max-width: 768px) {',
    '  #toast-container.toast-bottom-right {',
      '  top: 12px;',
      '  left: 50%;',
      '  right: auto;',
      '  transform: translateX(-50%);',
      '    width: 95%;',
      '  }',
    '  #toast-container.toast-bottom-right > .toast {',
    '    width: 100%;',
    '    max-width: 100%;',
    '    box-sizing: border-box;',
    '  }',
    '}'
  ].join('\n');
  document.head.appendChild(toastStyle);

  // incase specific message
  // var _origNotify = $flare.notify;
  // $flare.notify = {
  //   info: function(message) {
  //     _origNotify.info(message);
  //   },
  //   success: function(message) {
  //     if (message === 'Now connected to internet' || 'Success free trial') {
  //       applyToastStyle();
  //     }
  //     _origNotify.success(message);
  //   },
  //   warning: function(message) {
  //     if (message == 'Now disconnected from internet') {
  //       applyToastStyle();
  //     }
  //     _origNotify.warning(message);
  //   },
  //   error: function(message) {
  //     if (message == 'Not connected to internet' || 'Unable to generate session') {
  //       applyToastStyle();
  //     }
  //     _origNotify.error(message);
  //   }
  // };

  // Format seconds into human-readable string (mirrors Go sdkutils.FormatTimeSecs)
  // Examples: "5h 30m 0s", "2d 0h 3m 0s", "1d 0h 0m 30s", "45m 20s", "0s"
  function formatTimeSecs(timeSec) {
    var days = Math.floor(timeSec / 86400);
    timeSec = timeSec % 86400;
    var hours = Math.floor(timeSec / 3600);
    timeSec = timeSec % 3600;
    var minutes = Math.floor(timeSec / 60);
    var seconds = timeSec % 60;

    var result = '';
    var started = false;

    if (days > 0) {
      result += days + 'd ';
      started = true;
    }
    if (hours > 0 || (started && (minutes > 0 || seconds >= 0))) {
      result += hours + 'h ';
      started = true;
    }
    if (minutes > 0 || (started && seconds >= 0)) {
      result += minutes + 'm ';
      started = true;
    }
    result += seconds + 's';

    return result;
  }

  $(document).ready(function() {

    setInterval(function() {
      var scriptEl = $('#portal-data');
      var data = JSON.parse(scriptEl.text());
      var running = data.session_running;
      if (running) {
        var timeEl = $('#session-time');
        var secs = (timeEl.data('value') * 1) - 1;
        if (secs < 0) {
          secs = 0;
        }
        timeEl.data('value', secs.toString());
        timeEl.text(formatTimeSecs(secs));
      }
    }, 1000);

  });
})(window.$, window.$flare);
