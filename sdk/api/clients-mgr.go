package sdkapi

import "errors"

// ErrClientAlreadyRegistered is returned by RegisterClient when a device
// with the given MAC address is already registered. Check for this
// specifically with errors.Is before treating a RegisterClient error as an
// expected, ignorable duplicate — any other error is a real failure
// (blocked by an EventClientBeforeCreate subscriber, a DB error, etc.) and
// should not be silently swallowed the same way.
var ErrClientAlreadyRegistered = errors.New("client already registered")

// IClientsMgrApi lets a plugin register a client device it already knows
// the exact MAC/IP/hostname for (e.g. importing history from an external
// source), skipping the cookie/fingerprint/ARP-NDP disambiguation the live
// captive-portal registration flow performs.
type IClientsMgrApi interface {
	// RegisterClient persists dev (built via SessionsMgr().NewClientDevice)
	// as a real device record, emitting the same EventClientBeforeCreate,
	// EventClientCreated, and EventClientRegistered events the live
	// captive-portal registration flow emits. Returns
	// ErrClientAlreadyRegistered if a device with dev's MAC address is
	// already registered.
	RegisterClient(IClientDevice) error
}
