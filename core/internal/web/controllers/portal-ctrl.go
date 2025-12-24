package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"core/internal/api"
	"core/internal/network"
	"core/internal/sessmgr"
	devicetoken "core/internal/utils/device-token"
	machineuid "core/internal/utils/machine-uid"
	portalview "core/resources/views/portal"
	"core/tools/hostfinder"
	sse "core/tools/sse"
	sdkapi "sdk/api"
)

// PortalRootCtrl handles the root path "/"
// Renders a simple HTML page with inline JavaScript that redirects to http://<lan-ip>/portal/redirect
func PortalRootCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var lanIP string

		// Get client IP from request
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// Fallback: use request host
			lanIP = r.Host
		} else {
			// Find LAN interface by client IP
			lan, err := network.FindByIp(ip)
			if err != nil {
				// Fallback: use request host
				lanIP = r.Host
			} else {
				// Get LAN IP address
				lanIPAddr, err := lan.GetInterface().IpV4Addr()
				if err != nil {
					// Fallback: use request host
					lanIP = r.Host
				} else {
					lanIP = lanIPAddr.Addr
				}
			}
		}

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

		// Always render AJAX registration page to trigger device validation/update
		// This ensures MAC/IP changes are detected (e.g., MAC randomization)
		// The AJAX endpoint will validate existing tokens or register new devices
		page := portalview.PortalRedirectPage(g.CoreAPI, redirectUrl, registerUrl, fallbackUrl)
		v := sdkapi.ViewPage{
			Assets: sdkapi.ViewAssets{
				JsFile:  "portal-register.js",
				CssFile: "portal-redirect.css",
			},
			PageContent: page,
		}
		g.CoreAPI.HttpAPI.Response().PortalView(w, r, v)
	}
}

func PortalRegisterCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Request received - RemoteAddr: %s", r.RemoteAddr))
		clntMgr := g.CoreAPI.ClientRegister

		// Get device ID from JWT cookie (if exists)
		var cookieDeviceID *int64
		deviceID, cookieErr := devicetoken.GetDeviceCookie(r)
		if cookieErr == nil && deviceID > 0 {
			cookieDeviceID = &deviceID
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Found device cookie - ID: %d", deviceID))
		} else if cookieErr != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Invalid device cookie - Error: %v", cookieErr))
		}

		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil {
			// Check if error is DHCP-related (we still have MAC/IP) or ARP failure (critical)
			if h != nil && h.MacAddr != "" {
				// Got MAC/IP but DHCP lease read failed - log warning but continue
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: DHCP lease lookup failed (continuing with empty hostname) - RemoteAddr: %s, MAC: %s, IP: %s, Error: %v",
					r.RemoteAddr, h.MacAddr, h.IpAddr, err))
			} else {
				// Critical error - couldn't identify device at all
				userMsg := g.CoreAPI.Translate("error", "Unable to identify your device from the network")
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Failed to identify device - RemoteAddr: %s, Error: %v", r.RemoteAddr, err))
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(userMsg), http.StatusBadRequest)
				return
			}
		}
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Identified device - MAC: %s, IP: %s, Hostname: %s", h.MacAddr, h.IpAddr, h.Hostname))

		// Register/identify device with cookie validation and MAC randomization support
		clnt, shouldSetCookie, err := clntMgr.Register(r.Context(), sessmgr.ClientRegisterParams{
			CookieDeviceID: cookieDeviceID,
			MacAddr:        h.MacAddr,
			IpAddr:         h.IpAddr,
			Hostname:       h.Hostname,
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

		// Only set cookie if validation passed
		if shouldSetCookie {
			if err := devicetoken.SetDeviceCookie(w, clnt.ID()); err != nil {
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Failed to set device cookie - DeviceID: %d, Error: %v", clnt.ID(), err))
				// Don't fail the request for cookie errors
			} else {
				g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Set device cookie - DeviceID: %d", clnt.ID()))
			}
		} else {
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Cookie validation failed, not setting cookie - DeviceID: %d", clnt.ID()))
		}

		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Successfully registered device - ID: %d, MAC: %s, IP: %s",
			clnt.ID(), clnt.MacAddr(), clnt.IpAddr()))

		// Set "register" cookie to allow access to /portal/index
		g.CoreAPI.HttpAPI.Cookie().SetPlainCookie(w, "register", "1", nil)
		g.CoreAPI.HttpAPI.Response().Redirect(w, r, "portal:index")
	}
}

// PortalRegisterAjaxRequest represents the JSON request body for AJAX registration
type PortalRegisterAjaxRequest struct {
	DeviceToken string `json:"device_token"`
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

		hasToken := reqBody.DeviceToken != ""
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Request parsed - HasToken: %v", hasToken))

		// Get device MAC/IP/hostname from request
		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil {
			// Check if error is DHCP-related (we still have MAC/IP) or ARP failure (critical)
			if h != nil && h.MacAddr != "" {
				// Got MAC/IP but DHCP lease read failed - log warning but continue
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: DHCP lease lookup failed (continuing with empty hostname) - RemoteAddr: %s, MAC: %s, IP: %s, Error: %v",
					r.RemoteAddr, h.MacAddr, h.IpAddr, err))
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
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Identified device - MAC: %s, IP: %s, Hostname: %s", h.MacAddr, h.IpAddr, h.Hostname))

		// Determine device ID from token or cookie
		var cookieDeviceID *int64

		// SCENARIO 1: Token provided - validate and extract device ID
		if reqBody.DeviceToken != "" {
			g.CoreAPI.LoggerAPI.Info("PortalRegisterAjax: Token provided, validating...")

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

			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Token valid - DeviceID: %d", deviceID))
			cookieDeviceID = &deviceID
		} else {
			// SCENARIO 2: No token - check for cookie fallback
			g.CoreAPI.LoggerAPI.Info("PortalRegisterAjax: No token provided, checking for cookie")

			deviceID, cookieErr := devicetoken.GetDeviceCookie(r)
			if cookieErr == nil && deviceID > 0 {
				cookieDeviceID = &deviceID
				g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Found cookie - DeviceID: %d", deviceID))
			} else if cookieErr != nil {
				g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterAjax: Invalid or missing device cookie - Error: %v", cookieErr))
			}
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
			} else {
				g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Set device cookie - DeviceID: %d", clnt.ID()))
			}
		} else {
			g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Cookie validation failed, not setting cookie - DeviceID: %d", clnt.ID()))
		}

		// Get redirect URL
		redirectUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:index")

		// Set "register" cookie to allow access to /portal/index
		g.CoreAPI.HttpAPI.Cookie().SetPlainCookie(w, "register", "1", nil)

		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Success - DeviceID: %d, MAC: %s, IP: %s, Updated: %v",
			clnt.ID(), clnt.MacAddr(), clnt.IpAddr(), updated))

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
			// Cookie not present, redirect to portal root
			g.CoreAPI.LoggerAPI.Info("PortalIndexPage: No register cookie found, redirecting to portal root")
			g.CoreAPI.HttpAPI.Response().RedirectToPortal(w, r)
			return
		}

		// Cookie present, remove it and render the page
		g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, "register")
		g.CoreAPI.LoggerAPI.Info("PortalIndexPage: Register cookie found and removed, rendering index page")

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

		sse.AddSocket(clnt.MacAddr(), s)
		s.Listen()
	}
}
