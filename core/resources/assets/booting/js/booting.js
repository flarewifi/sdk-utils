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
  function checkStatus(callback) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/', true);

    xhr.onreadystatechange = function () {
      if (xhr.readyState === 4) {
        callback(xhr.status);
      }
    };

    xhr.send();
  }

  function redirectHome() {
    checkStatus(function (status) {
      if (status === 200) {
        window.location.href = '/';
      } else {
        setTimeout(redirectHome, 1000); // Check again after 1 second
      }
    });
  }

  function callBackHook(data) {
    var logsEl = '<ul>';
    for (var i = 0; i < data.logs.length; i++) {
      logsEl += '<li>' + data.logs[i] + '</li>';
    }
    logsEl += '</ul>';
    document.getElementById('status-text').innerHTML = logsEl;
    sessionStorage.setItem("savedLogs", logsEl);
  }

  var evt = new EventSource('/boot/status');
  evt.addEventListener('boot:progress', function (res) {
    var data = JSON.parse(res.data);
    console.log(data);
    callBackHook(data);
    if (data.done) {
      redirectHome();
    }
  });

  evt.onerror = function (res) {
    console.error(res);
    setTimeout(redirectHome, 1000);
  };
});

window.onload = () => {
  const savedLogs = sessionStorage.getItem("savedLogs");
  if (savedLogs) {
    document.getElementById('status-text').innerHTML = savedLogs;
  }
};
