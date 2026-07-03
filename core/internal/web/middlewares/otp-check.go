package middlewares

import (
	"net/http"

	sdkapi "sdk/api"
)

// OtpVerifiedCookie marks that the caller passed the OTP step and may proceed to
// set a new password. Set on a successful VerifyOtp, cleared once the password is
// changed (see the forgot-password controllers).
const OtpVerifiedCookie = "otp_verified"

// CheckIfValidOtp gates the reset-password step: it only proceeds when the caller
// holds the OtpVerifiedCookie (set after a successful VerifyOtp). Otherwise it sends
// them back to the login page — the new-password form is unreachable without first
// proving the OTP.
func CheckIfValidOtp(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := api.Http().Cookie().GetCookie(r, OtpVerifiedCookie)
			if err != nil || cookie != "true" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
