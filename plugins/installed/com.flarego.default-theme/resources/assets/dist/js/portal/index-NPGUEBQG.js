(function() {
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __commonJS = function(cb, mod) {
    return function __require() {
      return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
    };
  };

  // shared/plugins/system/com.flarego.default-theme/resources/assets/portal/portal-index.js
  var require_portal_index = __commonJS({
    "shared/plugins/system/com.flarego.default-theme/resources/assets/portal/portal-index.js": function() {
      (function($, $flare) {
        "use strict";
        $(document).ready(function() {
          setInterval(function() {
            var scriptEl = $("#portal-data");
            var data = JSON.parse(scriptEl.text());
            var running = data.session_running;
            if (running) {
              var timeEl = $("#session-time");
              var secs = timeEl.data("value") * 1 - 1;
              timeEl.data("value", secs.toString());
              timeEl.text(secs.toString());
            }
          }, 1e3);
        });
      })(window.$, window.$flare);
    }
  });

  // shared/plugins/system/com.flarego.default-theme/resources/assets/dist/js/portal/index_index.js
  require_portal_index();
})();
//# sourceMappingURL=index-NPGUEBQG.js.map
