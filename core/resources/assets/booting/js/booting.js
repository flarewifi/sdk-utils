/**
 * Copyright 2021-2022 Flarego Technologies Corp. <business@flarego.ph>
 * @file             : booting.js
 * @author           : Adones Pitogo <pitogo.adones@gmail.com>
 * Date              : Nov 29, 2022
 * Last Modified Date: Feb 27, 2024
 */

/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */
window.addEventListener('load', function () {
  // Boot endpoints come from body data attributes (no hardcoded URLs).
  var statusURL = document.body.getAttribute('data-status-url');
  var progressURL = document.body.getAttribute('data-progress-url');
  var redirectURL = document.body.getAttribute('data-redirect-url') || '/';

  function checkStatus(callback) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', statusURL, true);

    xhr.onreadystatechange = function () {
      if (xhr.readyState === 4) {
        callback(xhr.status);
      }
    };

    xhr.send();
  }

  // Render the boot timeline returned by /boot/progress. The leading status tag
  // ([  OK  ] / [ .... ]) is supplied by CSS ::before keyed on the boot-step-*
  // class, so here we emit only the label. Counted steps (package install) append
  // an "(n/m)" suffix formatted here so the translated label stays number-free.
  // Plugin names rendered as child lines are plugin-supplied (incl. third-party
  // store plugins), so escape before injecting into innerHTML.
  function escapeHtml(text) {
    return String(text)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function renderSteps(steps) {
    var list = document.getElementById('boot-steps');
    if (!list || !steps) {
      return;
    }

    var html = '';
    for (var i = 0; i < steps.length; i++) {
      var step = steps[i];
      var label = escapeHtml(step.label);
      if (step.total) {
        label = label + ' (' + step.current + '/' + step.total + ')';
      }
      // Child lines (e.g. each plugin under "Loading plugins") are indented after
      // the status tag. The tag is a fixed-width ::before column and .boot-step uses
      // white-space: pre, so leading spaces here keep the tags aligned while nesting
      // the label.
      var cls = 'boot-step boot-step-' + step.status;
      if (step.indent) {
        cls += ' boot-step-sub';
        label = '  ' + label;
      }
      html += '<li class="' + cls + '">' + label + '</li>';
    }
    list.innerHTML = html;
    list.scrollTop = list.scrollHeight;
  }

  function pollProgress() {
    if (!progressURL) {
      return;
    }

    var xhr = new XMLHttpRequest();
    xhr.open('GET', progressURL, true);

    xhr.onreadystatechange = function () {
      // Ignore failures: once the app router takes over, this endpoint is gone and
      // the status check below redirects home.
      if (xhr.readyState === 4 && xhr.status === 200) {
        try {
          var data = JSON.parse(xhr.responseText);
          renderSteps(data.steps);
        } catch (e) {}
      }
    };

    xhr.send();
  }

  function tick() {
    pollProgress();
    checkStatus(function (status) {
      if (status === 200) {
        // replace(), not href=: automatic post-boot redirect (no user gesture),
        // so href= would push a history entry and trip Chrome's history
        // intervention; replace() also keeps the back button off the booting page.
        window.location.replace(redirectURL);
      } else {
        setTimeout(tick, 1000); // Check again after 1 second
      }
    });
  }

  tick();
});
