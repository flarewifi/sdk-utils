package controllers

import (
	"log"
	"net/http"
	"strings"

	"core/internal/api"
	corerpc "core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/internal/web/middlewares"
	authview "core/resources/views/auth"
	sdkapi "sdk/api"
)

// Forgot-password OTP flow, owned by the CORE. The machine's admin resets the local
// admin password by proving control of the email linked to this machine: request a
// one-time code (emailed by the cloud), verify it, then set a new password. The OTP
// business logic + email live in the cloud flarehotspot.v3 service; these handlers
// are the on-device UI that drives it via corerpc.GetTwirpServiceAndCtx().
//
// Previously each portal theme shipped its own copy of this flow; it now lives here
// once and every theme's login page just links to it (LoginPageData.ForgotPasswordUrl).

// forgotPwAssets are the core-owned per-page JS/CSS (resolved against core's manifest,
// while the surrounding layout comes from the active portal theme).
var forgotPwAssets = sdkapi.ViewAssets{
	JsFile:  "forgot-password.js",
	CssFile: "forgot-password.css",
}

// SendOtpView renders the "enter your email" page, seeding the resend countdown and
// masked-email hint from the current timer status.
func SendOtpView(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cooldown, maskedEmail := otpTimerStatus(g)
		csrfHTML := g.CoreAPI.HttpAPI.Helpers().CsrfHtmlTag(r)
		g.CoreAPI.HttpAPI.Response().PortalView(w, r, sdkapi.ViewPage{
			Assets:      forgotPwAssets,
			PageContent: authview.SendOtpPage(g.CoreAPI, csrfHTML, cooldown, maskedEmail),
		})
	}
}

// SendOtpCtrl requests a fresh OTP for the typed email. On success it advances to the
// verify page; otherwise it flashes the reason and stays on the send page.
func SendOtpCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		email := strings.TrimSpace(r.FormValue("email"))
		machineID := g.CoreAPI.Machine().GetID()

		srv, ctx := corerpc.GetTwirpServiceAndCtx()
		resp, err := srv.GenerateOtp(ctx, &rpc_flarewifi_v3.GenerateOtpRequest{
			MachineId: machineID,
			Email:     email,
		})
		if err != nil {
			log.Printf("[forgot-password] GenerateOtp error: %v", err)
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Failed to send OTP. Please try again later"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:send-otp")
			return
		}

		if !resp.GetSuccess() {
			res.FlashMsg(w, r, resp.GetMessage(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:send-otp")
			return
		}

		res.FlashMsg(w, r, g.CoreAPI.Translate("success", "OTP sent to your linked email"), sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "auth:verify-otp")
	}
}

// VerifyOtpView renders the "enter the code" page, gating the resend link with the
// current cooldown.
func VerifyOtpView(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cooldown, _ := otpTimerStatus(g)
		csrfHTML := g.CoreAPI.HttpAPI.Helpers().CsrfHtmlTag(r)
		g.CoreAPI.HttpAPI.Response().PortalView(w, r, sdkapi.ViewPage{
			Assets:      forgotPwAssets,
			PageContent: authview.VerifyOtpPage(g.CoreAPI, csrfHTML, cooldown),
		})
	}
}

// VerifyOtpCtrl checks the submitted code. On success it sets the otp_verified cookie
// (the gate for the reset-password step) and advances; otherwise it flashes the reason.
func VerifyOtpCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		otpCode := strings.TrimSpace(r.FormValue("otp"))
		machineID := g.CoreAPI.Machine().GetID()

		srv, ctx := corerpc.GetTwirpServiceAndCtx()
		resp, err := srv.VerifyOtp(ctx, &rpc_flarewifi_v3.VerifyOtpRequest{
			OtpCode:   otpCode,
			MachineId: machineID,
		})
		if err != nil {
			log.Printf("[forgot-password] VerifyOtp error: %v", err)
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Verification failed. Please try again"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:verify-otp")
			return
		}

		if !resp.GetSuccess() {
			res.FlashMsg(w, r, resp.GetMessage(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:verify-otp")
			return
		}

		// Proven: allow the reset-password step (CheckIfValidOtp reads this cookie).
		g.CoreAPI.HttpAPI.Cookie().SetCookie(w, middlewares.OtpVerifiedCookie, "true", nil)
		res.FlashMsg(w, r, g.CoreAPI.Translate("success", "OTP verified"), sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "auth:reset-password")
	}
}

// ResetPasswordView renders the new-password form (reached only past CheckIfValidOtp).
func ResetPasswordView(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		csrfHTML := g.CoreAPI.HttpAPI.Helpers().CsrfHtmlTag(r)
		g.CoreAPI.HttpAPI.Response().PortalView(w, r, sdkapi.ViewPage{
			Assets:      forgotPwAssets,
			PageContent: authview.NewPasswordPage(g.CoreAPI, csrfHTML),
		})
	}
}

// ResetPasswordCtrl applies the new local admin password, then clears the OTP gate,
// signs the session out, and returns to login.
func ResetPasswordCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		if len(newPassword) < 6 {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Password must be at least 6 characters"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:reset-password")
			return
		}
		if newPassword != confirmPassword {
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Passwords do not match"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:reset-password")
			return
		}

		// Find("admin") targets the reserved super-admin account (GetAll()[0] is not
		// guaranteed to be it — accounts are sorted alphabetically).
		account, err := g.CoreAPI.Acct().Find("admin")
		if err != nil || account == nil {
			log.Printf("[forgot-password] admin account lookup failed: %v", err)
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Account lookup failed"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:reset-password")
			return
		}

		if err := account.Update(account.Username(), newPassword, account.Permissions()); err != nil {
			log.Printf("[forgot-password] password update failed: %v", err)
			res.FlashMsg(w, r, g.CoreAPI.Translate("error", "Error changing password"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "auth:reset-password")
			return
		}

		// One-time gate — clear it so the reset form can't be replayed, and drop any
		// authenticated session so the admin re-logs in with the new password.
		g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, middlewares.OtpVerifiedCookie)
		if err := g.CoreAPI.HttpAPI.Auth().SignOut(w); err != nil {
			log.Printf("[forgot-password] sign out after reset failed: %v", err)
		}

		g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, g.CoreAPI.Translate("success", "Password successfully updated"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// =============================================================================
// HELPERS
// =============================================================================

// otpTimerStatus fetches the current resend cooldown (seconds) and masked-email hint
// from the cloud. Any transport error degrades gracefully to "no cooldown, no hint"
// so the page always renders.
func otpTimerStatus(g *api.CoreGlobals) (cooldownSeconds int64, maskedEmail string) {
	machineID := g.CoreAPI.Machine().GetID()
	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	resp, err := srv.GetOtpTimerStatus(ctx, &rpc_flarewifi_v3.GetOtpTimerStatusRequest{MachineId: machineID})
	if err != nil {
		log.Printf("[forgot-password] GetOtpTimerStatus error: %v", err)
		return 0, ""
	}
	if !resp.GetHasActiveTimer() {
		return 0, resp.GetMaskedEmail()
	}
	return resp.GetRemainingSeconds(), resp.GetMaskedEmail()
}
