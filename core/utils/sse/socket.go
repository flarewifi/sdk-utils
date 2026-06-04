/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sse

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewSocket(w http.ResponseWriter, r *http.Request) (s *SseSocket, err error) {
	f, ok := w.(http.Flusher)
	if !ok {
		err = errors.New("streaming not supported")
		return nil, err
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	f.Flush()

	id := sdkutils.RandomStr(8)

	return &SseSocket{
		id:      id,
		res:     w,
		req:     r,
		msgCh:   make(chan SseData),
		flusher: f,
	}, nil
}

type SseSocket struct {
	id      string
	res     http.ResponseWriter
	req     *http.Request
	flusher http.Flusher
	msgId   int32
	msgCh   chan SseData
}

type SseData struct {
	MsgType string
	Data    []byte
}

func (s *SseSocket) ID() string {
	return s.id
}

func (s *SseSocket) Emit(typ string, data []byte) (err error) {
	s.msgCh <- SseData{typ, data}
	return nil
}

func (s *SseSocket) Done() <-chan struct{} {
	return s.req.Context().Done()
}

func (s *SseSocket) Flush() {
	s.flusher.Flush()
}

func (s *SseSocket) Listen() {

	// Prevents the connection from being closed by the browser
	go s.pingLoop()

	for {
		select {
		case d := <-s.msgCh:
			data := string(d.Data)
			lines := strings.Split(data, "\n")
			fmt.Fprintf(s.res, "id: %d\nevent: %s\n", s.msgId, d.MsgType)
			for _, line := range lines {
				fmt.Fprintf(s.res, "data: %s\n", line)
			}
			fmt.Fprint(s.res, "\n")
			s.Flush()
			s.msgId += 1
		case <-s.Done():
			return
		}
	}
}

func (s *SseSocket) pingLoop() {
	for {
		select {
		case <-time.After(5 * time.Second):
			s.Emit("ping", []byte(""))
		case <-s.Done():
			return
		}

	}
}
