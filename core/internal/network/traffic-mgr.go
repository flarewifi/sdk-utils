package network

import (
	"log"
	"sync"
	"time"

	"core/internal/utils/nftables"
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

		for range self.ticker.C {
			go self.MakeTrafficData()
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
	self.mu.Lock()
	defer self.mu.Unlock()

	if len(self.listners) == 0 {
		return
	}

	stats, err := nftables.GetStats()
	if err != nil {
		log.Println(err)
		return
	}

	prevStats := &nftables.StatResult{
		MacStats: make(map[string]nftables.StatData),
		IpStats:  make(map[string]nftables.StatData),
	}

	if self.prevStats != nil {
		prevStats = self.prevStats
	}

	trfc := sdkapi.TrafficData{
		Download: make(map[string]sdkapi.ClientStat),
		Upload:   make(map[string]sdkapi.ClientStat),
	}

	for mac, stat := range stats.MacStats {
		prev, ok := prevStats.MacStats[mac]
		if ok {
			// If current stat is less than prev, user may have been reconnected.
			// In this case we discard previous stats.
			if stat.Packets < prev.Packets || stat.Bytes < prev.Bytes {
				trfc.Upload[mac] = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
				continue
			}

			pkts := stat.Packets - prev.Packets
			byts := stat.Bytes - prev.Bytes
			trfc.Upload[mac] = sdkapi.ClientStat{Packets: pkts, Bytes: byts}
		} else {
			trfc.Upload[mac] = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
		}
	}

	for ip, stat := range stats.IpStats {
		prev, ok := prevStats.IpStats[ip]
		if ok {
			// If current stat is less than prev, user may have been reconnected.
			// In this case we discard previous stats.
			if stat.Packets < prev.Packets || stat.Bytes < prev.Bytes {
				trfc.Download[ip] = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
				continue
			}

			pkts := stat.Packets - prev.Packets
			byts := stat.Bytes - prev.Bytes
			trfc.Download[ip] = sdkapi.ClientStat{Packets: pkts, Bytes: byts}
		} else {
			trfc.Download[ip] = sdkapi.ClientStat{Packets: stat.Packets, Bytes: stat.Bytes}
		}
	}

	for _, ch := range self.listners {
		ch <- trfc
	}

	self.prevStats = &stats
}

// func (self *DataConnMgr) nftStatToMap
