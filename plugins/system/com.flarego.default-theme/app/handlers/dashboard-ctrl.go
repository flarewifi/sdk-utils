package handlers

import (
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/app/dashboard"
	"com.flarego.default-theme/resources/views/admin"
)

func DashboardSalesCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sales := dashboard.GetSalesSummaryToday(api, ctx)
		view := admin.SalesSummary(api, sales)
		view.Render(r.Context(), w)
	}
}

func DashboardActiveDataCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := dashboard.GetActiveUsersDataToday(api, r.Context())
		view := admin.ActiveUsersCard(api, data)
		view.Render(r.Context(), w)
	}
}

func DashboardInternetStatusCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := dashboard.GetInternetStatus(api)
		view := admin.InternetStatusCard(api, data)
		view.Render(r.Context(), w)
	}
}

func DashboardRevenueChartCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := dashboard.GetRevenueChartData(api, r.Context())
		view := admin.RevenueChartCard(api, data)
		view.Render(r.Context(), w)
	}
}
