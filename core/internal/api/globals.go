package api

import (
	"core/db"
	"core/db/models"
	"core/internal/modules/ubus"
	"core/internal/network"
	"core/internal/sessmgr"
	"sync/atomic"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type AppState struct {
	NeedsRestart atomic.Bool
}

type CoreGlobals struct {
	GlobalAssets   *GlobalAssets
	Database       *db.Database
	State          *AppState
	CoreAPI        *PluginApi
	ClientRegister *sessmgr.ClientRegister
	ClientMgr      *sessmgr.SessionsMgr
	TrafficMgr     *network.TrafficMgr
	WifiMgr        *ubus.WifiMgr
	Models         *models.Models
	PluginMgr      *PluginsMgr
	PaymentsMgr    *PaymentsMgr
}

func NewGlobals() *CoreGlobals {
	state := &AppState{}

	info, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		panic(err)
	}

	assets := &GlobalAssets{}
	db := db.NewDatabase()
	mdls := models.New(db)
	clntReg := sessmgr.NewClientRegister(db, mdls)
	sessionMgr := sessmgr.NewSessionsMgr(db, mdls)
	trfcMgr := network.NewTrafficMgr()
	wifiMgr := ubus.NewWifiMgr()
	pmtMgr := NewPaymentMgr()

	clntReg.SetSessionsMgr(sessionMgr)
	trfcMgr.Start()

	// Set traffic channel for WiFi fallback detection before starting
	wifiMgr.SetTrafficChannel(trfcMgr.Listen())
	wifiMgr.Start()

	sessionMgr.ListenTraffic(trfcMgr)

	plgnMgr := NewPluginMgr(db, mdls, pmtMgr, clntReg, sessionMgr, trfcMgr)
	coreApi := NewPluginApi(sdkutils.PathCoreDir, info, assets, plgnMgr, trfcMgr, wifiMgr)
	plgnMgr.InitCoreApi(coreApi)
	sessionMgr.SetCoreAPI(coreApi)

	g := &CoreGlobals{
		GlobalAssets:   assets,
		Database:       db,
		State:          state,
		CoreAPI:        coreApi,
		ClientRegister: clntReg,
		ClientMgr:      sessionMgr,
		TrafficMgr:     trfcMgr,
		WifiMgr:        wifiMgr,
		Models:         mdls,
		PluginMgr:      plgnMgr,
		PaymentsMgr:    pmtMgr,
	}

	return g
}
