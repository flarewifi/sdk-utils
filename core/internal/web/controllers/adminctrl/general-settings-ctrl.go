package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	"core/internal/utils/activation"
	machineuid "core/internal/utils/machine-uid"
	generalview "core/resources/views/admin/general"
	"tools/config"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func GeneralSettingsIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		cfg, err := config.ReadApplicationConfig()
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		// Get machine ID
		machineID := machineuid.GetMachineUID()

		// Get software version
		pluginInfo, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
		softwareVersion := "unknown"
		if err == nil {
			softwareVersion = pluginInfo.Version
		}

		// Get form errors if any
		errors := g.CoreAPI.HttpAPI.Forms().Errors(w, r, "general_settings")

		// Get activation status
		activationStatus := "not_activated"
		if activation.IsValidating.Load() {
			activationStatus = "validating"
		} else if activation.IsActivated.Load() {
			activationStatus = "activated"
		}

		// Get supported languages
		supportedLanguages := config.SupportedLanguages

		params := generalview.AdminGeneralSettingsIndexParams{
			Cfg:                cfg,
			MachineID:          machineID,
			SoftwareVersion:    softwareVersion,
			ActivationStatus:   activationStatus,
			Errors:             errors,
			SupportedLanguages: supportedLanguages,
		}
		page := generalview.AdminGeneralSettingsIndex(g.CoreAPI, params)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func GeneralSettingsSaveCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		// Define the form validator
		formValidator := sdkapi.FormWithValidator{
			FormName: "application_settings",
			FormValidators: []sdkapi.FormValidator{
				{
					FieldName:  "language",
					FieldLabel: g.CoreAPI.Translate("label", "Language"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{
						Required: true,
					},
				},
				{
					FieldName:  "currency",
					FieldLabel: g.CoreAPI.Translate("label", "Currency"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{
						Required: true,
					},
				},
			},
		}

		// Parse and validate the form
		err := g.CoreAPI.HttpAPI.Forms().ParseFormWithValidator(w, r, formValidator)
		if err != nil {
			// Validation failed, redirect back to the form
			applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
			http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
			return
		}

		// Get form values
		language := r.FormValue("language")
		currency := r.FormValue("currency")

		// Read current config to preserve the Secret field
		currentCfg, err := config.ReadApplicationConfig()
		if err != nil {
			saveErrorMsg := g.CoreAPI.Translate("error", "save_settings_error")
			res.FlashMsg(w, r, saveErrorMsg, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
			http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
			return
		}

		// Save the application config
		err = config.WriteApplicationConfig(sdkapi.AppConfig{
			Lang:     language,
			Currency: currency,
			Channel:  currentCfg.Channel,
			Secret:   currentCfg.Secret, // Preserve existing secret
		})
		if err != nil {
			saveErrorMsg := g.CoreAPI.Translate("error", "save_settings_error")
			res.FlashMsg(w, r, saveErrorMsg, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
			http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
			return
		}

		successfulSavedMsg := g.CoreAPI.Translate("info", "saved_settings_message")
		res.FlashMsg(w, r, successfulSavedMsg, sdkapi.FlashMsgSuccess)

		applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
		http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
	}
}
