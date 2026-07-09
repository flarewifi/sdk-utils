package sdkapi

import (
	"context"
	"errors"
)

// ErrClientAlreadyRegistered is returned by RegisterClient when a device
// with the given MAC address is already registered. Check for this
// specifically with errors.Is before treating a RegisterClient error as an
// expected, ignorable duplicate — any other error is a real failure
// (blocked by an EventClientBeforeCreate subscriber, a DB error, etc.) and
// should not be silently swallowed the same way.
var ErrClientAlreadyRegistered = errors.New("client already registered")

// IClientsMgrApi manages client devices: looking them up, wrapping preview
// objects, and registering a client device a plugin already knows the exact
// MAC/IP/hostname for (e.g. importing history from an external source),
// skipping the cookie/fingerprint/ARP-NDP disambiguation the live
// captive-portal registration flow performs.
type IClientsMgrApi interface {
	// FindClientById finds a client device by its ID.
	FindClientById(ctx context.Context, devId int64) (IClientDevice, error)

	// FindClientByMac finds a client device by its MAC address.
	FindClientByMac(ctx context.Context, mac string) (IClientDevice, error)

	// FindClientByIp finds a client device by its IP address.
	// This is useful for scenarios where you have an IP address (e.g., from an HTTP request)
	// and need to find the associated device.
	FindClientByIp(ctx context.Context, ip string) (IClientDevice, error)

	// FindClientByUUID finds a client device by its globally unique identifier.
	// This is useful for referencing devices by their UUID rather than local database ID.
	FindClientByUUID(ctx context.Context, uuid string) (IClientDevice, error)

	// NewClientDevice wraps device data into an IClientDevice object without performing
	// additional database queries. This is useful when you already have device data from queries
	// and want to use SDK methods like Update(), Emit(), and Subscribe(). The params parameter
	// contains all device fields from the database row. Also use this to build an in-memory
	// preview (e.g. ID left at 0) to pass to RegisterClient.
	NewClientDevice(params NewDeviceParams) IClientDevice

	// RegisterClient persists dev (built via NewClientDevice) as a real
	// device record, emitting the same EventClientBeforeCreate,
	// EventClientCreated, and EventClientRegistered events the live
	// captive-portal registration flow emits. Returns
	// ErrClientAlreadyRegistered if a device with dev's MAC address is
	// already registered.
	RegisterClient(IClientDevice) error

	// MergeClientDevices merges the source device into the target device.
	// All sessions, purchases, and fingerprints are transferred from
	// source to target. The source device is deleted after the merge.
	//
	// Active sessions on either device are disconnected before the merge. If the
	// target device had an active session it is reconnected afterward.
	//
	// The OnClientMerge event is emitted after a successful merge so all registered
	// callbacks (e.g. cloud sync) are notified.
	//
	// Returns an error if the DB merge fails. Session disconnect/reconnect failures
	// are logged but do not abort the merge.
	MergeClientDevices(ctx context.Context, targetID, sourceID int64) error
}
