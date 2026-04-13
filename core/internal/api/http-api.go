package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"core/db"
	"core/db/models"
	devicetoken "core/internal/modules/device-token"
	"core/internal/sessmgr"
	"core/utils/hostfinder"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

func NewHttpApi(api *PluginApi, db *db.Database, assets *GlobalAssets, clnt *sessmgr.ClientRegister, mdls *models.Models, dmgr *sessmgr.ClientRegister, pmgr *PaymentsMgr) {
	navs := NewNavsApi(api)
	auth := NewHttpAuth(api)
	httpResp := NewHttpResponse(api, assets)
	httpRouter := NewHttpRouterApi(api, db, clnt)
	httpForm := NewHttpFormApi(api)
	httpCookie := NewHttpCookie(api)

	httpApi := &HttpApi{
		api:            api,
		auth:           auth,
		httpRouter:     httpRouter,
		navsApi:        navs,
		httpResp:       httpResp,
		formsApi:       httpForm,
		httpCookie:     httpCookie,
		clientRegister: clnt,
	}

	api.HttpAPI = httpApi
}

type HttpApi struct {
	api            *PluginApi
	auth           *HttpAuth
	httpRouter     *HttpRouterApi
	navsApi        *HttpNavsApi
	formsApi       *HttpFormApi
	httpResp       *HttpResponse
	httpCookie     *HttpCookie
	clientRegister *sessmgr.ClientRegister
}

func (self *HttpApi) Initialize() {
	self.httpRouter.Initialize()
}

func (self *HttpApi) GetClientDevice(r *http.Request) (sdkapi.IClientDevice, error) {
	var claims devicetoken.DeviceTokenClaims
	var err error

	// Priority 1: Check X-Device-Token header (localStorage-based AJAX auth)
	claims, err = devicetoken.GetDeviceFromHeader(r)
	if err != nil {
		// Priority 2: Fallback to cookie (traditional auth)
		claims, err = devicetoken.GetDeviceCookie(r)
		if err != nil {
			return nil, fmt.Errorf("failed to get device from header or cookie: %w", err)
		}
	}

	// Query device from database with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	clnt, err := self.clientRegister.FindByID(ctx, claims.DeviceID)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[TIMEOUT] GetClientDevice exceeded 2s timeout for DeviceID=%d", claims.DeviceID)
			return nil, fmt.Errorf("device lookup timed out")
		}
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// Validate cookie_token if the device has one set
	if clnt.CookieToken() != "" && claims.CookieToken != clnt.CookieToken() {
		return nil, fmt.Errorf("cookie token mismatch for device %d", claims.DeviceID)
	}

	// Cross-validate current request's MAC/IP against stored device record.
	// Cookie token already proves device identity (checked above), so a mismatch here
	// is a legitimate network change — MAC randomization, DHCP reassignment, dual-stack
	// reordering — not an attack. Sync the stored record so callers see live values.
	// Skip on SSE/events to avoid disconnect/reconnect storms on high-frequency endpoints.
	shouldCheckMAC := !strings.HasSuffix(r.URL.Path, "/events") && !strings.Contains(r.URL.Path, "/sse")
	if shouldCheckMAC {
		if h, arpErr := hostfinder.GetHostFromRequest(r); arpErr == nil && h != nil && h.MacAddr != "" {
			storedMAC := strings.ToUpper(clnt.MacAddr())
			currentMAC := strings.ToUpper(h.MacAddr)
			requestIP := h.IpAddr
			storedIPv4 := clnt.Ipv4Addr()
			storedIPv6 := clnt.Ipv6Addr()

			macChanged := storedMAC != "" && currentMAC != storedMAC
			ipChanged := requestIP != "" && (storedIPv4 != "" || storedIPv6 != "") && requestIP != storedIPv4 && requestIP != storedIPv6

			if macChanged || ipChanged {
				newIpv4, newIpv6 := storedIPv4, storedIPv6
				if parsed := net.ParseIP(requestIP); parsed != nil {
					if v4 := parsed.To4(); v4 != nil {
						newIpv4 = v4.String()
					} else {
						newIpv6 = parsed.String()
					}
				}
				newHostname := h.Hostname
				if newHostname == "" {
					newHostname = clnt.Hostname()
				}
				if err := self.clientRegister.UpdateDevice(ctx, clnt, h.MacAddr, newIpv4, newIpv6, newHostname); err != nil {
					log.Printf("[GetClientDevice] failed to sync device %d network details (MAC %s→%s, IP %s): %v",
						claims.DeviceID, storedMAC, currentMAC, requestIP, err)
				} else if refreshed, err := self.clientRegister.FindByID(ctx, claims.DeviceID); err == nil {
					clnt = refreshed
				}
			}
		}
	}

	return clnt, nil
}

func (self *HttpApi) Cookie() sdkapi.IHttpCookie {
	return self.httpCookie
}

func (self *HttpApi) Auth() sdkapi.IHttpAuth {
	return self.auth
}

func (self *HttpApi) Router() sdkapi.IHttpRouterApi {
	return self.httpRouter
}

func (self *HttpApi) Forms() sdkapi.IHttpFormsApi {
	return self.formsApi
}

func (self *HttpApi) Helpers() sdkapi.IHttpHelpers {
	return NewHttpHelpers(self.api)
}

func (self *HttpApi) Response() sdkapi.IHttpResponse {
	return self.httpResp
}

func (self *HttpApi) MuxVars(r *http.Request) map[string]string {
	return mux.Vars(r)
}

func (self *HttpApi) Navs() sdkapi.INavsApi {
	return self.navsApi
}
