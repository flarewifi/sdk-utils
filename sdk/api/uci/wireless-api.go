/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkuci

var (
	// source: https://openwrt.org/docs/guide-user/network/wifi/basic
	WifiEncryptions = []string{
		"none", "sae", "sae-mixed", "psk2+tkip+ccmp", "psk2+tkip+aes", "psk2+tkip", "psk2+ccmp", "psk2+aes", "psk2",
		"psk+tkip+ccmp", "psk+tkip+aes", "psk+tkip", "psk+ccmp", "psk+aes", "psk", "psk-mixed+tkip+ccmp", "psk-mixed+tkip+aes",
		"psk-mixed+tkip", "psk-mixed+ccmp", "psk-mixed+aes", "psk-mixed", "wep", "wep+open", "wep+shared", "wpa3", "wpa3-mixed",
		"wpa2+tkip+ccmp", "wpa2+tkip+aes", "wpa2+ccmp", "wpa2+aes", "wpa2", "wpa2+tkip", "wpa+tkip+ccmp", "wpa+tkip+aes",
		"wpa+ccmp", "wpa+aes", "wpa+tkip", "wpa", "wpa-mixed+tkip+ccmp", "wpa-mixed+tkip+aes", "wpa-mixed+tkip", "wpa-mixed+ccmp",
		"wpa-mixed+aes", "wpa-mixed", "owe",
	}
	WifiBands = []string{"2g", "5g", "6g"}
)

type WifiDev struct {
	Section  string
	Type     string
	Path     string
	Channel  uint
	Band     string
	Htmode   string
	Disabled bool
}

type WifiIface struct {
	Section    string
	Device     string
	Network    string
	Mode       string
	Ssid       string
	Encryption string
	Key        string
}

type IWirelessApi interface {
	GetDeviceSecs() (sections []string)
	GetIfaceSecs() (sections []string)
	GetDevice(section string) (dev *WifiDev, err error)
	GetIface(section string) (iface *WifiIface, err error)
	GetDevices() (devs []*WifiDev)
	GetIfaces() (ifaces []*WifiIface)
	SetDevice(section string, cfg *WifiDev) error
	SetIface(section string, cfg *WifiIface) error
}
