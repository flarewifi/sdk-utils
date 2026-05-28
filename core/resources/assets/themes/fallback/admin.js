/* Core fallback admin theme JS — minimal */
(function () {
  window.$flare = window.$flare || {};

  // Flash notification helper — drops messages into the #flash container
  // defined in admin layout. User dismisses manually via the close button.
  window.$flare.notify = {
    _show: function (msg, type) {
      var container = document.getElementById('flash');
      if (!container) return;
      var div = document.createElement('div');
      div.className = 'fw-flash-item ' + (type || 'info');

      var span = document.createElement('span');
      span.className = 'fw-flash-msg';
      span.textContent = msg;
      div.appendChild(span);

      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'fw-flash-close';
      btn.setAttribute('aria-label', 'Dismiss');
      btn.textContent = '×';
      btn.onclick = function () {
        if (div.parentNode) div.parentNode.removeChild(div);
      };
      div.appendChild(btn);

      container.appendChild(div);
    },
    info: function (m) { this._show(m, 'info'); },
    warning: function (m) { this._show(m, 'warning'); },
    warn: function (m) { this._show(m, 'warning'); },
    success: function (m) { this._show(m, 'success'); },
    error: function (m) { this._show(m, 'error'); },
    failed: function (m) { this._show(m, 'error'); }
  };

  // SSE flare_notification → flash.
  // $flare.events is created by the platform's global events glue.
  if (window.$flare.events && typeof window.$flare.events.ready === 'function') {
    window.$flare.events.ready(function () {
      window.$flare.events.on('flare_notification', function (event) {
        try {
          var data = JSON.parse(event.data);
          var fn = window.$flare.notify[data.type] || window.$flare.notify.info;
          fn.call(window.$flare.notify, data.subject);
        } catch (e) {}
      });
    });
  }
})();
