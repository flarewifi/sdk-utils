//go:build dev

package machineuid

// GetMachineUIDWithChange returns (cachedID, calculatedID).
// In dev mode, machine ID never changes, so always returns ("", machineID).
func GetMachineUIDWithChange() (string, string) {
	return "", "machine_003"
}

// GetMachineUID returns a unique identifier for the dev environment.
// In dev mode, machine ID never changes, so always returns ("", machineID).
// To simulate a different machine, change the constant below and restart the container.
func GetMachineUID() (string, string) {
	return "", "machine_003"
}

// WriteCachedMachineID is a no-op in dev mode since machine ID is hardcoded
func WriteCachedMachineID(uid string) {
	// No-op in dev mode
}
