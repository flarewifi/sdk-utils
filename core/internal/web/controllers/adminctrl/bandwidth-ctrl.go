package adminctrl

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"strconv"

// 	"core/internal/config/bandwdcfg"
// 	"core/internal/config/themecfg"
// 	"core/internal/fci"
// 	"core/internal/globals"
// 	"core/internal/network"
// 	"core/internal/plugins"
// 	"github.com/flarehotspot/sdk-utils/flash"
// 	"github.com/flarehotspot/sdk-utils/slices"
// 	"core/internal/utils/ubus"
// 	"core/internal/web/response"
// 	"core/internal/web/router"
// 	"core/internal/web/routes/names"
// 	"github.com/gorilla/mux"
// )

// type BandwidthCtrl struct {
// 	g           *globals.CoreGlobals
// 	capi        *plugins.PluginApi
// 	errRedirect *response.ErrRedirect
// }

// func NewBandwidthCtrl(g *globals.CoreGlobals, api *plugins.PluginApi) *BandwidthCtrl {
// 	redirect := response.NewErrRoute(routenames.RouteAdminBandwidthIndex)
// 	return &BandwidthCtrl{g, api, redirect}
// }

// func (self *BandwidthCtrl) Index(w http.ResponseWriter, r *http.Request) {
// 	ifnames, err := ubus.GetInterfaceNames()
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	cfg, err := bandwdcfg.Read()
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	configFor := func(ifname string) *bandwdcfg.IfCfg {
// 		bwd, ok := cfg.Lans[ifname]
// 		if !ok {
// 			return &bandwdcfg.IfCfg{}
// 		}
// 		return bwd
// 	}

// 	saveUrlFor := func(ifname string) string {
// 		url, _ := router.UrlForRoute(routenames.RouteAdminBandwidthSave, "ifname", ifname)
// 		return url
// 	}

// 	ifnames = slices.Filter(ifnames, func(ifname string) bool {
// 		return ifname != "loopback" && ifname != "wan"
// 	})

// 	data := map[string]any{
// 		"ifnames":    ifnames,
// 		"configFor":  configFor,
// 		"saveUrlFor": saveUrlFor,
// 	}

// 	self.capi.HttpApi().Respond().AdminView(w, r, "bandwidth/index.html", data)
// }

// func (self *BandwidthCtrl) Save(w http.ResponseWriter, r *http.Request) {
// 	var err error

// 	params := mux.Vars(r)
// 	ifname := params["ifname"]

// 	err = r.ParseForm()
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	updateExisting := r.PostFormValue("update_existing_users") == "1"
// 	useGlobal := r.PostFormValue("use_global") == "1"
// 	gDownMbitsStr := r.PostFormValue("global_down_mbits")
// 	gUpMbitsStr := r.PostFormValue("global_up_mbits")
// 	usrDownMbitsStr := r.PostFormValue("user_down_mbits")
// 	usrUpMbitsStr := r.PostFormValue("user_up_mbits")

// 	var gDownMbits, gUpMbits, usrDownMbits, usrUpMbits int

// 	gDownMbits, err = strconv.Atoi(gDownMbitsStr)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	gUpMbits, err = strconv.Atoi(gUpMbitsStr)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	usrUpMbits, err = strconv.Atoi(usrUpMbitsStr)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	usrDownMbits, err = strconv.Atoi(usrDownMbitsStr)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	cfg := bandwdcfg.IfCfg{
// 		UseGlobal:       useGlobal,
// 		GlobalDownMbits: gDownMbits,
// 		GlobalUpMbits:   gUpMbits,
// 		UserDownMbits:   usrDownMbits,
// 		UserUpMbits:     usrUpMbits,
// 	}

// 	bwd, err := bandwdcfg.Read()
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	prevCfg, ok := bwd.Lans[ifname]
// 	if !ok {
// 		prevCfg = &bandwdcfg.IfCfg{}
// 	}

// 	bwd.Lans[ifname] = &cfg

// 	err = bandwdcfg.Save(bwd)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	lan, err := network.FindByName(ifname)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	// Do not decrease existing global bandwidth
// 	// becasuse it will affect existing sessions
// 	if gDownMbits < prevCfg.GlobalDownMbits {
// 		gDownMbits = prevCfg.GlobalDownMbits
// 	}

// 	if gUpMbits < prevCfg.GlobalUpMbits {
// 		gUpMbits = prevCfg.GlobalUpMbits
// 	}

// 	err = lan.UpdateBandwidth(gDownMbits, gUpMbits)
// 	if err != nil {
// 		self.Error(w, r, err)
// 		return
// 	}

// 	if updateExisting {
// 		err = self.updateRunningSessions(r.Context(), ifname, usrDownMbits, usrUpMbits, useGlobal)
// 		if err != nil {
// 			self.Error(w, r, err)
// 			return
// 		}
// 	}

// 	flash.SetFlashMsg(w, flash.Success, "Bandwidth settings saved successfully.")
// 	http.Redirect(w, r, self.indexUrl(), http.StatusSeeOther)
// }

// func (self *BandwidthCtrl) Error(w http.ResponseWriter, r *http.Request, err error) {
// 	self.errRedirect.Redirect(w, r, err)
// }

// func (self *BandwidthCtrl) indexUrl() string {
// 	url, _ := router.UrlForRoute(routenames.RouteAdminBandwidthIndex)
// 	return url
// }

// func (self *BandwidthCtrl) updateRunningSessions(ctx context.Context, ifname string, downMbits, upMbits int, useGlobal bool) error {
// 	err := self.g.Models.Session().UpdateAllBandwidth(ctx, downMbits, upMbits, useGlobal)
// 	if err != nil {
// 		return err
// 	}

// 	return self.g.ClientMgr.ReloadSessions(ctx, ifname)
// }

// func (self *BandwidthCtrl) Test(w http.ResponseWriter, r *http.Request) {
// 	pkg := themecfg.Read().Admin
// 	cfg := fci.NewFciConfig(pkg, "bandwidth-test")
// 	sec := cfg.Section("test section", "test description")
// 	f := sec.Field("test field", "test label", "test help")
// 	f.SetAttr("type", "text")
// 	f.SetAttr("value", "some value")
// 	f.SetAttr("placeholder", "some placeholder")

// 	fl := sec.FieldList("some_field_list", "test label")
// 	fl.Cols("col1", "col2", "col3")
// 	row := fl.Row(0)
// 	f = row.Field("col1", "name01")
// 	f.SetAttr("type", "text")

// 	html, err := fci.FciComposeView(cfg)
// 	if err != nil {
// 		// self.Error(w, r, err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	fmt.Fprint(w, html)
// }
