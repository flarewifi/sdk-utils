package network

import (
	"log"
	"strings"
	"sync"
	"time"

	"core/internal/modules/nftables"
	sdkapi "sdk/api"
)

type TrafficMgr struct {
	mu        sync.RWMutex
	ticker    *time.Ticker
	listners  []chan sdkapi.TrafficData
	prevStats *nftables.StatResult
}

func NewTrafficMgr() *TrafficMgr {
	return &TrafficMgr{}
}

func (self *TrafficMgr) Start() {
	go func() {
		self.mu.Lock()
		self.ticker = time.NewTicker(5 * time.Second)
		self.mu.Unlock()

		// Run MakeTrafficData synchronously in the ticker loop.
		// If it takes longer than the tick interval, the next tick is simply delayed.
		// This prevents goroutine accumulation when nftables GetStats is slow.
		for range self.ticker.C {
			self.MakeTrafficData()
		}
	}()
}

func (self *TrafficMgr) Listen() <-chan sdkapi.TrafficData {
	retCh := make(chan chan sdkapi.TrafficData)
	go func() {
		self.mu.Lock()
		defer self.mu.Unlock()
		ch := make(chan sdkapi.TrafficData)
		self.listners = append(self.listners, ch)
		retCh <- ch
	}()

	return <-retCh
}

func (self *TrafficMgr) MakeTrafficData() {
	// Phase 1: check for listeners and snapshot prevStats under a short read lock.
	// Holding mu for the full function would block Listen() during slow external
	// calls (nftables shell commands, GetMacByIp nftMu acquisition).
	self.mu.RLock()
	hasListeners := len(self.listners) > 0
	prevStats := self.prevStats
	self.mu.RUnlock()

	if !hasListeners {
		return
	}

	// Phase 2: fetch stats and compute traffic deltas entirely outside the lock.
	// GetStats() runs shell commands; GetMacByIp() acquires nftMu.
	// Neither requires TrafficMgr.mu — all inputs come from the snapshot above.
	stats, err := nftables.GetStats()
	if err != nil {
		log.Println(err)
		return
	}

	prev := prevStats
	if prev == nil {
		prev = &nftables.StatResult{
			MacStats: make(map[string]nftables.StatData),
			IpStats:  make(map[string]nftables.StatData),
		}
	}

	trfc := sdkapi.TrafficData{
		Download: make(map[string]sdkapi.ClientStat),
		Upload:   make(map[string]sdkapi.ClientStat),
	}

	for mac, stat := range stats.MacStats {
		prevStat, ok := prev.MacStats[mac]
		macUpper := strings.ToUpper(mac)
		if ok {
			// If current stat is less than prev, user may have been reconnected.
			// In this case we discard previous stats.
			if stat.Packets < prevStat.Packets || stat.Bytes < prevStat.Bytes {
				trfc.Upload[macUpper] = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
			} else {
				pkts := stat.Packets - prevStat.Packets
				byts := stat.Bytes - prevStat.Bytes
				trfc.Upload[macUpper] = sdkapi.ClientStat{Packets: pkts, Bytes: byts}
			}
		} else {
			trfc.Upload[macUpper] = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
		}
	}

	// Aggregate download stats by MAC address.
	// A single session may have traffic on multiple IPs (IPv4 + several IPv6
	// addresses); summing them by MAC gives accurate per-device consumption.
	// IPs not in the ipToMac cache (not currently connected) are ignored.
	for ip, stat := range stats.IpStats {
		mac := nftables.GetMacByIp(ip)
		if mac == "" {
			continue // IP not associated with a connected session — skip
		}
		mac = strings.ToUpper(mac)

		prevStat, ok := prev.IpStats[ip]
		var delta sdkapi.ClientStat
		if ok {
			if stat.Packets < prevStat.Packets || stat.Bytes < prevStat.Bytes {
				// Counter reset (reconnect) — use absolute value for this tick.
				delta = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
			} else {
				delta = sdkapi.ClientStat{Packets: stat.Packets - prevStat.Packets, Bytes: stat.Bytes - prevStat.Bytes}
			}
		} else {
			delta = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
		}

		// Accumulate into the MAC bucket (multiple IPs may map to same MAC).
		existing := trfc.Download[mac]
		trfc.Download[mac] = sdkapi.ClientStat{
			Packets: existing.Packets + delta.Packets,
			Bytes:   existing.Bytes + delta.Bytes,
		}
	}

	// Phase 3: re-acquire write lock only for state mutation and broadcast.
	// This is the minimal critical section — no external calls inside.
	self.mu.Lock()
	defer self.mu.Unlock()

	// Use non-blocking send to prevent deadlock if a listener is not consuming.
	// Listeners that fail to receive are closed and removed.
	activeListeners := []chan sdkapi.TrafficData{}
	for _, ch := range self.listners {
		select {
		case ch <- trfc:
			activeListeners = append(activeListeners, ch)
		default:
			// Listener not consuming, close and remove
			close(ch)
		}
	}
	self.listners = activeListeners

	self.prevStats = &stats
}

// func (self *DataConnMgr) nftStatToMap
