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
  // Get the boot status url from body data attribute

  function checkStatus(callback) {
    var statusUrl = document.body.getAttribute('data-status-url');
    console.log('Checking status at:', statusUrl);
    var xhr = new XMLHttpRequest();
    xhr.open('GET', statusUrl, true);

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

  redirectHome();
});
