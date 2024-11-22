package events

import (
	"sync"

	"github.com/goccy/go-json"
)

var (
	subscribers sync.Map
)

func Subscribe(event string) <-chan []byte {
	ch := make(chan []byte)
	v, ok := subscribers.Load(event)
	if !ok {
		channels := []chan []byte{ch}
		subscribers.Store(event, channels)
	} else {
		channels := v.([]chan []byte)
		channels = append(channels, ch)
		subscribers.Store(event, channels)
	}

	return ch
}

func Unsubscribe(event string, ch <-chan []byte) {
	v, ok := subscribers.Load(event)
	if ok {
		channels := v.([]chan []byte)
		for i, c := range channels {
			if c == ch {
				channels = append(channels[:i], channels[i+1:]...)
				break
			}
		}
	}
}

func Emit(event string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	v, ok := subscribers.Load(event)
	if ok {
		channels := v.([]chan []byte)
		for _, ch := range channels {
			ch <- bytes
		}
	}

	return nil
}
