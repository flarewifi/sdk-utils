package ubus

import (
	"strings"

	"github.com/goccy/go-json"

	"core/internal/utils/cmd"
	jobque "core/internal/utils/job-que"
)

var jobQue *jobque.JobQue = jobque.NewJobQue()
var interfaceListeners map[string][]chan InterfaceEvent

func init() {
	interfaceListeners = map[string][]chan InterfaceEvent{}
}

type ifEvent map[string]struct {
	Action    string `json:"action"`
	Interface string `json:"interface"`
}

func parseEvent(b []byte) {
	jobQue.Exec(func() (interface{}, error) {
		eventStr := string(b)
		if strings.HasPrefix(eventStr, `{ "network.interface":`) {
			var evt ifEvent
			err := json.Unmarshal(b, &evt)
			if err != nil {
				return nil, err
			}

			ifEvent := evt["network.interface"]
			listeners, ok := interfaceListeners[ifEvent.Interface]
			if ok {
				ifEvt := InterfaceEvent{
					Ifname: ifEvent.Interface,
					Event:  ifAction(ifEvent.Action),
				}
				for _, ch := range listeners {
					ch <- ifEvt
				}
			}
		}
		return nil, nil
	})
}

func Listen() {
	listner := NewUbusListener()
	go cmd.ExecOutput("ubus listen", listner)
	go func() {
		// then continue listening for more events
		for evt := range listner.OutCh() {
			parseEvent(evt)
		}
	}()
}

func ListenInterface(name string) <-chan InterfaceEvent {
	ch := make(chan InterfaceEvent)
	jobQue.Exec(func() (interface{}, error) {
		_, ok := interfaceListeners[name]
		if !ok {
			interfaceListeners[name] = []chan InterfaceEvent{}
		}
		interfaceListeners[name] = append(interfaceListeners[name], ch)
		return ch, nil
	})
	return ch
}
