(function($, $flare) {
  'use strict';

  $(document).ready(function() {

    setInterval(function() {
      var scriptEl = $('#portal-data');
      var data = JSON.parse(scriptEl.text());
      var running = data.session_running;
      if (running) {
        var timeEl = $('#session-time');
        var secs = (timeEl.data('value') * 1) - 1;
        timeEl.data("value", secs.toString());
        timeEl.text(secs.toString());
      }
    }, 1000);


  });
})(window.$, window.$flare);
