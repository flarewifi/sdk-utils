package api

import (
	"fmt"
	"log"
	sdkplugin "sdk/api"

	"core/db"
	"core/db/models"
	"core/internal/config"
	"core/internal/connmgr"
	"core/internal/network"
)

func NewPluginMgr(d *db.Database, m *models.Models, paymgr *PaymentsMgr, clntReg *connmgr.ClientRegister, clntMgr *connmgr.SessionsMgr, trfkMgr *network.TrafficMgr) *PluginsMgr {
	pmgr := &PluginsMgr{
		db:      d,
		models:  m,
		paymgr:  paymgr,
		clntReg: clntReg,
		clntMgr: clntMgr,
		plugins: []*PluginApi{},
	}
	return pmgr
}

type PluginsMgr struct {
	CoreAPI *PluginApi
	db      *db.Database
	models  *models.Models
	paymgr  *PaymentsMgr
	clntReg *connmgr.ClientRegister
	clntMgr *connmgr.SessionsMgr
	trfkMgr *network.TrafficMgr
	plugins []*PluginApi
}

func (self *PluginsMgr) InitCoreApi(coreApi *PluginApi) {
	self.CoreAPI = coreApi
}

func (self *PluginsMgr) Plugins() []*PluginApi {
	return self.plugins
}

func (self *PluginsMgr) RegisterPlugin(p *PluginApi) error {
	if p.Info().Package != self.CoreAPI.Info().Package {
		err := p.Init()
		if err != nil {
			log.Println("Error initializing plugin: "+p.Dir(), err)
			// TODO: set plugin as broken
			return fmt.Errorf("%w: Error initializing plugin: %v", err, p.Dir())
		}
	}

	p.Initialize(self.CoreAPI)
	p.LoadAssetsManifest()
	self.plugins = append(self.plugins, p)

	return nil
}

func (self *PluginsMgr) FindByName(name string) (sdkplugin.IPluginApi, bool) {
	for _, p := range self.plugins {
		if p.Info().Name == name {
			return p, true
		}
	}
	return nil, false
}

func (self *PluginsMgr) FindByPkg(pkg string) (sdkplugin.IPluginApi, bool) {
	for _, p := range self.plugins {
		if p.Info().Package == pkg {
			return p, true
		}
	}
	return nil, false
}

func (self *PluginsMgr) All() []sdkplugin.IPluginApi {
	plugins := []sdkplugin.IPluginApi{}
	for _, p := range self.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (self *PluginsMgr) PaymentMethods() []sdkplugin.IPluginApi {
	methods := []sdkplugin.IPluginApi{}
	for _, p := range self.plugins {
		pmnt := p.Payments().(*PaymentsApi)
		if pmnt.paymentsMgr != nil {
			methods = append(methods, p)
		}
	}
	return methods
}

func (self *PluginsMgr) GetAdminTheme() (*PluginApi, *ThemesApi, error) {
	cfg, err := config.ReadThemesConfig()
	if err != nil {
		return nil, nil, err
	}

	pkg := cfg.AdminThemePkg
	p, ok := self.FindByPkg(pkg)
	if !ok {
		return nil, nil, fmt.Errorf("admin theme plugin '%s' is not installed", pkg)
	}

	themeApi := p.Themes().(*ThemesApi)
	if themeApi.AdminTheme == nil {
		return nil, nil, fmt.Errorf("plugin '%s' doesn't implement theme API", pkg)
	}

	return p.(*PluginApi), themeApi, nil
}

func (self *PluginsMgr) GetPortalTheme() (*PluginApi, *ThemesApi, error) {
	cfg, err := config.ReadThemesConfig()
	if err != nil {
		return nil, nil, err
	}

	pkg := cfg.PortalThemePkg
	p, ok := self.FindByPkg(pkg)
	if !ok {
		return nil, nil, fmt.Errorf("portal theme plugin '%s' is not installed", pkg)
	}

	themeApi := p.Themes().(*ThemesApi)
	if themeApi.PortalTheme == nil {
		return nil, nil, fmt.Errorf("plugin '%s' doesn't implement theme API", pkg)
	}

	return p.(*PluginApi), themeApi, nil
}
