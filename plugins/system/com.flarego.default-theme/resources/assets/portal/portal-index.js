(function($, $flare) {
  'use strict';

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
