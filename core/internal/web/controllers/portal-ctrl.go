package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"core/internal/api"
	"core/internal/connmgr"
	devicetoken "core/internal/utils/device-token"
	"core/internal/utils/hostfinder"
	machineuid "core/internal/utils/machine-uid"
	sse "core/internal/utils/sse"
	portalview "core/resources/views/portal"
	sdkapi "sdk/api"
)

// PortalRedirectCtrl handles the /portal route
// Uses AJAX-based registration with localStorage token management
// Falls back to cookie-based registration on any error
func PortalRedirectCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If device already registered, redirect to portal index
		clnt, err := g.CoreAPI.HttpAPI.GetClientDevice(r)
		if err == nil && clnt != nil {
			g.CoreAPI.HttpAPI.Response().Redirect(w, r, "portal:index")
			return
		}

		// Get URLs for AJAX registration flow
		redirectUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:index")
		registerUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:register:ajax")
		fallbackUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:register")

		// Render AJAX registration page
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
			userMsg := g.CoreAPI.Translate("error", "Unable to identify your device from the network")
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("PortalRegisterCtrl: Failed to identify device - RemoteAddr: %s, Error: %v", r.RemoteAddr, err))
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(userMsg), http.StatusBadRequest)
			return
		}
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterCtrl: Identified device - MAC: %s, IP: %s, Hostname: %s", h.MacAddr, h.IpAddr, h.Hostname))

		// Register/identify device with cookie validation and MAC randomization support
		clnt, shouldSetCookie, err := clntMgr.Register(r.Context(), connmgr.ClientRegisterParams{
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
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("PortalRegisterAjax: Identified device - MAC: %s, IP: %s", h.MacAddr, h.IpAddr))

		// Determine device ID from token or cookie
		var cookieDeviceID *int64

		// SCENARIO 1: Token provided - validate and extract device ID
		if reqBody.DeviceToken != "" {
			g.CoreAPI.LoggerAPI.Info("PortalRegisterAjax: Token provided, validating...")

			// Verify token signature
			machineID := machineuid.GetMachineUID()
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
		clnt, shouldSetCookie, err := clntMgr.Register(ctx, connmgr.ClientRegisterParams{
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
		machineID := machineuid.GetMachineUID()
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
