/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sse

import (
	"sync"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	socketStore = sync.Map{}
)

func AddSocket(key string, socket *SseSocket) {
	var sockets []*SseSocket
	v, ok := socketStore.Load(key)
	if ok {
		sockets = v.([]*SseSocket)
	}

	sockets = append(sockets, socket)
	socketStore.Store(key, sockets)

	go func() {
		<-socket.Done()
		RemoveSocket(key, socket)
	}()
}

func RemoveSocket(key string, socket *SseSocket) {
	v, ok := socketStore.Load(key)
	if ok {
		sockets := v.([]*SseSocket)
		sockets = sdkutils.SliceFilter(sockets, func(item *SseSocket) bool {
			return item.ID() != socket.ID()
		})

		if len(sockets) == 0 {
			socketStore.Delete(key)
		} else {
			socketStore.Store(key, sockets)
		}
	}
}

func Emit(key string, event string, data []byte) {
	v, ok := socketStore.Load(key)
	if ok {
		sockets := v.([]*SseSocket)
		for _, s := range sockets {
			s.Emit(event, data)
		}
	}
}

func Broadcast(event string, data []byte) {
	socketStore.Range(func(key, value interface{}) bool {
		sockets := value.([]*SseSocket)
		for _, s := range sockets {
			s.Emit(event, data)
		}
		return true
	})
}
