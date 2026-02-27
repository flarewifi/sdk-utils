package sysinfo

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
)

// netRateState tracks previous network readings for rate calculation
var netRateState struct {
	sync.Mutex
	timestamp   time.Time
	bytesSent   uint64
	bytesRecv   uint64
	initialized bool
}

// SystemInfo holds basic system metrics.
type SystemInfo struct {
	Arch            string    `json:"arch"`
	NumCPU          int       `json:"num_cpu"`
	CPUPercent      []float64 `json:"cpu_percent"`
	CPUTemperature  []float64 `json:"cpu_temperature,omitempty"`
	MemTotal        uint64    `json:"mem_total"`
	MemUsed         uint64    `json:"mem_used"`
	MemUsedPercent  float64   `json:"mem_used_percent"`
	DiskTotal       uint64    `json:"disk_total"`
	DiskUsed        uint64    `json:"disk_used"`
	DiskUsedPercent float64   `json:"disk_used_percent"`
	NetInterface    string    `json:"net_interface"`
	NetDownloadRate uint64    `json:"net_download_rate"` // bytes per second (RX)
	NetUploadRate   uint64    `json:"net_upload_rate"`   // bytes per second (TX)
}

// GetSystemInfo retrieves basic system information: CPU, memory, disk, and temperature.
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		Arch:   runtime.GOARCH,
		NumCPU: runtime.NumCPU(),
	}

	// CPU usage per core (measure over 100ms for accuracy)
	cpuPercents, _ := cpu.Percent(100*time.Millisecond, true)
	info.CPUPercent = cpuPercents

	// CPU temperature (optional)
	temps, _ := sensors.SensorsTemperatures()
	if len(temps) > 0 {
		tempVals := make([]float64, 0, len(temps))
		for _, t := range temps {
			tempVals = append(tempVals, t.Temperature)
		}
		info.CPUTemperature = tempVals
	}

	// Memory stats
	vmem, _ := mem.VirtualMemory()
	if vmem != nil {
		info.MemTotal = vmem.Total
		info.MemUsed = vmem.Used
		info.MemUsedPercent = vmem.UsedPercent
	}

	// Disk stats (root)
	diskUsage, _ := disk.Usage("/")
	if diskUsage != nil {
		info.DiskTotal = diskUsage.Total
		info.DiskUsed = diskUsage.Used
		info.DiskUsedPercent = diskUsage.UsedPercent
	}

	// Network stats - find WAN interface and calculate rates
	netStats, _ := net.IOCounters(true) // per interface
	wanInterface := findWANInterface(netStats)
	if wanInterface != nil {
		info.NetInterface = wanInterface.Name

		// Calculate rates based on previous reading
		netRateState.Lock()
		now := time.Now()
		if netRateState.initialized {
			elapsed := now.Sub(netRateState.timestamp).Seconds()
			if elapsed > 0 {
				info.NetDownloadRate = uint64(float64(wanInterface.BytesRecv-netRateState.bytesRecv) / elapsed)
				info.NetUploadRate = uint64(float64(wanInterface.BytesSent-netRateState.bytesSent) / elapsed)
			}
		}
		netRateState.timestamp = now
		netRateState.bytesSent = wanInterface.BytesSent
		netRateState.bytesRecv = wanInterface.BytesRecv
		netRateState.initialized = true
		netRateState.Unlock()
	}

	return info, nil
}

// findWANInterface attempts to identify the WAN interface from network statistics.
// It looks for interfaces named: wan, wan0, eth0, eth1, en0, en1, etc.
// Returns the first matching interface or nil if none found.
func findWANInterface(stats []net.IOCountersStat) *net.IOCountersStat {
	// Priority order: wan, wan0, eth1, eth0, en0, en1
	wanNames := []string{"wan", "wan0", "eth1", "eth0", "en0", "en1"}

	for _, wanName := range wanNames {
		for _, stat := range stats {
			if stat.Name == wanName {
				return &stat
			}
		}
	}

	// Fallback: look for any interface starting with "wan", "eth", or "en"
	for _, stat := range stats {
		name := strings.ToLower(stat.Name)
		if strings.HasPrefix(name, "wan") ||
			strings.HasPrefix(name, "eth") ||
			strings.HasPrefix(name, "en") {
			// Skip loopback and local interfaces
			if !strings.Contains(name, "lo") &&
				!strings.Contains(name, "docker") &&
				!strings.Contains(name, "veth") {
				return &stat
			}
		}
	}

	return nil
}
