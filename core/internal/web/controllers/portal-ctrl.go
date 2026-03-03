package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"core/internal/api"
	devicetoken "core/internal/modules/device-token"
	machineuid "core/internal/modules/machine-uid"
	"core/internal/sessmgr"
	"core/internal/web/helpers"
	portalview "core/resources/views/portal"
	"core/utils/hostfinder"
	sse "core/utils/sse"
	sdkapi "sdk/api"
)

// PortalRootCtrl handles the root path "/"
// OPTIMIZATION: Checks for valid device cookie and performs inline registration
// If device cookie exists and validation passes, redirects directly to /portal/index (fast path)
// Otherwise, renders redirect page to /portal/redirect (normal path)
func PortalRootCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clntMgr := g.CoreAPI.ClientRegister

		// FAST PATH: Check for device cookie
		deviceID, cookieErr := devicetoken.GetDeviceCookie(r)
		if cookieErr == nil && deviceID > 0 {
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRoot: Device cookie found (ID=%d), attempting fast-path registration", deviceID))

			// Get device MAC/IP/hostname from request
			h, err := hostfinder.GetHostFromRequest(r)
			if err != nil {
				// Check if error is DHCP-related (we still have MAC/IP) or ARP failure (critical)
				if h != nil && h.MacAddr != "" {
					// Got MAC/IP but DHCP lease read failed - continue with empty hostname
					g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRoot: DHCP lookup failed but MAC/IP available - MAC: %s, IP: %s", h.MacAddr, h.IpAddr))
				} else {
					// Critical error - couldn't identify device, fall through to normal path
					g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRoot: Failed to identify device - RemoteAddr: %s, Error: %v - falling back to normal path", r.RemoteAddr, err))
					goto NORMAL_PATH
				}
			}

			// Get User-Agent from request headers
			userAgent := r.Header.Get("User-Agent")

			// Register/validate device inline (handles MAC changes, fingerprint validation)
			clnt, shouldSetCookie, err := clntMgr.Register(r.Context(), sessmgr.ClientRegisterParams{
				CookieDeviceID: &deviceID,
				MacAddr:        h.MacAddr,
				IpAddr:         h.IpAddr,
				Hostname:       h.Hostname,
				UserAgent:      userAgent,
			})

			// Check if registration succeeded
			if err != nil {
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRoot: Fast-path registration failed - DeviceID: %d, MAC: %s, Error: %v - falling back to normal path", deviceID, h.MacAddr, err))
				goto NORMAL_PATH
			}
			if clnt == nil {
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRoot: Fast-path registration returned nil client - DeviceID: %d, MAC: %s - falling back to normal path", deviceID, h.MacAddr))
				goto NORMAL_PATH
			}

			// Registration successful - set cookies and redirect
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRoot: Fast-path registration successful - DeviceID: %d, MAC: %s", clnt.ID(), clnt.MacAddr()))

			// Set device cookie if validation passed
			if shouldSetCookie {
				if err := devicetoken.SetDeviceCookie(w, clnt.ID()); err != nil {
					g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRoot: Failed to set device cookie - DeviceID: %d, Error: %v", clnt.ID(), err))
				}
			}

			// Emit WiFi connect event - portal access confirms the client is connected
			api.EmitWifiEvent(sdkapi.WifiEventClientConnected, clnt.MacAddr())

			// Always set "register" cookie with 12-hour expiration (skip check, always set)
			cookieOpts := &sdkapi.HttpCookieOpts{
				Path:    "/",
				Expires: time.Now().Add(12 * time.Hour),
			}
			g.CoreAPI.HttpAPI.Cookie().SetPlainCookie(w, "register", "1", cookieOpts)

			// Redirect directly to portal index (FAST PATH - no loading screen)
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRoot: Redirecting to portal index (fast path) - DeviceID: %d", clnt.ID()))
			g.CoreAPI.HttpAPI.Response().Redirect(w, r, "portal:index")
			return
		}

		// Log why we're taking the normal path
		if cookieErr != nil {
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRoot: No valid device cookie found - Error: %v - using normal path", cookieErr))
		} else {
			g.CoreAPI.LoggerAPI.Info("PortalRoot: Device cookie has invalid ID (0) - using normal path")
		}

	NORMAL_PATH:
		// NORMAL PATH: No cookie or fast-path failed - render redirect page
		lanIP := helpers.GetLanIP(r)

		// Get redirect path using UrlForRoute
		redirectPath := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:redirector")

		// Render portal root page with inline JavaScript
		page := portalview.PortalRootPage(g.CoreAPI, lanIP, redirectPath)

		// Render directly without ViewPage wrapper (no assets needed)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page.Render(r.Context(), w)
	}
}

