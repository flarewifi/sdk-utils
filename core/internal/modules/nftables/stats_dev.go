//go:build dev

package nftables

import (
	"math/rand"
	"strings"
	"time"

	jobque "core/utils/job-que"
)

var (
	// nftStatsQue serializes GetStats calls, mirroring the production stats.go queue.
	// Using a separate queue (not nftQue) prevents GetStats from blocking behind
	// Connect/Disconnect operations, matching the production code path.
	nftStatsQue = jobque.NewJobQueue[StatResult]()

	// mockStatsTracker is protected by nftStatsQue serialization.
	mockStatsTracker = make(map[string]*mockClientStats)
	mockRand         = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// mockClientStats tracks cumulative stats for each connected client
type mockClientStats struct {
	totalBytes   uint
	totalPackets uint
	lastUpdate   time.Time
}

type StatData struct {
	Bytes   uint
	Packets uint
}

type StatResult struct {
	MacStats map[string]StatData
	IpStats  map[string]StatData
}

// GetStats generates mock traffic statistics for connected clients in dev mode.
// It simulates realistic data consumption by generating random traffic between 50KB-500KB per 5-second interval.
// Uses nftStatsQue for serialization, matching production behavior (separate from nftQue so
// GetStats does not block behind Connect/Disconnect operations).
// For dual-stack devices both IPv4 and IPv6 IPs receive the same simulated traffic totals.
func GetStats() (stat StatResult, err error) {
	return nftStatsQue.Exec("GetStats", func() (StatResult, error) {
		now := time.Now()
		macStats := make(map[string]StatData)
		ipStats := make(map[string]StatData)

		// mockStatsTracker and connTable are only ever accessed from within
		// nftStatsQue or nftQue jobs. Since these are separate single-worker
		// queues, both maps require a snapshot of connTable under nftMu before
		// iterating, so we don't race with doConnect/doDisconnect.
		nftMu.RLock()
		connectedMACs := make([]string, 0, len(macToIps))
		for mac := range macToIps {
			// Paused devices are disconnected from the internet, so they generate no
			// (accepted) traffic. Emit no stats for them — in production their
			// attempted upload is a real counter, but the dev mock has no real
			// packets to count, so reporting zero keeps a paused session paused
			// instead of self-resuming on fabricated traffic.
			if pausedTable[strings.ToUpper(mac)] {
				continue
			}
			connectedMACs = append(connectedMACs, mac)
		}
		nftMu.RUnlock()

		// Clean up disconnected clients from tracker
		for mac := range mockStatsTracker {
			found := false
			for _, m := range connectedMACs {
				if m == mac {
					found = true
					break
				}
			}
			if !found {
				delete(mockStatsTracker, mac)
			}
		}

		// Generate mock stats for each connected client
		for _, mac := range connectedMACs {
			// Initialize tracker if new client
			if _, exists := mockStatsTracker[mac]; !exists {
				mockStatsTracker[mac] = &mockClientStats{
					totalBytes:   0,
					totalPackets: 0,
					lastUpdate:   now,
				}
			}

			tracker := mockStatsTracker[mac]

			// Generate random traffic data (50KB to 500KB per interval)
			// This simulates realistic browsing/streaming behavior
			minBytes := uint(50 * 1024)  // 50 KB
			maxBytes := uint(500 * 1024) // 500 KB
			randomBytes := minBytes + uint(mockRand.Intn(int(maxBytes-minBytes)))

			// Packets: roughly 1 packet per 1500 bytes (typical MTU)
			randomPackets := randomBytes / 1500
			if randomPackets == 0 {
				randomPackets = 1
			}

			// Update cumulative totals
			tracker.totalBytes += randomBytes
			tracker.totalPackets += randomPackets
			tracker.lastUpdate = now

			// MAC stats (upload is keyed by uppercase MAC)
			macUpper := strings.ToUpper(mac)
			macStats[macUpper] = StatData{
				Bytes:   tracker.totalBytes,
				Packets: tracker.totalPackets,
			}

			// IP stats — emit an entry only for the primary IP (IPv4 preferred, else IPv6).
			// TrafficMgr aggregates download bytes by MAC, so emitting the same total on
			// multiple IPs would not double-count, but a single IP is sufficient here.
			nftMu.RLock()
			ips := macToIps[mac]
			nftMu.RUnlock()

			primaryAddr := ""
			for ip := range ips {
				if !strings.Contains(ip, ":") { // IPv4 has no colon
					primaryAddr = ip
					break
				}
			}
			if primaryAddr == "" {
				// IPv6-only device — use whichever IP is registered
				for ip := range ips {
					primaryAddr = ip
					break
				}
			}
			if primaryAddr != "" {
				ipStats[primaryAddr] = StatData{
					Bytes:   tracker.totalBytes,
					Packets: tracker.totalPackets,
				}
			}
		}

		return StatResult{
			MacStats: macStats,
			IpStats:  ipStats,
		}, nil
	})
}
