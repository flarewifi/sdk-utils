package api

import (
	"database/sql"
	"net/http"
	"sync"

	"core/db"
	"core/db/models"
	"core/internal/events"
	"core/internal/modules/scheduler"
	"core/internal/modules/ubus"
	"core/internal/network"
	"core/internal/sessmgr"
	"core/utils/plugins"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func NewPluginApi(dir string, info sdkutils.PluginInfo, assets *GlobalAssets, pmgr *PluginsMgr, trfkMgr *network.TrafficMgr, wifiMgr *ubus.WifiMgr) *PluginApi {
	pluginApi := &PluginApi{
		dir:            dir,
		db:             pmgr.db,
		PluginsMgrApi:  pmgr,
		ClientRegister: pmgr.clntReg,
		SessionMgr:     pmgr.clntMgr,
		EventsMgr:      pmgr.eventsMgr,
		SchedulerMgr:   pmgr.schedulerMgr,
	}

	pluginApi.info = info

	pluginApi.Utl = NewPluginUtils(pluginApi)
	pluginApi.models = pmgr.models

	NewAcctApi(pluginApi)
	NewHttpApi(pluginApi, pmgr.db, assets, pmgr.clntReg, pmgr.models, pmgr.clntReg, pmgr.paymgr)
	NewConfigApi(pluginApi)
	NewPaymentsApi(pluginApi, pmgr.paymgr)
	NewThemesApi(pluginApi)
	NewNetworkApi(pluginApi, trfkMgr)
	NewAdsApi(pluginApi)
	NewInAppPurchaseApi(pluginApi)
	NewUciApi(pluginApi)
	NewLoggerApi(pluginApi)
	NewSessionsMgrApi(pluginApi)
	NewMachineApi(pluginApi)
	NewFirewallApi(pluginApi)
	pluginApi.UIApi = NewUIApi(pluginApi)
	pluginApi.NotificationAPI = NewNotificationAPI(pluginApi, pmgr.models)
	NewVouchersApi(pluginApi)
	NewWifiApi(pluginApi, wifiMgr)
	pluginApi.StorageAPI = NewStorageApi(pluginApi).(*StorageApi)
	NewSchedulerApi(pluginApi)

	return pluginApi
}

type PluginApi struct {
	info             sdkutils.PluginInfo
	dir              string
	db               *db.Database
	models           *models.Models
	CoreAPI          *PluginApi
	AcctAPI          *AccountsApi
	HttpAPI          *HttpApi
	ConfigAPI        *ConfigApi
	PaymentsAPI      *PaymentsApi
	ThemesAPI        *ThemesApi
	NetworkAPI       *NetworkApi
	AdsAPI           *AdsApi
	InAppPurchaseAPI *InAppPurchaseApi
	PluginsMgrApi    *PluginsMgr
	ClientRegister   *sessmgr.ClientRegister
	SessionMgr       *sessmgr.SessionsMgr
	SessionsMgrAPI   *SessionsMgrApi
	UciAPI           *UciApi
	Utl              *PluginUtils
	LoggerAPI        *LoggerApi
	AssetsManifest   plugins.OutputManifest
	UIApi            *UIApi
	NotificationAPI  *NotificationAPI
	MachineAPI       *MachineApi
	FirewallAPI      *FirewallApi
	VouchersAPI      *VouchersApi
	WifiAPI          *WifiApi
	StorageAPI       *StorageApi
	EventsMgr        *events.EventsManager
	SchedulerMgr     *scheduler.Manager
	SchedulerAPI     *SchedulerApi

	// initFn is the plugin's entry point — func Init(api sdkapi.IPluginApi) error.
	// Loading the plugin (plugin.Open in non-mono, the generated mono loader
	// otherwise) only resolves and stores it here; RunInit invokes it. Splitting
	// the two lets the plugin be loaded at boot (offline-safe) while its Init is
	// held back until any internet-dependent provisioning (system_packages /
	// preinstall) has completed. See boot.InitLoadedPlugins / ProvisionInstalledPlugins.
	initFn   func(sdkapi.IPluginApi) error
	initMu   sync.Mutex
	initDone bool
}

func (self *PluginApi) Initialize(coreApi *PluginApi) {
	self.CoreAPI = coreApi
	self.HttpAPI.Initialize()
}

// SetInitFn records the plugin's Init entry point, to be invoked later by RunInit.
// Loaders call this instead of invoking Init directly.
func (self *PluginApi) SetInitFn(fn func(sdkapi.IPluginApi) error) {
	self.initFn = fn
}

// InitDone reports whether the plugin's Init has already run successfully.
func (self *PluginApi) InitDone() bool {
	self.initMu.Lock()
	defer self.initMu.Unlock()
	return self.initDone
}

// RunInit invokes the plugin's Init entry point exactly once. It is safe to call
// from both the boot goroutine and the online-monitor provisioning goroutine: the
// first successful run marks the plugin initialized and later calls are no-ops. A
// failed Init leaves the plugin un-initialized so a later pass can retry.
func (self *PluginApi) RunInit() error {
	self.initMu.Lock()
	defer self.initMu.Unlock()
	if self.initDone {
		return nil
	}
	if self.initFn != nil {
		if err := self.initFn(self); err != nil {
			return err
		}
	}
	self.initDone = true
	return nil
}

func (self *PluginApi) Info() sdkutils.PluginInfo {
	return self.info
}

func (self *PluginApi) Dir() string {
	return self.dir
}

func (self *PluginApi) Translate(t string, msgk string, pairs ...any) string {
	return self.Utl.Translate(t, msgk, pairs...)
}

func (self *PluginApi) Resource(f string) (path string) {
	return self.Utl.Resource(f)
}

func (self *PluginApi) SqlDB() *sql.DB {
	return self.db.DB
}

func (self *PluginApi) Acct() sdkapi.IAccountsApi {
	return self.AcctAPI
}

func (self *PluginApi) Http() sdkapi.IHttpApi {
	return self.HttpAPI
}

func (self *PluginApi) Config() sdkapi.IConfigApi {
	return self.ConfigAPI
}

func (self *PluginApi) Payments() sdkapi.IPaymentsApi {
	return self.PaymentsAPI
}

func (self *PluginApi) Ads() sdkapi.IAdsApi {
	return self.AdsAPI
}

func (self *PluginApi) InAppPurchases() sdkapi.IInAppPurchasesApi {
	return self.InAppPurchaseAPI
}

func (self *PluginApi) PluginsMgr() sdkapi.IPluginsMgrApi {
	// Return a per-plugin view bound to this PluginApi. The shared PluginsMgr is a
	// singleton and can't know which plugin called it, but GetPurchaseURL must
	// resolve a callback route in the CALLER's namespace — so the wrapper carries
	// the owner. All other methods pass through to the embedded PluginsMgr.
	return &pluginScopedPluginsMgr{PluginsMgr: self.PluginsMgrApi, owner: self}
}

// pluginScopedPluginsMgr adapts the shared PluginsMgr to a single calling plugin.
// It embeds *PluginsMgr (so every IPluginsMgrApi method is inherited unchanged)
// and overrides only GetPurchaseURL, which needs the caller's route namespace.
type pluginScopedPluginsMgr struct {
	*PluginsMgr
	owner *PluginApi
}

func (w *pluginScopedPluginsMgr) GetPurchaseURL(r *http.Request, pkg string, callbackRouteName string, pairs ...string) (string, error) {
	return w.PluginsMgr.buildPurchaseURL(r, w.owner, pkg, callbackRouteName, pairs...)
}

func (self *PluginApi) Network() sdkapi.INetworkApi {
	return self.NetworkAPI
}

func (self *PluginApi) SessionsMgr() sdkapi.ISessionsMgrApi {
	return self.SessionsMgrAPI
}

func (self *PluginApi) Uci() sdkapi.IUciApi {
	return self.UciAPI
}

func (self *PluginApi) UI() sdkapi.IUIApi {
	return self.UIApi
}

func (self *PluginApi) Notification() sdkapi.INotificationAPI {
	return self.NotificationAPI
}

func (self *PluginApi) Themes() sdkapi.IThemesApi {
	return self.ThemesAPI
}

func (self *PluginApi) Features() []string {
	features := []string{}
	if self.ThemesAPI.AdminTheme != nil {
		features = append(features, "theme:admin")
	}
	if self.ThemesAPI.PortalTheme != nil {
		features = append(features, "theme:portal")
	}
	return features
}

func (self *PluginApi) Logger() sdkapi.ILoggerApi {
	return self.LoggerAPI
}

func (self *PluginApi) Machine() sdkapi.IMachineApi {
	return self.MachineAPI
}

func (self *PluginApi) Firewall() sdkapi.IFirewallAPI {
	return self.FirewallAPI
}

func (self *PluginApi) Vouchers() sdkapi.IVouchersApi {
	return self.VouchersAPI
}

func (self *PluginApi) Wifi() sdkapi.IWifiApi {
	return self.WifiAPI
}

func (self *PluginApi) Storage() sdkapi.IStorageApi {
	return self.StorageAPI
}

func (self *PluginApi) Events() sdkapi.IEventsApi {
	return self.EventsMgr
}

func (self *PluginApi) Scheduler() sdkapi.ISchedulerApi {
	return self.SchedulerAPI
}

func (self *PluginApi) LoadAssetsManifest() {
	self.AssetsManifest = plugins.GetAssetManifest(self.dir)
}