// PortalRedirectCtrl handles the /portal route
// Uses AJAX-based registration with localStorage token management
// Falls back to cookie-based registration on any error
// ALWAYS triggers registration flow to handle MAC/IP changes (MAC randomization support)
func PortalRedirectCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get URLs for AJAX registration flow
		redirectUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:index")
		registerUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:register:ajax")
		fallbackUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:register")

		// Get device token key for synchronization between cookie and localStorage
		localStorageKey := devicetoken.GetDeviceTokenKey()

		// Always render AJAX registration page to trigger device validation/update
		// This ensures MAC/IP changes are detected (e.g., MAC randomization)
		// The AJAX endpoint will validate existing tokens or register new devices
		page := portalview.PortalRedirectPage(g.CoreAPI, redirectUrl, registerUrl, fallbackUrl, localStorageKey)
		v := sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				JsFile:  "portal-register.js",
				CssFile: "portal-redirect.css",
			},
			PageContent:   page,
			PreserveFlash: true, // Preserve flash for the next page (portal:index)
		}
		g.CoreAPI.HttpAPI.Response().PortalView(w, r, v)
	}
}

func PortalRegisterCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clntMgr := g.CoreAPI.ClientRegister

		// Get device ID from JWT cookie (if exists)
		var cookieDeviceID *int64
		deviceID, cookieErr := devicetoken.GetDeviceCookie(r)
		if cookieErr == nil && deviceID > 0 {
			cookieDeviceID = &deviceID
		}

		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil {
			// Check if error is DHCP-related (we still have MAC/IP) or ARP failure (critical)
			if h != nil && h.MacAddr != "" {
				// Got MAC/IP but DHCP lease read failed - continue with empty hostname
			} else {
				// Critical error - couldn't identify device at all
				userMsg := g.CoreAPI.Translate("error", "Unable to identify your device from the network")
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Failed to identify device - RemoteAddr: %s, Error: %v", r.RemoteAddr, err))
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(userMsg), http.StatusBadRequest)
				return
			}
		}

		// Get User-Agent from request headers
		userAgent := r.Header.Get("User-Agent")

		// Register/identify device with cookie validation and MAC randomization support
		clnt, shouldSetCookie, err := clntMgr.Register(r.Context(), sessmgr.ClientRegisterParams{
			CookieDeviceID: cookieDeviceID,
			MacAddr:        h.MacAddr,
			IpAddr:         h.IpAddr,
			Hostname:       h.Hostname,
			UserAgent:      userAgent,
		})
		if err != nil {
			userMsg := g.CoreAPI.Translate("error", "Failed to register your device")
			cookieIDStr := "none"
			if cookieDeviceID != nil {
				cookieIDStr = fmt.Sprintf("%d", *cookieDeviceID)
			}
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Registration failed - MAC: %s, IP: %s, CookieID: %s, Error: %v",
				h.MacAddr, h.IpAddr, cookieIDStr, err))
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(userMsg), http.StatusBadRequest)
			return
		}
		if clnt == nil {
			userMsg := g.CoreAPI.Translate("error", "Failed to register your device")
			cookieIDStr := "none"
			if cookieDeviceID != nil {
				cookieIDStr = fmt.Sprintf("%d", *cookieDeviceID)
			}
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Register returned nil client - MAC: %s, IP: %s, CookieID: %s",
				h.MacAddr, h.IpAddr, cookieIDStr))
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(userMsg), http.StatusInternalServerError)
			return
		}

		// Only set cookie if validation passed
		if shouldSetCookie {
			if err := devicetoken.SetDeviceCookie(w, clnt.ID()); err != nil {
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Failed to set device cookie - DeviceID: %d, Error: %v", clnt.ID(), err))
			}
		}

		// Emit WiFi connect event - portal access confirms the client is connected
		// This serves as redundancy for hostapd_cli events and cancels any pending auto-pause
		api.EmitWifiEvent(sdkapi.WifiEventClientConnected, clnt.MacAddr())

		// Set "register" cookie with 12-hour expiration to allow access to /portal/index (only if not already set)
		if _, err := g.CoreAPI.HttpAPI.Cookie().GetPlainCookie(r, "register"); err != nil {
			cookieOpts := &sdkapi.HttpCookieOpts{
				Path:    "/",
				Expires: time.Now().Add(12 * time.Hour),
			}
			g.CoreAPI.HttpAPI.Cookie().SetPlainCookie(w, "register", "1", cookieOpts)
		}
		g.CoreAPI.HttpAPI.Response().Redirect(w, r, "portal:index")
	}
}

