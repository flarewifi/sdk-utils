package ubus

type NetworkDevice struct {
	Name          string
	Type          string   `json:"devtype"`
	Up            bool     `json:"up"`
	Carrier       bool     `json:"carrier"`
	Speed         string   `json:"speed"`
	Duplex        string   `json:"duplex"`
	MacAddr       string   `json:"macaddr"`
	BridgeMembers []string `json:"bridge-members"`
	Stats         struct {
		RxBytes uint `json:"rx_bytes"`
		TxBytes uint `json:"tx_bytes"`
	} `json:"statistics"`
}
