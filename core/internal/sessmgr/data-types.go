package sessmgr

import (
	"context"
	"time"

	"core/internal/network"
	sdkapi "sdk/api"
)

// =============================================================================
// ENUM TYPES
// =============================================================================

// StopReason indicates why a session was stopped.
type StopReason int

const (
	// StopReasonManual indicates the session was stopped by user/admin action (e.g., Disconnect).
	StopReasonManual StopReason = iota

	// StopReasonConsumed indicates the session's time or data allowance was exhausted.
	StopReasonConsumed
)

// String returns a human-readable representation of the stop reason.
func (r StopReason) String() string {
	switch r {
	case StopReasonManual:
		return "manual"
	case StopReasonConsumed:
		return "consumed"
	default:
		return "unknown"
	}
}

// =============================================================================
// FUNCTION TYPES
// =============================================================================

// IsConnectedFunc is a callback to check if a device is connected.
type IsConnectedFunc func(deviceID int64) bool

// =============================================================================
// INTERFACES
// =============================================================================

// SessionEventEmitter is the narrow interface that RunningSession uses to emit session events.
// It is satisfied by *events.EventsManager, keeping RunningSession decoupled from the full manager.
type SessionEventEmitter interface {
	EmitSessionEvent(ctx context.Context, event sdkapi.SessionEvent, data sdkapi.SessionEventData) error
}

// =============================================================================
// PARAMETER STRUCTS
// =============================================================================

// ClientRegisterParams contains parameters for device registration.
type ClientRegisterParams struct {
	CookieDeviceID    *int64
	CookieCookieToken string // cookie_token from the JWT cookie, used for additional validation
	MacAddr           string
	Ipv4Addr          string
	Ipv6Addr          string
	Hostname          string
	// Fingerprint data
	UserAgent string
	ScreenRes string
	Language  string
	Timezone  string
}

// ApplyTimeUpdateParams contains parameters for applying a time update.
type ApplyTimeUpdateParams struct {
	RemainingSecs int
}

// ApplyBandwidthUpdateParams contains parameters for applying a bandwidth update.
type ApplyBandwidthUpdateParams struct {
	DownMbits int
	UpMbits   int
	UseGlobal bool
}

// =============================================================================
// INTERNAL DATA STRUCTS
// =============================================================================

// sessionData holds all session fields as an immutable snapshot.
// This enables lock-free reads via atomic.Pointer.
type sessionData struct {
	id          int64
	uuid        string
	providerPkg string
	devId       int64
	sessionType string
	timeSecs    int
	dataMb      float64
	timeCons    int
	dataCons    float64
	startedAt   *time.Time
	resumedAt   *time.Time
	expDays     *int
	downMbits   int
	upMbits     int
	useGlobal   bool
	createdAt   time.Time
	updatedAt   time.Time

	// Dirty tracking - which fields changed since last save/load
	dirtyTimeSecs       bool // time_secs changed
	dirtyDataMb         bool // data_mb changed
	dirtyTimeCons       bool // time_secs_consumed changed
	dirtyDataCons       bool // data_mb_consumed changed
	dirtyDownMbits      bool // down_speed_mbits changed
	dirtyUpMbits        bool // up_speed_mbits changed
	dirtyUseGlobalSpeed bool // use_global_speed changed
	dirtyExpDays        bool // exp_days changed
	dirtyStartedAt      bool // started_at changed
	dirtyResumedAt      bool // resumed_at changed
}

// networkState holds network-related fields that rarely change.
// Uses atomic pointer for lock-free reads.
type networkState struct {
	ipv4 string
	ipv6 string
	mac  string
	lan  *network.NetworkLan
}
