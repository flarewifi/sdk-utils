'use strict';

/**
 * Device Token Manager
 * Manages device authentication tokens in localStorage
 * Automatically adds X-Device-Token header to all AJAX requests
 */

(function() {
  window.$flare = window.$flare || {};

  // Get storage key from loading container data attribute (injected by backend)
  // This key is synchronized with the cookie name (e.g., "flare_device_token_e5f6")
  // Falls back to base key if not found
  var STORAGE_KEY = (function() {
    try {
      var container = document.getElementById('loading-container');
      if (container) {
        var key = container.getAttribute('data-storage-key');
        if (key) {
          return key;
        }
      }
    } catch (e) {
      console.warn('[DeviceToken] Could not read storage key from container:', e);
    }
    // Fallback to base key
    return 'flare_device_token';
  })();

  /**
   * Check if localStorage is available
   * @returns {boolean}
   */
  function isLocalStorageAvailable() {
    try {
      var test = '__localStorage_test__';
      localStorage.setItem(test, test);
      localStorage.removeItem(test);
      return true;
    } catch (e) {
      return false;
    }
  }

  /**
   * Get device token from localStorage
   * @returns {string|null}
   */
  function getToken() {
    try {
      if (!isLocalStorageAvailable()) {
        console.warn('[DeviceToken] localStorage not available');
        return null;
      }
      return localStorage.getItem(STORAGE_KEY);
    } catch (e) {
      console.error('[DeviceToken] Error getting token:', e);
      return null;
    }
  }

  /**
   * Save device token to localStorage
   * @param {string} token
   * @returns {boolean} Success status
   */
  function setToken(token) {
    try {
      if (!isLocalStorageAvailable()) {
        console.warn('[DeviceToken] localStorage not available');
        return false;
      }
      if (!token || typeof token !== 'string') {
        console.error('[DeviceToken] Invalid token provided');
        return false;
      }
      localStorage.setItem(STORAGE_KEY, token);
      return true;
    } catch (e) {
      console.error('[DeviceToken] Error setting token:', e);
      return false;
    }
  }

  /**
   * Remove device token from localStorage
   * @returns {boolean} Success status
   */
  function clearToken() {
    try {
      if (!isLocalStorageAvailable()) {
        console.warn('[DeviceToken] localStorage not available');
        return false;
      }
      localStorage.removeItem(STORAGE_KEY);
      return true;
    } catch (e) {
      console.error('[DeviceToken] Error clearing token:', e);
      return false;
    }
  }

  /**
   * Check if device token exists
   * @returns {boolean}
   */
  function hasToken() {
    try {
      var token = getToken();
      return token !== null && token !== '';
    } catch (e) {
      console.error('[DeviceToken] Error checking token:', e);
      return false;
    }
  }

  /**
   * Setup automatic X-Device-Token header for all AJAX requests
   */
  function setupAjaxHeaders() {
    try {
      // Check if jQuery is available
      if (typeof $ === 'undefined' || typeof $.ajaxSetup === 'undefined') {
        console.warn('[DeviceToken] jQuery not available, skipping AJAX setup');
        return;
      }

      $.ajaxSetup({
        beforeSend: function(xhr) {
          try {
            var token = getToken();
            if (token) {
              xhr.setRequestHeader('X-Device-Token', token);
            }
          } catch (e) {
            console.error('[DeviceToken] Error setting AJAX header:', e);
          }
        }
      });

      console.log('[DeviceToken] AJAX headers configured');
    } catch (e) {
      console.error('[DeviceToken] Error setting up AJAX headers:', e);
    }
  }

  // Initialize AJAX headers when jQuery is ready
  if (typeof $ !== 'undefined' && typeof $.ajaxSetup !== 'undefined') {
    setupAjaxHeaders();
  } else {
    // Wait for jQuery to be available
    var checkJQuery = setInterval(function() {
      if (typeof $ !== 'undefined' && typeof $.ajaxSetup !== 'undefined') {
        clearInterval(checkJQuery);
        setupAjaxHeaders();
      }
    }, 50);

    // Stop checking after 5 seconds
    setTimeout(function() {
      clearInterval(checkJQuery);
    }, 5000);
  }

  // Export public API
  window.$flare.deviceToken = {
    get: getToken,
    set: setToken,
    clear: clearToken,
    has: hasToken,
    isAvailable: isLocalStorageAvailable
  };

  module.exports = window.$flare.deviceToken;
})();
