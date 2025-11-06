package sysinfo

import (
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"runtime"
)

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
}

// GetSystemInfo retrieves basic system information: CPU, memory, disk, and temperature.
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		Arch:   runtime.GOARCH,
		NumCPU: runtime.NumCPU(),
	}

	// CPU usage per core
	cpuPercents, err := cpu.Percent(0, true)
	if err == nil {
		info.CPUPercent = cpuPercents
	}

	// CPU temperature (optional)
	temps, err := host.SensorsTemperatures()
	if err == nil && len(temps) > 0 {
		tempVals := make([]float64, 0, len(temps))
		for _, t := range temps {
			tempVals = append(tempVals, t.Temperature)
		}
		info.CPUTemperature = tempVals
	}

	// Memory stats
	vmem, err := mem.VirtualMemory()
	if err == nil {
		info.MemTotal = vmem.Total
		info.MemUsed = vmem.Used
		info.MemUsedPercent = vmem.UsedPercent
	}

	// Disk stats (root)
	diskUsage, err := disk.Usage("/")
	if err == nil {
		info.DiskTotal = diskUsage.Total
		info.DiskUsed = diskUsage.Used
		info.DiskUsedPercent = diskUsage.UsedPercent
	}

	return info, nil
}
