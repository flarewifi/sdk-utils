package dashboard

import (
	"fmt"
	"net"
	"strconv"
	"time"

	sdkapi "sdk/api"
)

// InternetStatusData holds real-time WAN connectivity metrics.
type InternetStatusData struct {
	Connected    bool
	DownloadMbps float64
	UploadMbps   float64
	LatencyMs    int64
}

// dialTarget is the host:port used to measure WAN latency and reachability.
// We dial DNS (port 53) on a well-known public resolver — no external binary needed.
const dialTarget = "8.8.8.8:53"

// dialTimeout is the maximum time allowed for a single TCP connectivity probe.
const dialTimeout = 2 * time.Second

// GetInternetStatus checks WAN connectivity via a TCP dial and measures
// throughput using the SDK's rate tracking methods.
func GetInternetStatus(api sdkapi.IPluginApi) InternetStatusData {
	data := InternetStatusData{}

	latencyMs, ok := measureLatency()
	data.Connected = ok
	data.LatencyMs = latencyMs

	down, up := measureThroughput(api)
	data.DownloadMbps = down
	data.UploadMbps = up

	return data
}

// measureLatency dials dialTarget over TCP and returns the round-trip time in
// milliseconds. Returns (0, false) when the host is unreachable.
func measureLatency() (int64, bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", dialTarget, dialTimeout)
	if err != nil {
		return 0, false
	}
	conn.Close()
	return time.Since(start).Milliseconds(), true
}

// measureThroughput returns the current WAN download/upload rates in Mbps.
// Uses the SDK's RxRate()/TxRate() methods which track rate across calls.
func measureThroughput(api sdkapi.IPluginApi) (downloadMbps, uploadMbps float64) {
	wanIface, err := api.Network().GetWanInterface()
	if err != nil {
		return 0, 0
	}

	wanDevice, err := wanIface.Device()
	if err != nil {
		return 0, 0
	}

	// Use SDK rate methods - no blocking!
	rxRate := wanDevice.RxRate() // bytes/sec
	txRate := wanDevice.TxRate() // bytes/sec

	// Convert to Mbps (bits per second / 1,000,000)
	downloadMbps = float64(rxRate) * 8 / 1e6
	uploadMbps = float64(txRate) * 8 / 1e6

	return formatMbps(downloadMbps), formatMbps(uploadMbps)
}

// formatMbps clamps negative values (counter wrap) to 0 and rounds to 1dp.
func formatMbps(v float64) float64 {
	if v < 0 {
		v = 0
	}
	// Round to 1 decimal place.
	parsed, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", v), 64)
	return parsed
}
