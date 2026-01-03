package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	"core/internal/modules/activation"
	machineuid "core/internal/modules/machine-uid"
	generalview "core/resources/views/admin/general"
	"core/utils/config"
	"core/utils/env"
	"core/utils/flaretmpl"
	"core/utils/sysinfo"

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
		_, machineID := machineuid.GetMachineUID()

		// Get software version
		pluginInfo, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
		softwareVersion := "unknown"
		if err == nil {
			softwareVersion = pluginInfo.Version
		}

		// Get form errors if any
		errors := g.CoreAPI.HttpAPI.Forms().Errors(w, r, "application_settings")

		// Get system info
		systemInfo, err := sysinfo.GetSystemInfo()
		if err != nil {
			// If there's an error, provide empty/default system info
			systemInfo = &sysinfo.SystemInfo{}
		}

		// Get activation status
		activationStatus := "not_activated"
		if activation.IsValidating.Load() {
			activationStatus = "validating"
		} else if activation.IsActivated.Load() {
			activationStatus = "activated"
		}

		// Get supported languages
		supportedLanguages := config.SupportedLanguages

		// Get supported currencies
		supportedCurrencies := []sdkapi.SupportedCurrency{
			{Code: sdkapi.CurrencyUsDollar, Name: "US Dollar", Symbol: "$"},
			{Code: sdkapi.CurrencyPhilippinePeso, Name: "Philippine Peso", Symbol: "₱"},
			{Code: sdkapi.CurrencyNigerianNaira, Name: "Nigerian Naira", Symbol: "₦"},
		}

		params := generalview.AdminGeneralSettingsIndexParams{
			Cfg:                 cfg,
			MachineID:           machineID,
			SoftwareVersion:     softwareVersion,
			ActivationStatus:    activationStatus,
			Errors:              errors,
			SupportedLanguages:  supportedLanguages,
			SupportedCurrencies: supportedCurrencies,
			SystemInfo:          systemInfo,
		}
		page := generalview.AdminGeneralSettingsIndex(g.CoreAPI, params)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func GeneralSettingsSaveCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		// Define the form validator
		formValidator := sdkapi.FormValidator{
			Name: "application_settings",
			Validators: []sdkapi.FormFieldValidator{
				{
					FieldName:  "language",
					FieldLabel: g.CoreAPI.Translate("label", "Language"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true},
				},
				{
					FieldName:  "currency",
					FieldLabel: g.CoreAPI.Translate("label", "Currency"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true},
				},
			},
		}

		// Parse and validate the form
		formValues, err := g.CoreAPI.HttpAPI.Forms().ParseForm(w, r, formValidator)
		if err != nil {
			// Validation failed, redirect back to the form
			applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
			http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
			return
		}

		// Get form values
		language, _ := formValues.GetStringValue("language")
		currency, _ := formValues.GetStringValue("currency")

		// Read current config to preserve the Secret field
		currentCfg, err := config.ReadApplicationConfig()
		if err != nil {
			saveErrorMsg := g.CoreAPI.Translate("error", "Unable to Save Settings")
			res.FlashMsg(w, r, saveErrorMsg, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
			http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
			return
		}

		// Check if language changed
		languageChanged := currentCfg.Lang != language

		// Save the application config
		err = config.WriteApplicationConfig(sdkapi.AppConfig{
			Lang:     language,
			Currency: currency,
			Channel:  currentCfg.Channel,
			Secret:   currentCfg.Secret, // Preserve existing secret
		})
		if err != nil {
			saveErrorMsg := g.CoreAPI.Translate("error", "Unable to Save Settings")
			res.FlashMsg(w, r, saveErrorMsg, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
			http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
			return
		}

		// Handle language switching if language changed
		if languageChanged {
			if env.GO_ENV != env.ENV_DEV {
				// In production, physically switch language files
				if err := sdkutils.SwitchAllLanguages(currentCfg.Lang, language); err != nil {
					g.CoreAPI.LoggerAPI.Error("Failed to switch language: " + err.Error())
					// Don't fail the save operation, just log the error
				}
			}
			// Clear template cache in both dev and production to ensure new language translations are loaded
			flaretmpl.ClearCache()
			g.CoreAPI.LoggerAPI.Info("Language changed to " + language + ", template cache cleared")
		}

		successfulSavedMsg := g.CoreAPI.Translate("info", "Settings Successfully Saved")
		res.FlashMsg(w, r, successfulSavedMsg, sdkapi.FlashMsgSuccess)

		applicationSettingsIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:general:index")
		http.Redirect(w, r, applicationSettingsIndexUrl, http.StatusSeeOther)
	}
}
