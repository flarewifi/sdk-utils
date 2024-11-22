package plugins

import (
	"log"
	"path/filepath"

	"core/internal/connmgr"
	"core/internal/db"
	"core/internal/db/models"
	"core/internal/network"
	"core/internal/utils/migrate"
	"core/internal/utils/pkg"
	sdkacct "sdk/api/accounts"
	sdkads "sdk/api/ads"
	sdkcfg "sdk/api/config"
	sdkconnmgr "sdk/api/connmgr"
	sdkhttp "sdk/api/http"
	sdkinappur "sdk/api/inappur"
	sdklogger "sdk/api/logger"
	sdknet "sdk/api/network"
	sdkpayments "sdk/api/payments"
	sdkplugin "sdk/api/plugin"
	sdkuci "sdk/api/uci"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPluginApi(dir string, pmgr *PluginsMgr, trfkMgr *network.TrafficMgr) *PluginApi {
	pluginApi := &PluginApi{
		dir:           dir,
		db:            pmgr.db,
		PluginsMgrApi: pmgr,
		ClntReg:       pmgr.clntReg,
		ClntMgr:       pmgr.clntMgr,
	}

	pluginApi.Utl = NewPluginUtils(pluginApi)

	info, err := pkg.GetSrcInfo(dir)
	if err != nil {
		log.Println("Error getting plugin info: ", err.Error())
		return nil
	}

	pluginApi.info = &info
	pluginApi.models = pmgr.models

	NewAcctApi(pluginApi)
	NewHttpApi(pluginApi, pmgr.db, pmgr.clntReg, pmgr.models, pmgr.clntReg, pmgr.paymgr)
	NewConfigApi(pluginApi)
	NewPaymentsApi(pluginApi, pmgr.paymgr)
	NewThemesApi(pluginApi)
	NewNetworkApi(pluginApi, trfkMgr)
	NewAdsApi(pluginApi)
	NewInAppPurchaseApi(pluginApi)
	NewUciApi(pluginApi)
	NewLoggerApi(pluginApi)

	log.Println("NewPluginApi: ", dir, " - ", info.Package, " - ", info.Name, " - ", info.Version, " - ", info.Description)

	return pluginApi
}

type PluginApi struct {
	info             *sdkplugin.PluginInfo
	dir              string
	db               *db.Database
	models           *models.Models
	CoreAPI          *PluginApi
	AcctAPI          *AccountsApi
	HttpAPI          *HttpApi
	ConfigAPI        *ConfigApi
	PaymentsAPI      *PaymentsApi
	ThemesAPI        *HttpThemesApi
	NetworkAPI       *NetworkApi
	AdsAPI           *AdsApi
	InAppPurchaseAPI *InAppPurchaseApi
	PluginsMgrApi    *PluginsMgr
	ClntReg          *connmgr.ClientRegister
	ClntMgr          *connmgr.SessionsMgr
	UciAPI           *UciApi
	Utl              *PluginUtils
	LoggerAPI        *LoggerApi
	AssetsManifest   pkg.OutputManifest
}

func (self *PluginApi) Initialize(coreApi *PluginApi) {
	self.CoreAPI = coreApi
	self.HttpAPI.Initialize()
}

func (self *PluginApi) Migrate() error {
	migdir := filepath.Join(self.dir, "resources/migrations")
	err := migrate.MigrateUp(self.db.SqlDB(), migdir)
	if err != nil {
		log.Println("Error in plugin migration "+self.Name(), ":", err.Error())
		return err
	}

	log.Println("Done migrating plugin:", self.Name())
	return nil
}

func (self *PluginApi) Name() string {
	return self.info.Name
}

func (self *PluginApi) Pkg() string {
	return self.info.Package
}

func (self *PluginApi) Version() string {
	return self.info.Version
}

func (self *PluginApi) Description() string {
	info, err := pkg.GetSrcInfo(self.dir)
	if err != nil {
		return ""
	}
	return info.Description
}

func (self *PluginApi) Dir() string {
	return self.dir
}

func (self *PluginApi) Translate(t string, msgk string, pairs ...interface{}) string {
	return self.Utl.Translate(t, msgk, pairs...)
}

func (self *PluginApi) Resource(f string) (path string) {
	return self.Utl.Resource(f)
}

func (self *PluginApi) SqlDb() *pgxpool.Pool {
	return self.db.SqlDB()
}

func (self *PluginApi) Acct() sdkacct.AccountsApi {
	return self.AcctAPI
}

func (self *PluginApi) Http() sdkhttp.IHttpApi {
	return self.HttpAPI
}

func (self *PluginApi) Config() sdkcfg.IConfigApi {
	return self.ConfigAPI
}

func (self *PluginApi) Payments() sdkpayments.IPaymentsApi {
	return self.PaymentsAPI
}

func (self *PluginApi) Ads() sdkads.IAdsApi {
	return self.AdsAPI
}

func (self *PluginApi) InAppPurchases() sdkinappur.IInAppPurchasesApi {
	return self.InAppPurchaseAPI
}

func (self *PluginApi) PluginsMgr() sdkplugin.IPluginsMgrApi {
	return self.PluginsMgrApi
}

func (self *PluginApi) Network() sdknet.INetworkApi {
	return self.NetworkAPI
}

func (self *PluginApi) DeviceHooks() sdkconnmgr.IDeviceHooksApi {
	return self.ClntReg
}

func (self *PluginApi) SessionsMgr() sdkconnmgr.ISessionsMgrApi {
	return self.ClntMgr
}

func (self *PluginApi) Uci() sdkuci.IUciApi {
	return self.UciAPI
}

func (self *PluginApi) Themes() sdkhttp.IHttpThemesApi {
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

func (self *PluginApi) Logger() sdklogger.ILoggerApi {
	return self.LoggerAPI
}

func (self *PluginApi) LoadAssetsManifest() {
	self.AssetsManifest = pkg.GetAssetManifest(self.dir)
}
