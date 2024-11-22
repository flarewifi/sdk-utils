"use strict";

/*!
 * Color mode toggler for Bootstrap's docs (https://getbootstrap.com/)
 * Copyright 2011-2024 The Bootstrap Authors
 * Licensed under the Creative Commons Attribution 3.0 Unported License.
 */

(function () {
  'use strict';

  var getStoredTheme = function getStoredTheme() {
    return localStorage.getItem('theme');
  };
  var setStoredTheme = function setStoredTheme(theme) {
    return localStorage.setItem('theme', theme);
  };
  var getPreferredTheme = function getPreferredTheme() {
    var storedTheme = getStoredTheme();
    if (storedTheme) {
      return storedTheme;
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  };
  var setTheme = function setTheme(theme) {
    if (theme === 'auto') {
      document.documentElement.setAttribute('data-bs-theme', window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    } else {
      document.documentElement.setAttribute('data-bs-theme', theme);
    }
  };
  setTheme(getPreferredTheme());
  var showActiveTheme = function showActiveTheme(theme) {
    var focus = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    var themeSwitcher = document.querySelector('#bd-theme');
    if (!themeSwitcher) {
      return;
    }
    var themeSwitcherText = document.querySelector('#bd-theme-text');
    var activeThemeIcon = document.querySelector('.theme-icon-active use');
    var btnToActive = document.querySelector("[data-bs-theme-value=\"".concat(theme, "\"]"));
    var svgOfActiveBtn = btnToActive.querySelector('svg use').getAttribute('href');
    document.querySelectorAll('[data-bs-theme-value]').forEach(function (element) {
      element.classList.remove('active');
      element.setAttribute('aria-pressed', 'false');
    });
    btnToActive.classList.add('active');
    btnToActive.setAttribute('aria-pressed', 'true');
    activeThemeIcon.setAttribute('href', svgOfActiveBtn);
    var themeSwitcherLabel = "".concat(themeSwitcherText.textContent, " (").concat(btnToActive.dataset.bsThemeValue, ")");
    themeSwitcher.setAttribute('aria-label', themeSwitcherLabel);
    if (focus) {
      themeSwitcher.focus();
    }
  };
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function () {
    var storedTheme = getStoredTheme();
    if (storedTheme !== 'light' && storedTheme !== 'dark') {
      setTheme(getPreferredTheme());
    }
  });
  window.addEventListener('DOMContentLoaded', function () {
    showActiveTheme(getPreferredTheme());
    document.querySelectorAll('[data-bs-theme-value]').forEach(function (toggle) {
      toggle.addEventListener('click', function () {
        var theme = toggle.getAttribute('data-bs-theme-value');
        setStoredTheme(theme);
        setTheme(theme);
        showActiveTheme(theme, true);
      });
    });
  });
})();
