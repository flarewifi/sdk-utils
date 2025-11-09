package api

import (
	"core/db"
	"core/db/models"
	"core/internal/connmgr"
	"core/internal/network"
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
	ClientRegister *connmgr.ClientRegister
	ClientMgr      *connmgr.SessionsMgr
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
	clntReg := connmgr.NewClientRegister(db, mdls)
	clntMgr := connmgr.NewSessionsMgr(db, mdls)
	trfcMgr := network.NewTrafficMgr()
	pmtMgr := NewPaymentMgr()

	clntReg.SetSessionsMgr(clntMgr)

	trfcMgr.Start()
	clntMgr.ListenTraffic(trfcMgr)

	plgnMgr := NewPluginMgr(db, mdls, pmtMgr, clntReg, clntMgr, trfcMgr)
	coreApi := NewPluginApi(sdkutils.PathCoreDir, info, assets, plgnMgr, trfcMgr)
	plgnMgr.InitCoreApi(coreApi)

	return &CoreGlobals{
		assets,
		db,
		state,
		coreApi,
		clntReg,
		clntMgr,
		trfcMgr,
		mdls,
		plgnMgr,
		pmtMgr,
	}
}
