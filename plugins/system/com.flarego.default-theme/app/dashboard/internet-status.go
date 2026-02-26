package dashboard

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
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
// throughput by sampling the WAN interface rx/tx byte counters over a short
// interval.
func GetInternetStatus(api sdkapi.IPluginApi, ctx context.Context) InternetStatusData {
	data := InternetStatusData{}

	latencyMs, ok := measureLatency()
	data.Connected = ok
	data.LatencyMs = latencyMs

	down, up := measureThroughput(api, ctx)
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

// measureThroughput samples WAN interface byte counters before and after a
// short sleep, then derives Mbps. It picks the first non-loopback device
// whose name starts with "eth" or "wan" as the WAN interface heuristic.
func measureThroughput(api sdkapi.IPluginApi, ctx context.Context) (downloadMbps, uploadMbps float64) {
	devices, err := api.Network().ListDevices()

	if err != nil || len(devices) == 0 {
		return 0, 0
	}

	// Find the WAN device: prefer "wan", then "eth0", then first non-loopback.
	var wan sdkapi.INetworkDevice
	for _, d := range devices {
		name := d.Name()
		if name == "wan" || strings.HasPrefix(name, "wan") {
			wan = d
			break
		}
	}
	if wan == nil {
		for _, d := range devices {
			name := d.Name()
			if strings.HasPrefix(name, "eth") {
				wan = d
				break
			}
		}
	}
	if wan == nil {
		for _, d := range devices {
			if d.Name() != "lo" {
				wan = d
				break
			}
		}
	}
	if wan == nil {
		return 0, 0
	}

	rxBefore := wan.RxBytes()
	txBefore := wan.TxBytes()

	const defaultMs = 500
	select {
	case <-time.After(defaultMs * time.Millisecond):
	case <-ctx.Done():
		return 0, 0
	}

	// Re-fetch device to get updated counters.
	wan2, err := api.Network().GetDevice(wan.Name())
	if err != nil {
		return 0, 0
	}
	rxAfter := wan2.RxBytes()
	txAfter := wan2.TxBytes()

	elapsed := float64(defaultMs) / 1000.0 // seconds
	rxDelta := float64(rxAfter-rxBefore) * 8 / 1e6 / elapsed
	txDelta := float64(txAfter-txBefore) * 8 / 1e6 / elapsed

	return formatMbps(rxDelta), formatMbps(txDelta)
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
