package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	userview "core/resources/views/admin/user"
)

func AdminUserIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		acct, err := g.CoreAPI.HttpAPI.Auth().CurrentAcct(r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		errors := g.CoreAPI.HttpAPI.Forms().Errors(w, r, "change_password")

		params := userview.AdminUserIndexParams{
			Username: acct.Username(),
			Errors:   errors,
		}

		page := userview.AdminUserIndex(g.CoreAPI, params)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func AdminUserClearHistoryCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		indexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:user:index")

		ctx := r.Context()

		// Delete all fingerprints
		if err := g.Models.DeviceFingerprint().DeleteAll(ctx); err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to clear device history"), sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		// Delete non-current MACs (keep only the latest in use)
		if err := g.Models.DeviceMac().DeleteNonCurrent(ctx); err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to clear device history"), sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		res.FlashMsg(w, r, g.CoreAPI.Translate("success", "Device history cleared successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, indexUrl, http.StatusSeeOther)
	}
}

func AdminUserChangePasswordCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		indexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:user:index")

		// Get the current account
		acct, err := g.CoreAPI.HttpAPI.Auth().CurrentAcct(r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		// Define the form validator
		formValidator := sdkapi.FormValidator{
			Name: "change_password",
			Validators: []sdkapi.FormFieldValidator{
				{
					FieldName:  "current_password",
					FieldLabel: g.CoreAPI.Translate("label", "Current Password"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true},
				},
				{
					FieldName:  "new_password",
					FieldLabel: g.CoreAPI.Translate("label", "New Password"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true, Minimum: "4"},
				},
				{
					FieldName:  "confirm_password",
					FieldLabel: g.CoreAPI.Translate("label", "Confirm Password"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true, Minimum: "4"},
				},
			},
		}

		// Parse and validate the form
		formValues, err := g.CoreAPI.HttpAPI.Forms().ParseForm(w, r, formValidator)
		if err != nil {
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		currentPassword, _ := formValues.GetStringValue("current_password")
		newPassword, _ := formValues.GetStringValue("new_password")
		confirmPassword, _ := formValues.GetStringValue("confirm_password")

		// Verify the current password is correct
		_, err = g.CoreAPI.HttpAPI.Auth().Authenticate(acct.Username(), currentPassword)
		if err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Current password is incorrect"), sdkapi.FlashMsgError)
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		// Verify new password matches confirmation
		if newPassword != confirmPassword {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "New passwords do not match"), sdkapi.FlashMsgError)
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		// Update the account password (keep same username and permissions)
		err = acct.Update(acct.Username(), newPassword, acct.Permissions())
		if err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to update password"), sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		// Re-authenticate to refresh the session after password change
		updatedAcct, err := g.CoreAPI.HttpAPI.Auth().Authenticate(acct.Username(), newPassword)
		if err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to re-authenticate"), sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		err = g.CoreAPI.HttpAPI.Auth().SignIn(w, updatedAcct)
		if err != nil {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to refresh session"), sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			http.Redirect(w, r, indexUrl, http.StatusSeeOther)
			return
		}

		res.FlashMsg(w, r, g.CoreAPI.Translate("success", "Password changed successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, indexUrl, http.StatusSeeOther)
	}
}
