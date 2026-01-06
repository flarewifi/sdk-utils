package ubus

type ifAction string

const (
	IfEventDown ifAction = "ifdown"
	IfEventUp   ifAction = "ifup"
)

type InterfaceEvent struct {
	Ifname string
	Event  ifAction
}
