//go:build dev

package nftables

import (
	"math/rand"
	"strings"
	"time"
)

var (
	// mockStatsTracker is protected by nftQue serialization (same queue as Connect/Disconnect)
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
// Uses nftQue for serialization (same queue as Connect/Disconnect/IsConnected).
func GetStats() (stat StatResult, err error) {
	result, err := nftQue.Exec("GetStats", func() (any, error) {
		now := time.Now()
		macStats := make(map[string]StatData)
		ipStats := make(map[string]StatData)

		// Clean up disconnected clients from tracker
		for mac := range mockStatsTracker {
			if _, exists := connClients[mac]; !exists {
				delete(mockStatsTracker, mac)
			}
		}

		// Generate mock stats for each connected client
		for mac, client := range connClients {
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

			// Add to result maps
			macUpper := strings.ToUpper(mac)
			macStats[macUpper] = StatData{
				Bytes:   tracker.totalBytes,
				Packets: tracker.totalPackets,
			}

			// Use the actual IP from the connected client
			ipStats[client.ip] = StatData{
				Bytes:   tracker.totalBytes,
				Packets: tracker.totalPackets,
			}
		}

		return StatResult{
			MacStats: macStats,
			IpStats:  ipStats,
		}, nil
	})

	if err != nil {
		return StatResult{}, err
	}

	return result.(StatResult), nil
}
