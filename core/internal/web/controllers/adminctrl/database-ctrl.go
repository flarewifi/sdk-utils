package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	"core/internal/utils/database"
	databaseview "core/resources/views/admin/database"
	"tools/config"
)

func DatabaseSettingsIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		cfg, err := config.ReadDatabaseConfig()
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		// Get form errors if any
		errors := g.CoreAPI.HttpAPI.Forms().Errors(w, r, "database_settings")

		params := databaseview.AdminDatabaseSettingsIndexParams{
			Cfg:    cfg,
			Errors: errors,
		}
		page := databaseview.AdminDatabaseSettingsIndex(g.CoreAPI, params)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func DatabaseResetCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		// Parse form to get confirmation
		if err := r.ParseForm(); err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Invalid request"), sdkapi.FlashMsgError)
			http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:database:index"), http.StatusSeeOther)
			return
		}

		// Check confirmation checkbox
		confirmed := r.FormValue("confirm_reset")
		if confirmed != "yes" {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Please confirm database reset"), sdkapi.FlashMsgError)
			http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:database:index"), http.StatusSeeOther)
			return
		}

		// Perform database reset with plugin migrations callback
		newDB, err := database.ResetDatabase(g.Database.DB, g.PluginMgr.RerunPluginMigrations)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error("Database reset failed: " + err.Error())
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to reset database"), sdkapi.FlashMsgError)
			http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:database:index"), http.StatusSeeOther)
			return
		}

		// Reopen database connection with new DB
		g.Database.ReopenConnection(newDB)

		g.CoreAPI.LoggerAPI.Info("Database reset completed successfully")
		res.FlashMsg(w, r, g.CoreAPI.Translate("success", "Database reset successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:database:index"), http.StatusSeeOther)
	}
}
