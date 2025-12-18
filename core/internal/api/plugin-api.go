package api

import (
	"database/sql"
	"log"

	"core/db"
	"core/db/models"
	"core/internal/network"
	"core/internal/sessmgr"
	"core/tools/plugins"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewPluginApi(dir string, info sdkutils.PluginInfo, assets *GlobalAssets, pmgr *PluginsMgr, trfkMgr *network.TrafficMgr) *PluginApi {
	pluginApi := &PluginApi{
		dir:            dir,
		db:             pmgr.db,
		PluginsMgrApi:  pmgr,
		ClientRegister: pmgr.clntReg,
		SessionMgr:     pmgr.clntMgr,
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

	log.Println("NewPluginApi: ", dir, " - ", info.Package, " - ", info.Name, " - ", info.Version, " - ", info.Description)

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
}

func (self *PluginApi) Initialize(coreApi *PluginApi) {
	self.CoreAPI = coreApi
	self.HttpAPI.Initialize()
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
	return self.PluginsMgrApi
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

func (self *PluginApi) LoadAssetsManifest() {
	self.AssetsManifest = plugins.GetAssetManifest(self.dir)
}
