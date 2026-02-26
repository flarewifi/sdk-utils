package dashboard

import (
	"context"
	"log"
	sdkapi "sdk/api"
	"time"

	"com.flarego.default-theme/app/utils"
	"com.flarego.default-theme/db/queries"
)

// RevenueChartPoint is one data point for the 7-day revenue chart.
type RevenueChartPoint struct {
	Label    string
	Coinslot float64
	Voucher  float64
}

// DashboardSalesData holds today's revenue summary for the dashboard.
type DashboardSalesData struct {
	TotalRevenue    float64
	CoinslotRevenue float64
	VoucherRevenue  float64
}

func GetSalesSummaryToday(api sdkapi.IPluginApi, ctx context.Context) DashboardSalesData {
	db := queries.New(api.SqlDB())

	row, err := db.GetDashboardSalesSummary(ctx)
	if err != nil {
		log.Printf("[DEBUG] Failed to get sales summary: " + err.Error())
		return DashboardSalesData{
			TotalRevenue:    0,
			CoinslotRevenue: 0,
			VoucherRevenue:  0,
		}
	}

	return DashboardSalesData{
		TotalRevenue:    utils.ToFloat64(row.TotalRevenue),
		CoinslotRevenue: utils.ToFloat64(row.CoinslotRevenue),
		VoucherRevenue:  utils.ToFloat64(row.VoucherRevenue),
	}
}

// GetRevenueChartData returns coinslot + voucher revenue for each of the last
// 7 days (oldest first). Days with no sales are included with zero values so
// the chart always has exactly 7 points.
func GetRevenueChartData(api sdkapi.IPluginApi, ctx context.Context) []RevenueChartPoint {
	db := queries.New(api.SqlDB())

	rows, err := db.GetRevenueChartLast7Days(ctx)
	if err != nil {
		log.Printf("[DEBUG] Failed to get revenue chart data: " + err.Error())
		rows = nil
	}

	// Index DB rows by date string "2006-01-02".
	// Day is interface{} from SQLite driver; assert to string.
	byDay := make(map[string]queries.GetRevenueChartLast7DaysRow, len(rows))
	for _, r := range rows {
		if key, ok := r.Day.(string); ok {
			byDay[key] = r
		}
	}

	now := time.Now()
	points := make([]RevenueChartPoint, 7)
	for i := 0; i < 7; i++ {
		day := now.AddDate(0, 0, i-6)
		key := day.Format("2006-01-02")
		label := day.Format("Jan 2")
		var coinslot, voucher float64
		if row, ok := byDay[key]; ok {
			coinslot = utils.ToFloat64(row.CoinslotRevenue)
			voucher = utils.ToFloat64(row.VoucherRevenue)
		}
		points[i] = RevenueChartPoint{Label: label, Coinslot: coinslot, Voucher: voucher}
	}

	return points
}
