//go:build dev

package machineuid

// GetMachineUID returns a unique identifier for the dev environment.
// In dev mode, machine ID never changes, so always returns ("", machineID).
// To simulate a different machine, change the constant below and restart the container.
func GetMachineUID() (string, string) {
	return "", "machine_003"
}
