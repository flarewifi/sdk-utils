package adminctrl

// import (
// 	"net/http"

// 	"core/internal/globals"
// 	"core/internal/web/router"
// 	"core/internal/web/routes/names"
// )

// type DashboardCtrl struct {
// 	g *globals.CoreGlobals
// }

// func (self *DashboardCtrl) Index(w http.ResponseWriter, r *http.Request) {
// 	api := self.g.CoreApi
// 	data := map[string]any{
// 		"html": "<h3 style='color: red;'>Sample HTML</h3>",
// 	}
// 	api.HttpApi().Respond().AdminView(w, r, "dashboard.html", data)
// }

// func (self *DashboardCtrl) RedirectToDash(w http.ResponseWriter, r *http.Request) {
// 	adminURL, _ := router.UrlForRoute(routenames.RouteAdminDashboardIndex)
// 	http.Redirect(w, r, adminURL, http.StatusSeeOther)
// }

// func NewDashboardCtrl(g *globals.CoreGlobals) *DashboardCtrl {
// 	return &DashboardCtrl{g}
// }
