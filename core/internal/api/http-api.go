package api

import (
	"context"
	"fmt"
	"log"
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
	var deviceID int64
	var err error

	// Priority 1: Check X-Device-Token header (localStorage-based AJAX auth)
	deviceID, err = devicetoken.GetDeviceFromHeader(r)
	if err != nil {
		// Priority 2: Fallback to cookie (traditional auth)
		deviceID, err = devicetoken.GetDeviceCookie(r)
		if err != nil {
			return nil, fmt.Errorf("failed to get device from header or cookie: %w", err)
		}
	}

	// Query device from database with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	clnt, err := self.clientRegister.FindByID(ctx, deviceID)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[TIMEOUT] GetClientDevice exceeded 2s timeout for DeviceID=%d", deviceID)
			return nil, fmt.Errorf("device lookup timed out")
		}
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// Security: Cross-validate current request's MAC/IP against stored device record
	// This detects potential token theft or spoofing attempts
	// Log-only mode: do not reject to avoid breaking legitimate cases (DHCP reassignment, MAC randomization)
	// Skip ARP check for SSE and other high-frequency endpoints to reduce DHCP/ARP lookup overhead
	shouldCheckMAC := !strings.HasSuffix(r.URL.Path, "/events") && !strings.Contains(r.URL.Path, "/sse")
	if shouldCheckMAC {
		if h, arpErr := hostfinder.GetHostFromRequest(r); arpErr == nil && h != nil && h.MacAddr != "" {
			storedMAC := strings.ToUpper(clnt.MacAddr())
			currentMAC := strings.ToUpper(h.MacAddr)
			if storedMAC != "" && currentMAC != storedMAC {
				log.Printf("[SECURITY] GetClientDevice: MAC mismatch - token claims device %d (MAC: %s) but ARP shows MAC: %s (IP: %s, RemoteAddr: %s)",
					deviceID, storedMAC, currentMAC, h.IpAddr, r.RemoteAddr)
			}
			// Also check IP mismatch (less critical but useful for logging)
			if clnt.IpAddr() != "" && h.IpAddr != "" && clnt.IpAddr() != h.IpAddr {
				log.Printf("[SECURITY] GetClientDevice: IP mismatch - device %d stored IP: %s but request from IP: %s (MAC: %s)",
					deviceID, clnt.IpAddr(), h.IpAddr, currentMAC)
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
