package events

import (
	"sync"
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

func Emit(event string, data []byte) error {
	v, ok := subscribers.Load(event)
	if ok {
		channels := v.([]chan []byte)
		for _, ch := range channels {
			// Non-blocking send: if the subscriber is not reading (e.g., the HTTP
			// connection was dropped without calling Unsubscribe), skip it rather
			// than blocking the caller indefinitely. A blocked caller can prevent
			// Connect/Disconnect from ever completing.
			select {
			case ch <- data:
			default:
			}
		}
	}

	return nil
}