// PortalRegisterAjaxRequest represents the JSON request body for AJAX registration
type PortalRegisterAjaxRequest struct {
	DeviceToken string `json:"device_token"`
	// Fingerprint fields
	UserAgent string `json:"user_agent"`
	ScreenRes string `json:"screen_res"`
	Language  string `json:"language"`
	Timezone  string `json:"timezone"`
}

// PortalRegisterAjaxCtrl handles AJAX-based device registration and token validation
// Supports two scenarios:
// 1. Token provided: Validates existing token and updates device MAC/IP if changed
// 2. No token: Registers new device and returns new token
func PortalRegisterAjaxCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Request received - RemoteAddr: %s", r.RemoteAddr))
		clntMgr := g.CoreAPI.ClientRegister
		ctx := r.Context()

		// Parse JSON request body
		var reqBody PortalRegisterAjaxRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err.Error() != "EOF" {
			errMsg := g.CoreAPI.Translate("error", "Invalid request format")
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Failed to parse request body - Error: %v", err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   errMsg,
			})
			return
		}

		// Get device MAC/IP/hostname from request
		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil {
			// Check if error is DHCP-related (we still have MAC/IP) or ARP failure (critical)
			if h != nil && h.MacAddr != "" {
				// Got MAC/IP but DHCP lease read failed - continue with empty hostname
			} else {
				// Critical error - couldn't identify device at all
				errMsg := g.CoreAPI.Translate("error", "Unable to identify device")
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Failed to get host from request - RemoteAddr: %s, Error: %v", r.RemoteAddr, err))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   errMsg,
				})
				return
			}
		}

		// Determine device ID from token or cookie
		var cookieDeviceID *int64

		// SCENARIO 1: Token provided - validate and extract device ID
		if reqBody.DeviceToken != "" {
			// Verify token signature
			_, machineID := machineuid.GetMachineUID()
			deviceID, err := devicetoken.VerifyDeviceToken(reqBody.DeviceToken, machineID)
			if err != nil {
				// Check if it's an expired token error
				errorCode := "invalid_token"
				if err.Error() == "token is expired" || err.Error() == "Token is expired" {
					errorCode = "expired_token"
				}
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Token verification failed - Error: %v", err))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   errorCode,
				})
				return
			}

			cookieDeviceID = &deviceID
		} else {
			// SCENARIO 2: No token - check for cookie fallback
			deviceID, cookieErr := devicetoken.GetDeviceCookie(r)
			if cookieErr == nil && deviceID > 0 {
				cookieDeviceID = &deviceID
			}
		}

		// Fallback to HTTP header for User-Agent if JavaScript collection failed
		// This ensures we always have User-Agent even if JS fingerprinting fails silently
		userAgent := reqBody.UserAgent
		if userAgent == "" {
			userAgent = r.Header.Get("User-Agent")
		}

		// Use ClientRegister.Register() for all scenarios
		// This handles: new device registration, MAC changes, IP changes, hostname changes
		// The Register() function will:
		// - Find device by ID (if cookieDeviceID provided)
		// - Handle MAC address changes (including randomization)
		// - Handle MAC conflicts (prevent cookie sharing)
		// - Update IP/hostname if changed
		// - Disconnect/reconnect sessions if device was active
		// - Create new device if not found
		clnt, shouldSetCookie, err := clntMgr.Register(ctx, sessmgr.ClientRegisterParams{
			CookieDeviceID: cookieDeviceID,
			MacAddr:        h.MacAddr,
			IpAddr:         h.IpAddr,
			Hostname:       h.Hostname,
			UserAgent:      userAgent,
			ScreenRes:      reqBody.ScreenRes,
			Language:       reqBody.Language,
			Timezone:       reqBody.Timezone,
		})
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "Failed to register device")
			cookieIDStr := "none"
			if cookieDeviceID != nil {
				cookieIDStr = fmt.Sprintf("%d", *cookieDeviceID)
			}
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Registration failed - MAC: %s, IP: %s, CookieID: %s, Error: %v",
				h.MacAddr, h.IpAddr, cookieIDStr, err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   errMsg,
			})
			return
		}
		if clnt == nil {
			errMsg := g.CoreAPI.Translate("error", "Failed to register device")
			cookieIDStr := "none"
			if cookieDeviceID != nil {
				cookieIDStr = fmt.Sprintf("%d", *cookieDeviceID)
			}
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Register returned nil client - MAC: %s, IP: %s, CookieID: %s",
				h.MacAddr, h.IpAddr, cookieIDStr))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   errMsg,
			})
			return
		}

		// Note: We can't reliably determine if device was "updated" here because
		// Register() already updated the device internally. The frontend doesn't
		// strictly need this flag - it's informational only.
		// For now, we'll set it to false and rely on the token validation flow.
		updated := false

		// Generate JWT device token for localStorage
		_, machineID := machineuid.GetMachineUID()
		deviceToken, err := devicetoken.GenerateDeviceToken(clnt.ID(), machineID)
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "Failed to generate device token")
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Failed to generate device token - DeviceID: %d, Error: %v", clnt.ID(), err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   errMsg,
			})
			return
		}

		// Set cookie as fallback (only if validation passed)
		if shouldSetCookie {
			if err := devicetoken.SetDeviceCookie(w, clnt.ID()); err != nil {
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Failed to set device cookie - DeviceID: %d, Error: %v", clnt.ID(), err))
			}
		}

		// Emit WiFi connect event - portal access confirms the client is connected
		// This serves as redundancy for hostapd_cli events and cancels any pending auto-pause
		api.EmitWifiEvent(sdkapi.WifiEventClientConnected, clnt.MacAddr())

		// Get redirect URL
		redirectUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:index")

		// Set "register" cookie with 12-hour expiration to allow access to /portal/index (only if not already set)
		if _, err := g.CoreAPI.HttpAPI.Cookie().GetPlainCookie(r, "register"); err != nil {
			cookieOpts := &sdkapi.HttpCookieOpts{
				Path:    "/",
				Expires: time.Now().Add(12 * time.Hour),
			}
			g.CoreAPI.HttpAPI.Cookie().SetPlainCookie(w, "register", "1", cookieOpts)
		}

		// Return JSON response with device token
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"device_token": deviceToken,
			"device_id":    clnt.ID(),
			"redirect_url": redirectUrl,
			"updated":      updated,
		})
	}
}

func PortalIndexPage(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for "register" cookie
		_, err := g.CoreAPI.HttpAPI.Cookie().GetPlainCookie(r, "register")
		if err != nil {
			// Cookie not present, redirect to root to trigger registration flow
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		p, t, err := g.PluginMgr.GetPortalTheme()
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "Unable to Get Portal Theme")
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(errMsg), http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}

		page := t.PortalTheme.IndexPageFactory(w, r)
		p.Http().Response().PortalView(w, r, page)
	}
}

func PortalSseHandler(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := sse.NewSocket(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		clnt, err := g.CoreAPI.HttpAPI.GetClientDevice(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse.AddSocket(fmt.Sprintf("%d", clnt.ID()), s)
		s.Listen()
	}
}
