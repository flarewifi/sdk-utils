package adminctrl

import (
	"core/internal/plugins"
	"fmt"
	"net/http"
)

func NewFormsCtrl(g *plugins.CoreGlobals) *FormsCtrl {
	return &FormsCtrl{g}
}

type FormsCtrl struct {
	g *plugins.CoreGlobals
}

func (ctrl *FormsCtrl) SaveForm(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	pkg := r.URL.Query().Get("pkg")
	name := r.URL.Query().Get("name")

	plugin, ok := ctrl.g.CoreAPI.PluginsMgrApi.FindByPkg(pkg)
	if !ok {
		http.Error(w, fmt.Sprintf("Plugin %s not found", pkg), 404)
		return
	}

	p, ok := plugin.(*plugins.PluginApi)
	if !ok {
		http.Error(w, fmt.Sprintf("Plugin %s not found", pkg), 404)
		return
	}

	form, err := p.HttpAPI.Forms().GetForm(name)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	httpForm, ok := form.(*plugins.HttpFormInstance)
	if !ok {
		http.Error(w, fmt.Sprintf("Form %s not found", name), 404)
		return
	}

	err = httpForm.SaveForm(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	url := httpForm.GetRedirectUrl()
	http.Redirect(w, r, url, http.StatusSeeOther)
	// TODO: redirect back to the form page and show success message if redirect url is empty
}
