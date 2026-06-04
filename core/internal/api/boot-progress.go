package api

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	sse "core/utils/sse"
)

type BootProgData struct {
	Logs []string `json:"logs"`
	Done bool     `json:"done"`
}

type BootProgress struct {
	mu      sync.RWMutex
	logs    []string
	done    atomic.Bool
	sockets []*sse.SseSocket
	DONE_C  chan error // should only be used once
}

func NewBootProgress() *BootProgress {
	return &BootProgress{
		sockets: []*sse.SseSocket{},
		DONE_C:  make(chan error),
	}
}

func (bp *BootProgress) AppendLog(s string) {
	go func() {
		bp.mu.Lock()
		defer bp.mu.Unlock()
		s = fmt.Sprintf("%s %s", time.Now().Format("01-02-2006 15:04:05"), s)
		bp.logs = append(bp.logs, s)
		bp.emit()
	}()
}

func (bp *BootProgress) Logs() []string {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.logs
}

func (bp *BootProgress) IsDone() bool {
	return bp.done.Load()
}

func (bp *BootProgress) Done(err error) {
	bp.done.Store(true)

	bp.mu.Lock()
	if err != nil {
		bp.logs = append(bp.logs, err.Error())
	}
	bp.emit()
	bp.mu.Unlock()

	bp.DONE_C <- err
}

func (bp *BootProgress) AddSocket(s *sse.SseSocket) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.sockets = append(bp.sockets, s)

	go func() {
		<-s.Done()
		bp.mu.Lock()
		defer bp.mu.Unlock()

		sockets := []*sse.SseSocket{}
		for _, ss := range sockets {
			if s.ID() != ss.ID() {
				sockets = append(sockets, s)
			}
		}

		bp.sockets = sockets
	}()
}

func (bp *BootProgress) emit() {
	bootProgData := BootProgData{bp.logs, bp.done.Load()}

	data, err := json.Marshal(bootProgData)
	if err != nil {
		return
	}

	for _, s := range bp.sockets {
		s.Emit("boot:progress", data)
	}
}
