package api

import (
	"net/http"
	"sync/atomic"

	"core/db"
	"core/db/models"
	"core/internal/events"
	"core/internal/modules/bootprogress"
	"core/internal/modules/scheduler"
	"core/internal/modules/ubus"
	"core/internal/network"
	"core/internal/sessmgr"

	sdkutils "github.com/flarewifi/sdk-utils"
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
	EventsMgr      *events.EventsManager
	SchedulerMgr   *scheduler.Manager
	BootProgress   *bootprogress.Tracker

	// AppServer is the final (post-boot) HTTP server handle, set once boot
	// completes (core/internal/boot/init-http.go). nil until then. A graceful
	// shutdown calls AppServer.Shutdown(ctx) on it; see core/main.go.
	AppServer *http.Server
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
	eventsMgr := events.NewEventsManager()
	clntReg := sessmgr.NewClientRegister(db, mdls)
	sessionMgr := sessmgr.NewSessionsMgr(db, mdls, eventsMgr)
	trfcMgr := network.NewTrafficMgr()
	wifiMgr := ubus.NewWifiMgr()
	pmtMgr := NewPaymentMgr()
	schedulerMgr := scheduler.NewManager()

	clntReg.SetSessionsMgr(sessionMgr)
	trfcMgr.Start()

	// Set traffic channel for WiFi fallback detection (Start() called in jobs.Init())
	wifiMgr.SetTrafficChannel(trfcMgr.Listen())

	sessionMgr.ListenTraffic(trfcMgr)

	plgnMgr := NewPluginMgr(db, mdls, pmtMgr, clntReg, sessionMgr, trfcMgr, eventsMgr, schedulerMgr)
	coreApi := NewPluginApi(sdkutils.PathCoreDir, info, assets, plgnMgr, trfcMgr, wifiMgr)
	plgnMgr.InitCoreApi(coreApi)
	plgnMgr.SetDeps(assets, wifiMgr)
	sessionMgr.SetCoreAPI(coreApi)

	// schedulerMgr is constructed before coreApi exists (it's threaded through
	// NewPluginMgr so every plugin's PluginApi can reach it), so its logger can
	// only be wired up now that coreApi's own LoggerAPI is available.
	schedulerMgr.SetLogger(coreApi.Logger())

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
		EventsMgr:      eventsMgr,
		SchedulerMgr:   schedulerMgr,
		BootProgress:   bootprogress.New(),
	}

	return g
}
