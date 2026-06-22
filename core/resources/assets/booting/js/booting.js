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
  function renderSteps(steps) {
    var list = document.getElementById('boot-steps');
    if (!list || !steps) {
      return;
    }

    var html = '';
    for (var i = 0; i < steps.length; i++) {
      var step = steps[i];
      var label = step.label;
      if (step.total) {
        label = label + ' (' + step.current + '/' + step.total + ')';
      }
      html += '<li class="boot-step boot-step-' + step.status + '">' + label + '</li>';
    }
    list.innerHTML = html;
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
        window.location.replace('/');
      } else {
        setTimeout(tick, 1000); // Check again after 1 second
      }
    });
  }

  tick();
});
