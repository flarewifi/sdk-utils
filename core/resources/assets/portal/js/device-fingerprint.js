/**
 * Device Fingerprint Collector (ES3 compatible)
 * Collects browser fingerprint data for device identification
 */

(function() {
  window.$flare = window.$flare || {};
  
  /**
   * Collect device fingerprint data
   * @returns {Object} Fingerprint data or null if collection fails
   */
  window.$flare.collectFingerprint = function() {
    try {
      var fingerprint = {
        user_agent: '',
        screen_res: '',
        language: '',
        timezone: ''
      };
      
      // User-Agent
      try {
        fingerprint.user_agent = navigator.userAgent || '';
      } catch (e) {
        console.warn('[Fingerprint] Failed to get user agent:', e);
      }
      
      // Screen resolution
      try {
        if (window.screen && window.screen.width && window.screen.height) {
          fingerprint.screen_res = window.screen.width + 'x' + window.screen.height;
        }
      } catch (e) {
        console.warn('[Fingerprint] Failed to get screen resolution:', e);
      }
      
      // Language
      try {
        fingerprint.language = navigator.language || 
                               navigator.userLanguage || 
                               navigator.browserLanguage || '';
      } catch (e) {
        console.warn('[Fingerprint] Failed to get language:', e);
      }
      
      // Timezone (with half-hour support)
      try {
        var offset = new Date().getTimezoneOffset();
        var hours = Math.floor(Math.abs(offset) / 60);
        var minutes = Math.abs(offset) % 60;
        fingerprint.timezone = 'UTC' + (offset <= 0 ? '+' : '-') + hours +
                              (minutes > 0 ? ':' + (minutes < 10 ? '0' : '') + minutes : '');
      } catch (e) {
        console.warn('[Fingerprint] Failed to get timezone:', e);
      }
      
      console.log('[Fingerprint] Collected:', fingerprint);
      return fingerprint;
      
    } catch (e) {
      console.error('[Fingerprint] Collection failed:', e);
      return null;
    }
  };
  
  /**
   * Check if fingerprinting is available
   * @returns {boolean}
   */
  window.$flare.canFingerprint = function() {
    return typeof navigator !== 'undefined' && 
           typeof window.screen !== 'undefined';
  };
})();
