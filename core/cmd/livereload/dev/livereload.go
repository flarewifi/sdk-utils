//go:build dev

package dev

import (
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

type LiveReloader struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func NewLiveReloader() *LiveReloader {
	return &LiveReloader{
		clients: make(map[*websocket.Conn]bool),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins in dev
	},
}

func (lr *LiveReloader) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	lr.mu.Lock()
	lr.clients[conn] = true
	lr.mu.Unlock()

	log.Println("🔌 Browser connected for live reload")
	go func() {
		time.Sleep(500 * time.Millisecond) // slight delay to ensure connection is ready
		lr.BroadcastReload()
	}()

	// Keep connection alive until closed
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	lr.mu.Lock()
	delete(lr.clients, conn)
	lr.mu.Unlock()
	conn.Close()
	log.Println("❌ Browser disconnected")
}

func (lr *LiveReloader) BroadcastReload() {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	for conn := range lr.clients {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("reload")); err != nil {
			log.Println("failed to send reload:", err)
			conn.Close()
			delete(lr.clients, conn)
		}
	}
}

func (lr *LiveReloader) WatchPaths(paths []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("fsnotify error:", err)
	}
	defer watcher.Close()

	for _, path := range paths {
		if err := watcher.Add(path); err != nil {
			log.Println("watch add error:", err)
		} else {
			log.Println("📂 Watching:", path)
		}
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				f := filepath.Base(event.Name)
				if f == filepath.Base(sdkutils.PathServerUp) {
					log.Println("🔄 Change detected:", event.Name)
					lr.BroadcastReload()
				}
			}
		case err := <-watcher.Errors:
			log.Println("watch error:", err)
		}
	}
}
