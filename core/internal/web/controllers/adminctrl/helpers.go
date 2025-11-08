//go:build !mono

package adminctrl

import (
	"sync"
)

type InstallStatus string

const (
	PendingStatus    InstallStatus = "pending"
	InProgressStatus InstallStatus = "in-progress"
	SuccessStatus    InstallStatus = "success"
	FailedStatus     InstallStatus = "failed"
)

type PluginProgress struct {
	Name      string        `json:"name"`
	Status    InstallStatus `json:"status"`
	Progress  int           `json:"progress"`
	Message   string        `json:"message"`
	IsRunning bool          `json:"is_running"`
}

type StatusManager struct {
	mu     sync.RWMutex
	status map[string]*PluginProgress
}

var manager = &StatusManager{
	status: make(map[string]*PluginProgress),
}

// SaveInitialState initializes plugin status
func SaveInitialState(name string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.status[name] = &PluginProgress{
		Name:      name,
		Status:    PendingStatus,
		Progress:  25,
		IsRunning: true,
	}
}

// UpdateStatus updates plugin installation state
func UpdateStatus(name string, status InstallStatus, msg string, progress int) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if s, ok := manager.status[name]; ok {
		s.Status = status
		s.Progress = progress
		s.Message = msg

		if status == FailedStatus || status == SuccessStatus {
			s.IsRunning = false
		}
	}
}

// GetStatus returns current plugin status
func GetStatus(name string) *PluginProgress {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if s, ok := manager.status[name]; ok {
		cp := *s
		return &cp
	}
	return nil
}
