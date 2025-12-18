package api

import (
	"core/db"
	"core/db/models"
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
	pmtMgr := NewPaymentMgr()

	clntReg.SetSessionsMgr(sessionMgr)

	trfcMgr.Start()
	sessionMgr.ListenTraffic(trfcMgr)

	plgnMgr := NewPluginMgr(db, mdls, pmtMgr, clntReg, sessionMgr, trfcMgr)
	coreApi := NewPluginApi(sdkutils.PathCoreDir, info, assets, plgnMgr, trfcMgr)
	plgnMgr.InitCoreApi(coreApi)
	sessionMgr.SetCoreAPI(coreApi)

	return &CoreGlobals{
		assets,
		db,
		state,
		coreApi,
		clntReg,
		sessionMgr,
		trfcMgr,
		mdls,
		plgnMgr,
		pmtMgr,
	}
}
