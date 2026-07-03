package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	sdkapi "sdk/api"
)

// ForgotPasswordRoutes registers the core-owned forgot-password OTP pages on the
// core's HTTPS-only plugin router (the flow carries an OTP + a new password, so it
// must never be reachable over plain HTTP). Their names resolve via UrlForRoute
// against the core package (com.flarego.core); a portal theme's login page links
// here via UrlForPkgRoute("com.flarego.core", …).
//
// No auth (this is the pre-auth password-recovery flow). The reset-password step is
// gated by CheckIfValidOtp so it is unreachable without a verified OTP.
func ForgotPasswordRoutes(g *api.CoreGlobals) {
	httpR := g.CoreAPI.HttpAPI.Router().HttpRouter(&sdkapi.HttpRouterOpts{HttpsOnly: true})
	otpGuard := middlewares.CheckIfValidOtp(g.CoreAPI)

	httpR.Get("/send-otp", controllers.SendOtpView(g)).Name("auth:send-otp")
	httpR.Post("/send-otp", controllers.SendOtpCtrl(g))
	httpR.Get("/verify-otp", controllers.VerifyOtpView(g)).Name("auth:verify-otp")
	httpR.Post("/verify-otp", controllers.VerifyOtpCtrl(g))
	httpR.Get("/reset-password", controllers.ResetPasswordView(g), otpGuard).Name("auth:reset-password")
	httpR.Post("/reset-password", controllers.ResetPasswordCtrl(g), otpGuard)
}
