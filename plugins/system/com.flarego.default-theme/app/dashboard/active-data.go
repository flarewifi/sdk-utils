package dashboard

import (
	"context"
	"fmt"

	sdkapi "sdk/api"

	"com.flarego.default-theme/app/utils"
	"com.flarego.default-theme/db/queries"
)

// ActiveUsersData holds real-time active user metrics for the dashboard.
type ActiveUsersData struct {
	ConnectedToday  int64
	SessionsToday   int64
	AvgSessionToday string
	PeakToday       int64
}

// GetActiveUsersDataToday queries today's connected device count, session count,
// and average session duration. On every call it also records the current
// connected count as a candidate peak for today, keeping the maximum seen.
// Each metric defaults to 0 on error.
func GetActiveUsersDataToday(api sdkapi.IPluginApi, ctx context.Context) ActiveUsersData {
	db := queries.New(api.SqlDB())

	connected, err := db.GetConnectedUsersToday(ctx)
	if err != nil {
		api.Logger().Error("Failed to get connected users today: " + err.Error())
		connected = 0
	}

	// Record current connected count as a peak candidate.
	if connected > 0 {
		if upsertErr := db.UpsertPeakUsersToday(ctx, connected); upsertErr != nil {
			api.Logger().Error("Failed to upsert peak users today: " + upsertErr.Error())
		}
	}

	sessionsToday, err := db.GetSessionsCountToday(ctx)
	if err != nil {
		api.Logger().Error("Failed to get today's session count: " + err.Error())
		sessionsToday = 0
	}

	avgRaw, err := db.GetAvgSessionSecsToday(ctx)
	if err != nil {
		api.Logger().Error("Failed to get avg session secs today: " + err.Error())
		avgRaw = 0
	}

	peakToday, err := db.GetPeakUsersToday(ctx)
	if err != nil {
		api.Logger().Error("Failed to get peak users today: " + err.Error())
		peakToday = 0
	}

	avgHours := utils.ToFloat64(avgRaw) / 3600.0

	return ActiveUsersData{
		ConnectedToday:  connected,
		SessionsToday:   sessionsToday,
		AvgSessionToday: fmt.Sprintf("%.1fh", avgHours),
		PeakToday:       peakToday,
	}
}
