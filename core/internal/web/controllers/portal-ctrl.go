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
			fmt.Println("DeviceMiddleware: Found cookie with device ID:", deviceID)
		} else if cookieErr != nil {
			fmt.Println("DeviceMiddleware: Invalid or missing device cookie:", cookieErr)
		}

		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Register/identify device with cookie validation and MAC randomization support
		clnt, shouldSetCookie, err := clntMgr.Register(r.Context(), connmgr.ClientRegisterParams{
			CookieDeviceID: cookieDeviceID,
			MacAddr:        h.MacAddr,
			IpAddr:         h.IpAddr,
			Hostname:       h.Hostname,
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Only set cookie if validation passed
		if shouldSetCookie {
			if err := devicetoken.SetDeviceCookie(w, clnt.ID()); err != nil {
				fmt.Println("DeviceMiddleware: Failed to set device cookie:", err)
			}
		} else {
			fmt.Println("DeviceMiddleware: Cookie validation failed, not setting cookie")
		}

		fmt.Println("DeviceMiddleware: Registered device:", clnt.ID(), clnt.MacAddr(), clnt.IpAddr())
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
			fmt.Println("PortalRegisterAjax: Failed to parse request body:", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   errMsg,
			})
			return
		}

		// Get device MAC/IP/hostname from request
		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "Unable to identify device")
			fmt.Println("PortalRegisterAjax: Failed to get host from request:", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   errMsg,
			})
			return
		}

		// Determine device ID from token or cookie
		var cookieDeviceID *int64

		// SCENARIO 1: Token provided - validate and extract device ID
		if reqBody.DeviceToken != "" {
			fmt.Println("PortalRegisterAjax: Token provided, validating...")

			// Verify token signature
			machineID := machineuid.GetMachineUID()
			deviceID, err := devicetoken.VerifyDeviceToken(reqBody.DeviceToken, machineID)
			if err != nil {
				// Check if it's an expired token error
				errorCode := "invalid_token"
				if err.Error() == "token is expired" || err.Error() == "Token is expired" {
					errorCode = "expired_token"
				}
				fmt.Println("PortalRegisterAjax: Token verification failed:", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   errorCode,
				})
				return
			}

			fmt.Println("PortalRegisterAjax: Token valid, device ID:", deviceID)
			cookieDeviceID = &deviceID
		} else {
			// SCENARIO 2: No token - check for cookie fallback
			fmt.Println("PortalRegisterAjax: No token provided, checking for cookie")

			deviceID, cookieErr := devicetoken.GetDeviceCookie(r)
			if cookieErr == nil && deviceID > 0 {
				cookieDeviceID = &deviceID
				fmt.Println("PortalRegisterAjax: Found cookie with device ID:", deviceID)
			} else if cookieErr != nil {
				fmt.Println("PortalRegisterAjax: Invalid or missing device cookie:", cookieErr)
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
			fmt.Println("PortalRegisterAjax: Failed to register device:", err)
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
			fmt.Println("PortalRegisterAjax: Failed to generate device token:", err)
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
				fmt.Println("PortalRegisterAjax: Failed to set device cookie:", err)
			}
		} else {
			fmt.Println("PortalRegisterAjax: Cookie validation failed, not setting cookie")
		}

		// Get redirect URL
		redirectUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("portal:index")

		fmt.Println("PortalRegisterAjax: Success - device:", clnt.ID(), clnt.MacAddr(), clnt.IpAddr(), "updated:", updated)

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
