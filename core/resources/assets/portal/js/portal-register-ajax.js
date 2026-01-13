'use strict';

/**
 * Portal AJAX Registration
 * Handles device registration via AJAX with localStorage token management
 * Falls back to cookie-based registration on ANY error
 */

(function() {
  /**
   * Redirect to a URL (with error handling)
   * @param {string} url
   */
  function safeRedirect(url) {
    try {
      window.location.href = url;
    } catch (e) {
      console.error('[PortalRegister] Redirect failed:', e);
      // Last resort: try direct assignment
      try {
        window.location = url;
      } catch (e2) {
        console.error('[PortalRegister] Direct redirect failed:', e2);
      }
    }
  }

  /**
   * Get data attribute from loading container
   * @param {string} attr
   * @returns {string|null}
   */
  function getDataAttribute(attr) {
    try {
      var container = document.getElementById('loading-container');
      if (!container) {
        console.error('[PortalRegister] Loading container not found');
        return null;
      }
      return container.getAttribute(attr);
    } catch (e) {
      console.error('[PortalRegister] Error getting data attribute:', e);
      return null;
    }
  }

  /**
   * Perform AJAX registration
   * @param {string} registerUrl
   * @param {string} redirectUrl
   * @param {string} fallbackUrl
   * @param {boolean} isRetry - Whether this is a retry after token validation failure
   */
  function performRegistration(registerUrl, redirectUrl, fallbackUrl, isRetry) {
    try {
      // Prepare request data
      var requestData = {};
      var hasToken = false;
      
      // Check if device token exists and include it in request
      try {
        hasToken = window.$flare.deviceToken.has();
        if (hasToken) {
          var token = window.$flare.deviceToken.get();
          if (token) {
            requestData.device_token = token;
            console.log('[PortalRegister] Sending request with existing token for validation');
          }
        } else {
          console.log('[PortalRegister] No token found, registering new device');
        }
      } catch (e) {
        console.error('[PortalRegister] Error checking device token:', e);
        // Continue without token
      }

      // Collect fingerprint data
      try {
        if (window.$flare && typeof window.$flare.collectFingerprint === 'function') {
          var fpData = window.$flare.collectFingerprint();
          if (fpData) {
            requestData.user_agent = fpData.user_agent;
            requestData.screen_res = fpData.screen_res;
            requestData.language = fpData.language;
            requestData.timezone = fpData.timezone;
            console.log('[PortalRegister] Including fingerprint data');
          }
        }
      } catch (e) {
        console.warn('[PortalRegister] Fingerprint collection failed, continuing without:', e);
        // Continue without fingerprint - graceful degradation
      }

      $.ajax({
        url: registerUrl,
        type: 'POST',
        data: JSON.stringify(requestData),
        contentType: 'application/json',
        dataType: 'json',
        timeout: 10000, // 10 second timeout
        success: function(response) {
          try {
            // Validate response
            if (!response || typeof response !== 'object') {
              console.error('[PortalRegister] Invalid response format');
              safeRedirect(fallbackUrl);
              return;
            }

            // Handle token validation errors
            if (!response.success) {
              var errorType = response.error || 'unknown_error';
              console.error('[PortalRegister] Registration failed:', errorType);

              // Check if this is a token validation error
              if ((errorType === 'invalid_token' || errorType === 'expired_token') && !isRetry) {
                console.log('[PortalRegister] Token validation failed, clearing localStorage and retrying');
                
                // Clear invalid token
                try {
                  window.$flare.deviceToken.clear();
                } catch (e) {
                  console.error('[PortalRegister] Error clearing token:', e);
                }

                // Retry registration without token
                performRegistration(registerUrl, redirectUrl, fallbackUrl, true);
                return;
              }

              // Other errors or retry failed - redirect to fallback
              safeRedirect(fallbackUrl);
              return;
            }

            // Validate device_token
            if (!response.device_token || typeof response.device_token !== 'string') {
              console.error('[PortalRegister] Invalid device_token in response');
              safeRedirect(fallbackUrl);
              return;
            }

            // Store/update token
            var stored = window.$flare.deviceToken.set(response.device_token);
            if (!stored) {
              console.error('[PortalRegister] Failed to store device token');
              safeRedirect(fallbackUrl);
              return;
            }

            // Log success with details
            if (response.updated) {
              console.log('[PortalRegister] Device updated successfully, device_id:', response.device_id);
            } else if (hasToken) {
              console.log('[PortalRegister] Token validated successfully, device_id:', response.device_id);
            } else {
              console.log('[PortalRegister] New device registered successfully, device_id:', response.device_id);
            }

            // Use redirect_url from response if available, otherwise use default
            var targetUrl = response.redirect_url || redirectUrl;
            safeRedirect(targetUrl);

          } catch (e) {
            console.error('[PortalRegister] Error processing response:', e);
            safeRedirect(fallbackUrl);
          }
        },
        error: function(xhr, status, error) {
          console.error('[PortalRegister] AJAX error:', status, error);
          safeRedirect(fallbackUrl);
        }
      });
    } catch (e) {
      console.error('[PortalRegister] Error performing AJAX request:', e);
      safeRedirect(fallbackUrl);
    }
  }

  /**
   * Main registration flow
   */
  function init() {
    try {
      // Get URLs from data attributes
      var redirectUrl = getDataAttribute('data-redirect-url');
      var registerUrl = getDataAttribute('data-register-url');
      var fallbackUrl = getDataAttribute('data-fallback-url');

      // Validate URLs
      if (!redirectUrl || !registerUrl || !fallbackUrl) {
        console.error('[PortalRegister] Missing required URLs');
        // If we have fallback URL, use it; otherwise try /portal/register
        safeRedirect(fallbackUrl || '/portal/register');
        return;
      }

      // Check prerequisites
      if (typeof $ === 'undefined' || typeof $.ajax === 'undefined') {
        console.error('[PortalRegister] jQuery not available');
        safeRedirect(fallbackUrl);
        return;
      }

      if (typeof window.$flare === 'undefined' || typeof window.$flare.deviceToken === 'undefined') {
        console.error('[PortalRegister] deviceToken manager not available');
        safeRedirect(fallbackUrl);
        return;
      }

      if (!window.$flare.deviceToken.isAvailable()) {
        console.error('[PortalRegister] localStorage not available');
        safeRedirect(fallbackUrl);
        return;
      }

      // Check if storage key is available (injected by backend)
      var storageKeyAttr = getDataAttribute('data-storage-key');
      if (!storageKeyAttr) {
        console.error('[PortalRegister] Storage key not found in container, falling back to non-AJAX registration');
        safeRedirect(fallbackUrl);
        return;
      }

      // Always perform AJAX registration (validates existing token or registers new device)
      console.log('[PortalRegister] Starting registration flow');
      performRegistration(registerUrl, redirectUrl, fallbackUrl, false);

    } catch (e) {
      console.error('[PortalRegister] Initialization error:', e);
      // Try to get fallback URL, or use default
      var fallback = getDataAttribute('data-fallback-url') || '/portal/register';
      safeRedirect(fallback);
    }
  }

  // Execute when DOM is ready
  if (typeof $ !== 'undefined' && typeof $.ready !== 'undefined') {
    $(document).ready(init);
  } else {
    // Fallback: execute on window load
    if (window.addEventListener) {
      window.addEventListener('load', init);
    } else if (window.attachEvent) {
      window.attachEvent('onload', init);
    } else {
      window.onload = init;
    }
  }
})();
