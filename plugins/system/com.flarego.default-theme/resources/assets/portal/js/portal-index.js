(function($, $flare) {
  'use strict';

  var toastStyle = document.createElement('style');
  toastStyle.textContent = [
    '@media (max-width: 768px) {',
    '  #toast-container.toast-bottom-right {',
    '    top: 12px;',
    '    left: 50%;',
    '    right: auto;',
    '    transform: translateX(-50%);',
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



  // ── Seconds fade animation ───────────────────────────────────────────────────

  function splitTimeSecs(formatted) {
    var idx = formatted.lastIndexOf(' ');
    if (idx === -1) return { prefix: '', secs: formatted };
    return { prefix: formatted.substring(0, idx + 1), secs: formatted.substring(idx + 1) };
  }

  function tickTimer(timeEl, secs) {
    var formatted = formatTimeSecs(secs);
    timeEl.setAttribute('data-value', secs.toString());

    var card = timeEl.querySelector('.timer-secs-card');
    var parts = splitTimeSecs(formatted);

    if (!card) {
      // First render — build structure
      var p = splitTimeSecs(formatted);
      timeEl.innerHTML = p.prefix + '<span class="timer-secs-card">' + p.secs + '</span>';
      return;
    }

    // Fade out (CSS transition handles 400ms fade)
    card.style.opacity = '0';

    // Swap text at midpoint then fade back in
    setTimeout(function() {
      card.textContent = parts.secs;
      var first = timeEl.firstChild;
      if (first && first.nodeType === 3) {
        first.textContent = parts.prefix;
      }
      card.style.opacity = '1';
    }, 400);
  }

  // ── Banner Slideshow ─────────────────────────────────────────────────────────
  var _slideshowTimer = null;
  var _slideshowObserver = null;

  function startSlideshow() {
    var slides = document.querySelectorAll('.banner-slide');
    if (!slides || slides.length === 0) return;

    if (_slideshowTimer) clearInterval(_slideshowTimer);

    var current = 0;

    function showSlide(idx) {
      for (var i = 0; i < slides.length; i++) {
        slides[i].style.opacity = '0';
      }
      slides[idx].style.opacity = '1';
    }

    showSlide(current);

    _slideshowTimer = setInterval(function() {
      current = (current + 1) % slides.length;
      showSlide(current);
    }, 3000);
  }

  function waitForSlides() {
    var slides = document.querySelectorAll('.banner-slide');
    if (slides && slides.length > 0) {
      startSlideshow();
      return;
    }
    if (_slideshowObserver) _slideshowObserver.disconnect();
    _slideshowObserver = new MutationObserver(function() {
      var found = document.querySelectorAll('.banner-slide');
      if (found && found.length > 0) {
        _slideshowObserver.disconnect();
        _slideshowObserver = null;
        startSlideshow();
      }
    });
    _slideshowObserver.observe(document.body, { childList: true, subtree: true });
  }

  // ── Timer state — tracked in JS, not the DOM ─────────────────────────────────
  var _timerSecs = 0;
  var _timerRunning = false;

  function seedTimerFromDOM() {
    var timeEl = document.getElementById('session-time');
    if (!timeEl) return;
    _timerSecs = parseInt(timeEl.getAttribute('data-value'), 10) || 0;
    // Build card span so tickTimer never hits the "first render" branch mid-tick
    var card = timeEl.querySelector('.timer-secs-card');
    if (!card) {
      var parts = splitTimeSecs(formatTimeSecs(_timerSecs));
      timeEl.innerHTML = parts.prefix + '<span class="timer-secs-card">' + parts.secs + '</span>';
    }
  }

  function seedRunningFromDOM() {
    var scriptEl = document.getElementById('portal-data');
    if (!scriptEl) return;
    var data = JSON.parse(scriptEl.textContent || scriptEl.text || '{}');
    _timerRunning = !!data.session_running;
  }

  $(document).ready(function() {

    seedTimerFromDOM();
    seedRunningFromDOM();
    waitForSlides();

    document.body.addEventListener('htmx:afterSettle', function() {
      // Reseed from the freshly swapped-in DOM values
      seedTimerFromDOM();
      seedRunningFromDOM();
      startSlideshow();
    });

    setInterval(function() {
      if (!_timerRunning) return;
      if (_timerSecs > 0) _timerSecs -= 1;
      var timeEl = document.getElementById('session-time');
      if (!timeEl) return;
      tickTimer(timeEl, _timerSecs);
    }, 1000);

  });
})(window.$, window.$flare);
