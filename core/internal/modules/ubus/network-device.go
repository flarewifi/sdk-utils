package ubus

type NetworkDevice struct {
	Name          string
	Type          string   `json:"devtype"`
	Up            bool     `json:"up"`
	Speed         string   `json:"speed"`
	MacAddr       string   `json:"macaddr"`
	BridgeMembers []string `json:"bridge-members"`
	Stats         struct {
		RxBytes uint `json:"rx_bytes"`
		TxBytes uint `json:"tx_bytes"`
	} `json:"statistics"`
}
