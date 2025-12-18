//go:build dev

package nftables

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	jobque "core/tools/job-que"
)

var (
	nftStatsQue      = jobque.NewJobQue[StatResult]()
	mockStatsMu      sync.RWMutex
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
func GetStats() (stat StatResult, err error) {
	result, err := nftStatsQue.Exec(func() (result StatResult, err error) {
		mockStatsMu.Lock()
		defer mockStatsMu.Unlock()

		now := time.Now()
		macStats := make(map[string]StatData)
		ipStats := make(map[string]StatData)

		// Get list of currently connected clients
		nftMu.RLock()
		connectedClients := make(map[string]*connectedClient)
		for mac, client := range connClients {
			connectedClients[mac] = client
		}
		nftMu.RUnlock()

		// Clean up disconnected clients from tracker
		for mac := range mockStatsTracker {
			if _, exists := connectedClients[mac]; !exists {
				delete(mockStatsTracker, mac)
			}
		}

		// Generate mock stats for each connected client
		for mac, client := range connectedClients {
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

		result = StatResult{
			MacStats: macStats,
			IpStats:  ipStats,
		}

		return result, nil
	})

	if err != nil {
		return StatResult{}, err
	}

	return result, nil
}
