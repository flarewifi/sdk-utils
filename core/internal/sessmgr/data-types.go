package sessmgr

import (
	"context"
	"time"

	"core/internal/network"
	sdkapi "sdk/api"
)

// =============================================================================
// FUNCTION TYPES
// =============================================================================

// IsConnectedFunc is a callback to check if a device is connected.
type IsConnectedFunc func(deviceID int64) bool

// =============================================================================
// INTERFACES
// =============================================================================

// SessionEventEmitter interface for emitting session events
type SessionEventEmitter interface {
	emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, changedFields ...sdkapi.SessionChangedFields) error
}

// =============================================================================
// PARAMETER STRUCTS
// =============================================================================

// ClientRegisterParams contains parameters for device registration.
type ClientRegisterParams struct {
	CookieDeviceID *int64
	MacAddr        string
	IpAddr         string
	Hostname       string
	// Fingerprint data
	UserAgent string
	ScreenRes string
	Language  string
	Timezone  string
}

// ApplyTimeUpdateParams contains parameters for applying a time update.
type ApplyTimeUpdateParams struct {
	Ctx           context.Context
	RemainingSecs int
}

// ApplyBandwidthUpdateParams contains parameters for applying a bandwidth update.
type ApplyBandwidthUpdateParams struct {
	Ctx       context.Context
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
	ip  string
	mac string
	lan *network.NetworkLan
}
