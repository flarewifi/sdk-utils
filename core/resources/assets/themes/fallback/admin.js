/* Core fallback admin theme JS */
(function() {
  window.$flare = window.$flare || {};
  window.$flare.notify = {
    _show: function(msg, type) {
      var container = document.getElementById('core-flash-container');
      if (!container) {
        container = document.createElement('div');
        container.id = 'core-flash-container';
        container.style.cssText = 'position:fixed;top:70px;right:20px;z-index:9999;max-width:400px;';
        document.body.appendChild(container);
      }
      var alertClass = 'alert-info';
      if (type === 'warning') alertClass = 'alert-warning';
      else if (type === 'success') alertClass = 'alert-success';
      else if (type === 'error' || type === 'failed') alertClass = 'alert-danger';
      var div = document.createElement('div');
      div.className = 'alert ' + alertClass + ' alert-dismissible fade show';
      div.setAttribute('role', 'alert');
      div.innerHTML = msg + '<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>';
      container.appendChild(div);
      setTimeout(function() {
        if (div.parentNode) div.parentNode.removeChild(div);
      }, 6000);
    },
    info: function(msg) { this._show(msg, 'info'); },
    warning: function(msg) { this._show(msg, 'warning'); },
    warn: function(msg) { this._show(msg, 'warning'); },
    success: function(msg) { this._show(msg, 'success'); },
    error: function(msg) { this._show(msg, 'error'); },
    failed: function(msg) { this._show(msg, 'error'); }
  };
})();
