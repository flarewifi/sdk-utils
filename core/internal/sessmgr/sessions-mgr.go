package sessmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/events"
	"core/internal/modules/nftables"
	"core/internal/network"
	jobque "core/utils/job-que"
	sdkapi "sdk/api"
)

// =============================================================================
// TYPES
// =============================================================================

type SessionsMgr struct {
	coreAPI    sdkapi.IPluginApi
	pluginsMgr sdkapi.IPluginsMgrApi
	db         *db.Database
	mdl        *models.Models
	sessions   sync.Map
	eventsMgr  *events.EventsManager

	// tcQueue serializes all TC/NFT commands globally.
	// Uses JobQueue for automatic panic recovery, queue-wait logging, and context support.
	tcQueue *jobque.JobQueue[struct{}]
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

func NewSessionsMgr(dtb *db.Database, mdl *models.Models, eventsMgr *events.EventsManager) *SessionsMgr {
	sessionMgr := &SessionsMgr{
		db:        dtb,
		mdl:       mdl,
		sessions:  sync.Map{},
		eventsMgr: eventsMgr,
		tcQueue:   jobque.NewJobQueue[struct{}](),
	}
	return sessionMgr
}

// =============================================================================
// PUBLIC METHODS - Initialization
// =============================================================================

func (self *SessionsMgr) SetCoreAPI(api sdkapi.IPluginApi) {
	self.coreAPI = api
	if api != nil {
		self.pluginsMgr = api.PluginsMgr()
	}
}

func (self *SessionsMgr) Init(ctx context.Context) error {
	// First, update consumption for all running sessions
	err := self.db.Queries.BulkUpdateTimeConsumption(ctx)
	if err != nil {
		return fmt.Errorf("failed to update consumption before reset: %w", err)
	}

	// Then reset all resumed_at fields to NULL
	err = self.db.Queries.ResetAllResumedAt(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset resumed_at fields: %w", err)
	}

	// Reset all device connection statuses to disconnected
	err = self.db.Queries.ResetAllDeviceStatuses(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset device statuses: %w", err)
	}

	return nil
}

// =============================================================================
// PUBLIC METHODS - Event Handling
// =============================================================================

// OnSessionEvent registers a callback for a session event (delegates to EventsManager).
func (self *SessionsMgr) OnSessionEvent(event sdkapi.SessionEvent, callback func(data sdkapi.SessionEventData) error) {
	self.eventsMgr.OnSessionEvent(event, callback)
}

// OnClientEvent registers a callback for a client event (delegates to EventsManager).
func (self *SessionsMgr) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice) error) {
	self.eventsMgr.OnClientEvent(event, callback)
}

// EmitSessionEvent dispatches a session event asynchronously via EventsManager.
func (self *SessionsMgr) EmitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession) {
	self.eventsMgr.EmitSessionEvent(event, sdkapi.SessionEventData{Session: session})
}

// EmitClientEvent dispatches a client event asynchronously via EventsManager.
func (self *SessionsMgr) EmitClientEvent(event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) {
	self.eventsMgr.EmitClientEvent(event, clnt)
}

// EmitClientMerge dispatches a client-merge event asynchronously via EventsManager.
// Internal use only — called by ClientRegister for MAC-collision merges that must
// defer the target reconnect to the surrounding UpdateDevice flow.
func (self *SessionsMgr) EmitClientMerge(data sdkapi.EventClientMergeData) {
	self.eventsMgr.EmitClientMerge(data)
}

// MergeClientDevices merges sourceID into targetID: transfers all data, deletes source,
// disconnects/reconnects active sessions around the merge, and emits OnClientMerge.
func (self *SessionsMgr) MergeClientDevices(ctx context.Context, targetID, sourceID int64) error {
	if targetID == sourceID {
		return fmt.Errorf("cannot merge device into itself (id=%d)", targetID)
	}

	targetClnt, err := self.FindDeviceByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("target device %d not found: %w", targetID, err)
	}

	sourceClnt, err := self.FindDeviceByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("source device %d not found: %w", sourceID, err)
	}

	// Disconnect source if it has an active session.
	if _, hasSession := self.GetRunningSession(sourceClnt); hasSession {
		log.Printf("[SessionsMgr.MergeClientDevices] Disconnecting active session on source device %d before merge", sourceID)
		if err := self.Disconnect(ctx, sourceClnt, ""); err != nil {
			log.Printf("[SessionsMgr.MergeClientDevices] WARN: Failed to disconnect source device %d: %v", sourceID, err)
			// Continue with merge anyway.
		}
	}

	// Disconnect target if it has an active session; remember so we can reconnect.
	var targetHadSession bool
	if _, hasSession := self.GetRunningSession(targetClnt); hasSession {
		targetHadSession = true
		disconnectMsg := self.coreAPI.Translate("info", "Device merge in progress, reconnecting")
		if err := self.Disconnect(ctx, targetClnt, disconnectMsg); err != nil {
			log.Printf("[SessionsMgr.MergeClientDevices] WARN: Failed to disconnect target device %d: %v", targetID, err)
			// Continue with merge anyway.
		}
	}

	// Capture source UUID before deletion.
	sourceDeviceUUID := sourceClnt.UUID()

	// Perform the DB merge (transfers sessions, purchases, fingerprints, wallet; deletes source).
	if err := self.mdl.Device().MergeDevices(ctx, targetID, sourceID); err != nil {
		return fmt.Errorf("failed to merge device %d into %d: %w", sourceID, targetID, err)
	}

	log.Printf("[SessionsMgr.MergeClientDevices] Successfully merged device %d into %d", sourceID, targetID)

	// Reload target once to get updated state after merge.
	// Reused for both the event emit and the optional reconnect below.
	reloadedTarget, reloadErr := self.FindDeviceByID(ctx, targetID)
	if reloadErr != nil {
		log.Printf("[SessionsMgr.MergeClientDevices] WARN: Failed to reload target device %d after merge: %v", targetID, reloadErr)
		// Fall back to original reference so the event is still emitted.
		reloadedTarget = targetClnt
	}

	self.eventsMgr.EmitClientMerge(sdkapi.EventClientMergeData{
		Target:           reloadedTarget,
		SourceDeviceID:   sourceID,
		SourceDeviceUUID: sourceDeviceUUID,
	})

	// Reconnect target if it was active before the merge.
	if targetHadSession {
		reconnectMsg := self.coreAPI.Translate("success", "Device merge completed, reconnected successfully")
		if err := self.Connect(ctx, reloadedTarget, reconnectMsg); err != nil {
			log.Printf("[SessionsMgr.MergeClientDevices] ERROR: Failed to reconnect target device %d after merge: %v", targetID, err)
		} else {
			log.Printf("[SessionsMgr.MergeClientDevices] Target device %d reconnected successfully after merge", targetID)
		}
	}

	return nil
}

// =============================================================================
// PUBLIC METHODS - Traffic and Session Management
// =============================================================================

func (self *SessionsMgr) ListenTraffic(trfk *network.TrafficMgr) {
	// Use a single goroutine to process traffic updates sequentially.
	// This prevents unbounded goroutine spawning under high traffic.
	// Each update iterates all running sessions, which is fast (in-memory).
	go func() {
		for data := range trfk.Listen() {
			dataCopy := data // Copy loop variable for safety
			self.sessions.Range(func(key, value any) bool {
				rs := value.(*RunningSession)
				rs.UpdateDataConsumption(&dataCopy)
				return true
			})
		}
	}()
}

func (self *SessionsMgr) ReloadSessions(ctx context.Context, iface string) error {
	errCh := make(chan error, 1) // Buffered to prevent goroutine leak

	go func() {
		var rangeErr error
		self.sessions.Range(func(key, value any) bool {
			rs := value.(*RunningSession)
			lan := rs.Lan()

			if lan.Name() == iface {
				// Skip sessions that are stopped or in the process of stopping.
				// ReloadSessions is called when network interfaces change, but a session
				// may be concurrently stopping (e.g., timer expired). Calling Start() on
				// a stopping session would clear the stopped flag and create inconsistent state.
				if rs.IsStopped() {
					log.Printf("[SessionsMgr] ReloadSessions: skipping stopped session for device %d", rs.ClientId())
					return true
				}

				cs := rs.GetSession()
				err := cs.Reload(ctx)
				if err != nil {
					rangeErr = err
					return false
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					rangeErr = err
					return false
				}
			}

			return true
		})

		errCh <- rangeErr // Always sends exactly once (nil or error)
	}()

	return <-errCh
}

func (self *SessionsMgr) StopSessions(ctx context.Context, iface string, reason string) {
	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)

		// Read network state atomically from the RunningSession.
		// This is the authoritative state (updated inside execTc+mu by UpdateNetworkDetails).
		net := rs.network.Load()
		if net == nil || net.lan == nil {
			return true
		}

		if net.lan.Name() == iface {
			mac := net.mac

			// Disconnect both IPv4 and IPv6 from their respective verdict maps.
			if net.ipv4 != "" {
				if err := nftables.Disconnect(net.ipv4, mac); err != nil {
					log.Printf("[SessionsMgr] StopSessions: failed to disconnect IPv4=%s MAC=%s: %v", net.ipv4, mac, err)
				}
			}
			if net.ipv6 != "" {
				if err := nftables.Disconnect(net.ipv6, mac); err != nil {
					log.Printf("[SessionsMgr] StopSessions: failed to disconnect IPv6=%s MAC=%s: %v", net.ipv6, mac, err)
				}
			}

			if err := rs.Stop(); err != nil {
				log.Printf("[SessionsMgr] StopSessions: failed to stop session for device MAC=%s: %v", mac, err)
			}

			// Clean up TC classes/filters and restore class ID to pool
			if err := rs.CleanupTc(); err != nil {
				log.Printf("[SessionsMgr] StopSessions: failed to cleanup TC for device MAC=%s: %v", mac, err)
			}

			// Remove from sessions map
			self.sessions.Delete(key)
		}

		return true
	})
}

// =============================================================================
// PUBLIC METHODS - Connection Management
// =============================================================================

// Connect connects a client device to the internet.
// Note: ctx is accepted for API compatibility but ignored internally to avoid
// complexity from context cancellation mid-operation.
func (self *SessionsMgr) Connect(_ context.Context, clnt sdkapi.IClientDevice, notify string) error {
	if clnt.Status() == sdkapi.DeviceStatusBlocked {
		return errors.New(self.coreAPI.Translate("error", "Device is blocked"))
	}

	// Launch session loop - handles nftables, session start, events, and chaining.
	startCh := make(chan error, 1)
	go self.loopSessions(startCh, clnt, notify)

	// Wait for result from loopSessions
	err := <-startCh

	if err == nil {
		err = clnt.Update(context.Background(), sdkapi.UpdateDeviceParams{
			UUID:     clnt.UUID(),
			Mac:      clnt.MacAddr(),
			Ipv4:     clnt.Ipv4Addr(),
			Ipv6:     clnt.Ipv6Addr(),
			Hostname: clnt.Hostname(),
			Status:   sdkapi.DeviceStatusConnected,
		})
		if err != nil {
			err = fmt.Errorf("unable to update device status to connected: %w", err)
		}
	}
	return err
}

// Disconnect disconnects a client device from the internet.
// Note: ctx is accepted for API compatibility but ignored internally to avoid
// complexity from context cancellation mid-operation.
func (self *SessionsMgr) Disconnect(_ context.Context, clnt sdkapi.IClientDevice, notify string) error {
	// Session events (EventSessionDisconnected) are emitted by StopWithReason() inside endSession().
	err := self.endSession(clnt)
	if err != nil {
		return err
	}

	// Emit disconnect events asynchronously so cloud sync RPC calls don't block
	// the HTTP handler. nftables/TC cleanup already happened in endSession() above.
	go func() {
		clnt.Emit(string(sdkapi.EventSessionDisconnected), []byte(notify))
		self.EmitClientEvent(sdkapi.EventClientDisconnected, clnt)
	}()

	return clnt.Update(context.Background(), sdkapi.UpdateDeviceParams{
		UUID:     clnt.UUID(),
		Mac:      clnt.MacAddr(),
		Ipv4:     clnt.Ipv4Addr(),
		Ipv6:     clnt.Ipv6Addr(),
		Hostname: clnt.Hostname(),
		Status:   sdkapi.DeviceStatusDisconnected,
	})
}

// disconnectInternal is called from loopSessions when the session already ended via
// StopWithReason (which already emitted EventSessionDisconnected to plugin callbacks).
// It performs the same cleanup as Disconnect but:
// - Still sends SSE notification to browser (clnt.Emit) so the UI updates
// - Emits EventClientDisconnected for client-level tracking
// - Does NOT re-emit EventSessionDisconnected to plugin callbacks (already done by StopWithReason)
func (self *SessionsMgr) disconnectInternal(clnt sdkapi.IClientDevice, notify string) error {
	err := self.endSession(clnt)
	if err != nil {
		return err
	}

	// Send SSE notification to browser so the UI updates.
	// Note: This only fires SSE + events.Emit, NOT plugin callbacks (emitSessionEvent).
	// StopWithReason already fired plugin callbacks, so we don't duplicate those.
	clnt.Emit(string(sdkapi.EventSessionDisconnected), []byte(notify))
	self.EmitClientEvent(sdkapi.EventClientDisconnected, clnt)

	return clnt.Update(context.Background(), sdkapi.UpdateDeviceParams{
		UUID:     clnt.UUID(),
		Mac:      clnt.MacAddr(),
		Ipv4:     clnt.Ipv4Addr(),
		Ipv6:     clnt.Ipv6Addr(),
		Hostname: clnt.Hostname(),
		Status:   sdkapi.DeviceStatusDisconnected,
	})
}

func (self *SessionsMgr) IsConnected(clnt sdkapi.IClientDevice) (connected bool) {
	return nftables.IsConnected(clnt.MacAddr())
}

// =============================================================================
// PUBLIC METHODS - Session Queries
// =============================================================================

func (self *SessionsMgr) CurrSession(clnt sdkapi.IClientDevice) (cs sdkapi.IClientSession, ok bool) {
	v, ok := self.sessions.Load(clnt.ID())
	if !ok {
		return nil, false
	}

	rs, ok := v.(*RunningSession)
	if !ok {
		return nil, false
	}

	return rs.GetSession(), true
}

// GetRunningSession returns the running session for a client device (public wrapper)
func (self *SessionsMgr) GetRunningSession(clnt sdkapi.IClientDevice) (rs *RunningSession, ok bool) {
	return self.getRunningSession(clnt)
}

func (self *SessionsMgr) GetSession(ctx context.Context, clnt sdkapi.IClientDevice) (sdkapi.IClientSession, error) {
	localClient := clnt.(*ClientDevice)
	s, err := self.mdl.Session().AvailableForDevice(ctx, localClient.id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(self.coreAPI.Translate("error", "No more available sessions"))
		}
		return nil, err
	}

	return self.wrapModelSession(s), nil
}

// SessionSummary returns the total remaining time/data from ALL sessions for a client device.
// The database queries return the total based on saved consumption values.
// We need to subtract both elapsed time and unsaved data consumption for running sessions.
func (self *SessionsMgr) SessionSummary(ctx context.Context, clnt sdkapi.IClientDevice) (*sdkapi.ClientSessionSummary, error) {
	summary, err := self.mdl.Session().Summary(ctx, clnt.ID())
	if err != nil {
		return nil, err
	}

	// Check if there's a running session
	rs, ok := self.getRunningSession(clnt)
	if !ok {
		// No running session, return database totals as-is
		return summary, nil
	}

	// Calculate elapsed time for the running session since resumed_at.
	// Use a single GetSession() call and a single ResumedAt() snapshot to avoid
	// a race where SnapshotTimeCons sets resumedAt to nil between the nil check
	// and the dereference, which would cause a nil pointer panic.
	var elapsedSecs int = 0
	session := rs.GetSession()
	resumedAt := session.ResumedAt()
	if resumedAt != nil {
		elapsedSecs = int(time.Since(*resumedAt).Seconds())
	}

	// Get unsaved data consumption diff (data consumed but not yet written to DB)
	mbDiff := rs.DiffMb()

	// Subtract both elapsed time and unsaved data consumption
	remainingTime := summary.RemainingTimeSecs - elapsedSecs
	remainingData := summary.RemainingDataMb - mbDiff

	// Ensure we don't go below zero
	remainingTime = max(remainingTime, 0)
	remainingData = max(remainingData, 0)

	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs: remainingTime,
		RemainingDataMb:   remainingData,
	}, nil
}

// ListRunningSessions returns all currently active (running) sessions.
// These are sessions that are actively connected and consuming time/data.
func (self *SessionsMgr) ListRunningSessions() ([]sdkapi.IClientSession, error) {
	var sessions []sdkapi.IClientSession

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		// Skip sessions that are stopped or in the process of stopping
		if rs.IsStopped() {
			return true // Continue iteration
		}
		session := rs.GetSession()
		if session != nil {
			sessions = append(sessions, session)
		}
		return true // Continue iteration
	})

	return sessions, nil
}

// FindRunningSessionByUUID finds a currently running session by its UUID.
// Returns the session and true if found, or nil and false if no running session
// exists with the given UUID.
func (self *SessionsMgr) FindRunningSessionByUUID(uuid string) (sdkapi.IClientSession, bool) {
	var foundSession sdkapi.IClientSession

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		// Skip sessions that are stopped or in the process of stopping
		if rs.IsStopped() {
			return true // Continue iteration
		}
		session := rs.GetSession()
		if session != nil && session.UUID() == uuid {
			foundSession = session
			return false // Stop iteration
		}
		return true // Continue iteration
	})

	if foundSession != nil {
		return foundSession, true
	}
	return nil, false
}

// =============================================================================
// PUBLIC METHODS - Device/Session Finders
// =============================================================================

// FindDeviceByID finds a device by its database ID and wraps it into an IClientDevice object.
func (self *SessionsMgr) FindDeviceByID(ctx context.Context, deviceID int64) (sdkapi.IClientDevice, error) {
	d, err := self.mdl.Device().Find(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	return self.wrapModelDevice(d), nil
}

// FindDeviceByUUID finds a device by its UUID and wraps it into an IClientDevice object.
func (self *SessionsMgr) FindDeviceByUUID(ctx context.Context, uuid string) (sdkapi.IClientDevice, error) {
	d, err := self.mdl.Device().FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return self.wrapModelDevice(d), nil
}

// FindSessionByID finds a session by its database ID and wraps it into an IClientSession object.
func (self *SessionsMgr) FindSessionByID(ctx context.Context, sessionID int64) (sdkapi.IClientSession, error) {
	s, err := self.mdl.Session().Find(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return self.wrapModelSession(s), nil
}

// FindSessionByUUID finds a session by its UUID and wraps it into an IClientSession object.
func (self *SessionsMgr) FindSessionByUUID(ctx context.Context, uuid string) (sdkapi.IClientSession, error) {
	s, err := self.mdl.Session().FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return self.wrapModelSession(s), nil
}

// FindClientById finds a client device by its database ID.
func (self *SessionsMgr) FindClientById(ctx context.Context, devId int64) (sdkapi.IClientDevice, error) {
	device, err := self.mdl.Device().Find(ctx, devId)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	return NewClientDevice(self.db, self.mdl, self, device), nil
}

// FindClientByMac finds a client device by its MAC address.
func (self *SessionsMgr) FindClientByMac(ctx context.Context, mac string) (sdkapi.IClientDevice, error) {
	device, err := self.mdl.Device().FindByMac(ctx, mac)
	if err != nil {
		return nil, fmt.Errorf("device not found by MAC %s: %w", mac, err)
	}

	return NewClientDevice(self.db, self.mdl, self, device), nil
}

// FindClientByIp finds a client device by its IP address.
func (self *SessionsMgr) FindClientByIp(ctx context.Context, ip string) (sdkapi.IClientDevice, error) {
	device, err := self.mdl.Device().FindByIp(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("device not found by IP %s: %w", ip, err)
	}

	return NewClientDevice(self.db, self.mdl, self, device), nil
}

// =============================================================================
// PUBLIC METHODS - Factory Methods
// =============================================================================

// NewClientSession wraps session data into an IClientSession object without performing
// additional database queries.
func (self *SessionsMgr) NewClientSession(params sdkapi.NewClientSessionParams) sdkapi.IClientSession {
	// Create a models.Session from the params using BuildSession
	s := models.BuildSession(models.BuildSessionParams{
		DB:             self.db,
		Models:         self.mdl,
		ID:             params.ID,
		UUID:           params.UUID,
		ProviderPkg:    params.ProviderPkg,
		DeviceID:       params.DeviceID,
		Type:           string(params.Type),
		TimeSecs:       params.TimeSecs,
		DataMb:         params.DataMb,
		TimeCons:       params.TimeCons,
		DataCons:       params.DataCons,
		StartedAt:      params.StartedAt,
		ResumedAt:      params.ResumedAt,
		ExpDays:        params.ExpDays,
		DownMbits:      params.DownMbits,
		UpMbits:        params.UpMbits,
		UseGlobalSpeed: params.UseGlobalSpeed,
		CreatedAt:      params.CreatedAt,
		UpdatedAt:      params.UpdatedAt,
	})
	return self.wrapModelSession(s)
}

// NewClientDevice wraps device data into an IClientDevice object without performing
// additional database queries.
func (self *SessionsMgr) NewClientDevice(params sdkapi.NewDeviceParams) sdkapi.IClientDevice {
	// Create a models.Device from the params using BuildDevice
	d := models.BuildDevice(models.BuildDeviceParams{
		DB:          self.db,
		Models:      self.mdl,
		ID:          params.ID,
		UUID:        params.UUID,
		CookieToken: params.CookieToken,
		MacAddr:     params.MacAddress,
		Ipv4Addr:    params.Ipv4Address,
		Ipv6Addr:    params.Ipv6Address,
		Hostname:    params.Hostname,
		Status:      params.Status,
		CreatedAt:   params.CreatedAt,
		UpdatedAt:   params.UpdatedAt,
	})
	return self.wrapModelDevice(d)
}

// =============================================================================
// PUBLIC METHODS - Bandwidth Updates
// =============================================================================

// UpdateInterfaceBandwidth updates the bandwidth settings for all running sessions on the specified interface.
// This is called when bandwidth settings are saved via Config().Bandwidth().Save().
// It iterates all running sessions, and for each session on the specified interface:
// - Updates bandwidth based on UseGlobal setting
// - Saves the session (which triggers ApplyBandwidthUpdate via the save callback)
func (self *SessionsMgr) UpdateInterfaceBandwidth(ctx context.Context, ifname string, cfg sdkapi.IBandwdCfg) {
	log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: updating sessions on interface %s", ifname)

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		lan := rs.Lan()

		if lan == nil || lan.Name() != ifname {
			return true // Continue to next session
		}

		// Skip stopped sessions
		if rs.IsStopped() {
			log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: skipping stopped session for device %d", rs.ClientId())
			return true
		}

		session := rs.GetSession()
		if session == nil {
			return true
		}

		// Determine bandwidth based on UseGlobal setting
		var downMbits, upMbits int
		if cfg.UseGlobal {
			downMbits = cfg.GlobalDownMbits
			upMbits = cfg.GlobalUpMbits
		} else {
			downMbits = cfg.UserDownMbits
			upMbits = cfg.UserUpMbits
		}

		log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: updating session %d - Down=%d, Up=%d, UseGlobal=%v",
			session.ID(), downMbits, upMbits, cfg.UseGlobal)

		// Update session bandwidth settings
		session.SetData(sdkapi.SessionUpdateData{
			DownMbits:      &downMbits,
			UpMbits:        &upMbits,
			UseGlobalSpeed: &cfg.UseGlobal,
		})

		// Save triggers the save callback which calls ApplyBandwidthUpdate
		if err := session.Save(ctx, nil); err != nil {
			log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: failed to save session %d: %v", session.ID(), err)
			// Continue updating other sessions
		}

		return true // Continue to next session
	})

	log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: completed for interface %s", ifname)
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// emitSessionEvent dispatches a session event with optional changed-fields data.
// It is an internal helper for call sites that pass changedFields (e.g. handleSessionSaved).
func (self *SessionsMgr) emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, changedFields ...sdkapi.SessionChangedFields) {
	data := sdkapi.SessionEventData{Session: session}
	if len(changedFields) > 0 {
		data.ChangedFields = changedFields[0]
	}
	self.eventsMgr.EmitSessionEvent(event, data)
}

func (self *SessionsMgr) loopSessions(startedCh chan<- error, clnt sdkapi.IClientDevice, notify string) {
	var startOnce sync.Once
	var started bool // tracks whether first session started successfully
	ctx := context.Background()

	// signalStart sends err to startedCh exactly once (first session result)
	signalStart := func(err error) {
		startOnce.Do(func() {
			startedCh <- err
			started = (err == nil)
		})
	}

	// Loop condition: continue while connected OR until first session starts
	// This allows the first iteration to run before nftables rules are added
	for nftables.IsConnected(clnt.MacAddr()) || !started {
		// Get next available session
		cs, err := self.GetSession(ctx, clnt)
		if err != nil {
			if !started {
				// First attempt failed - user sees error immediately
				signalStart(err)
			} else {
				// Session chaining failed - no more sessions available.
				// StopWithReason already emitted EventSessionDisconnected; use internal helper.
				self.disconnectInternal(clnt, self.coreAPI.Translate("info", "No more sessions available"))
			}
			return
		}

		// Get or create running session
		rs, ok := self.getRunningSession(clnt)
		newlyCreated := !ok
		if !ok {
			rs, err = NewRunningSession(clnt, cs, self.eventsMgr, self.tcQueue)
			if err != nil {
				if !started {
					signalStart(err)
				} else {
					// StopWithReason already emitted EventSessionDisconnected.
					self.disconnectInternal(clnt, self.coreAPI.Translate("error", "Failed to create session"))
				}
				return
			}
			self.sessions.Store(clnt.ID(), rs)
		}

		// Start the session (this also sets up TC classes/filters)
		err = rs.Start(ctx, cs)
		if err != nil {
			// If we just created and stored this RunningSession in this iteration,
			// remove it — Start() failed so it never became active.
			if newlyCreated {
				self.sessions.Delete(clnt.ID())
			}
			if !started {
				// First session start failed - user sees error immediately
				signalStart(err)
			} else {
				// Chained session start failed - disconnect.
				// StopWithReason already emitted EventSessionDisconnected.
				self.disconnectInternal(clnt, self.coreAPI.Translate("error", "Failed to start session"))
			}
			return
		}

		// First successful start - add firewall rules and emit events
		if !started {
			// Add firewall rules for each IP the device has (IPv4 and/or IPv6).
			// nftables.Connect is idempotent for the MAC entry: the second call
			// adds only the missing IP map entry, MAC rules are only added once.
			mac := clnt.MacAddr()
			if ipv4 := clnt.Ipv4Addr(); ipv4 != "" {
				if err := nftables.Connect(ipv4, mac); err != nil {
					rs.Stop()
					rs.CleanupTc()
					self.sessions.Delete(clnt.ID())
					signalStart(err)
					return
				}
			}
			if ipv6 := clnt.Ipv6Addr(); ipv6 != "" {
				if err := nftables.Connect(ipv6, mac); err != nil {
					// Rollback IPv4 if it was already connected
					if ipv4 := clnt.Ipv4Addr(); ipv4 != "" {
						nftables.Disconnect(ipv4, mac)
					}
					rs.Stop()
					rs.CleanupTc()
					self.sessions.Delete(clnt.ID())
					signalStart(err)
					return
				}
			}

			// Signal success to Connect() immediately after nftables rules are in place.
			// Event callbacks (cloud sync RPC calls) run after this in the loopSessions
			// goroutine so they don't block the HTTP handler.
			signalStart(nil)

			// Emit connection events (runs in this goroutine after unblocking the HTTP handler)
			clnt.Emit(string(sdkapi.EventSessionConnected), []byte(notify))
			if session, ok := self.CurrSession(clnt); ok {
				self.emitSessionEvent(sdkapi.EventSessionConnected, session)
			}
			self.EmitClientEvent(sdkapi.EventClientConnected, clnt)
		}

		// Wait for session to end
		err = <-rs.Done()

		// Handle session end
		if err != nil {
			if errors.Is(err, ErrSessionExpired) || errors.Is(err, ErrSessionStopped) {
				// Session expired or was stopped (e.g., via UpdateTime(0)) - prepare for next session
				// TC class/filter are preserved for reuse with the next session
				log.Printf("Session ended for device %s (reason: %v), checking for next available session...", clnt.MacAddr(), err)
				rs.PrepareForChain()

				// Check if this loop has been superseded by a new Connect() call.
				// UpdateDevice calls Disconnect (which deletes from sessions map and stops rs),
				// then Connect (which spawns a NEW loopSessions with a new RunningSession).
				// If the RunningSession in the map is no longer ours, we've been replaced — exit.
				currentRs, stillInMap := self.getRunningSession(clnt)
				if !stillInMap || currentRs != rs {
					log.Printf("[loopSessions] Session loop superseded for device %s (replaced by new Connect), exiting", clnt.MacAddr())
					return
				}

				continue
			}
			// Other error from rs.Done() - StopWithReason already emitted EventSessionDisconnected.
			self.disconnectInternal(clnt, self.coreAPI.Translate("error", "Session ended unexpectedly"))
			return
		}

		// Session ended without error - continue loop to check for next session
	}

	// Loop exited because nftables.IsConnected returned false
	// Device was disconnected externally
}

func (self *SessionsMgr) getRunningSession(clnt sdkapi.IClientDevice) (rs *RunningSession, ok bool) {
	v, ok := self.sessions.Load(clnt.ID())
	if !ok {
		return nil, false
	}

	rs, ok = v.(*RunningSession)
	if !ok {
		return nil, false
	}

	return rs, true
}

func (self *SessionsMgr) endSession(clnt sdkapi.IClientDevice) error {
	rs, ok := self.getRunningSession(clnt)

	// C1: Read IPs/MAC from the RunningSession's authoritative network snapshot,
	// not from clnt which may be a stale wrapper object.
	// Fall back to clnt only when there is no RunningSession (session was never started).
	var ipv4, ipv6, mac string
	if ok {
		net := rs.network.Load()
		ipv4, ipv6, mac = net.ipv4, net.ipv6, net.mac
	} else {
		ipv4, ipv6, mac = clnt.Ipv4Addr(), clnt.Ipv6Addr(), clnt.MacAddr()
	}

	if nftables.IsConnected(mac) {
		// C2: Disconnect both IPs independently. Collect errors but always attempt
		// both so a failure on IPv4 never leaves IPv6 stranded in the verdict map.
		var firstErr error
		if ipv4 != "" {
			if err := nftables.Disconnect(ipv4, mac); err != nil {
				log.Printf("[SessionsMgr] endSession: failed to disconnect IPv4=%s MAC=%s: %v", ipv4, mac, err)
				firstErr = err
			}
		}
		if ipv6 != "" {
			if err := nftables.Disconnect(ipv6, mac); err != nil {
				log.Printf("[SessionsMgr] endSession: failed to disconnect IPv6=%s MAC=%s: %v", ipv6, mac, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		if firstErr != nil {
			return firstErr
		}
	}

	if ok {
		if err := rs.Stop(); err != nil {
			return err
		}

		if err := rs.CleanupTc(); err != nil {
			return err
		}
	}

	self.sessions.Delete(clnt.ID())
	return nil
}

// wrapModelSession wraps a models.Session into an IClientSession.
// This is the internal helper used by all session-wrapping methods.
func (self *SessionsMgr) wrapModelSession(s *models.Session) *ClientSession {
	return NewClientSession(self.db, self.mdl, self.coreAPI.PluginsMgr(), self, s)
}

// wrapModelDevice wraps a models.Device into an IClientDevice.
// This is the internal helper used by all device-wrapping methods.
func (self *SessionsMgr) wrapModelDevice(d *models.Device) *ClientDevice {
	return NewClientDevice(self.db, self.mdl, self, d)
}

// isDeviceConnected checks if a device has a running session (resumed_at IS NOT NULL).
func (self *SessionsMgr) isDeviceConnected(deviceID int64) bool {
	// Check in-memory running sessions first (faster)
	if _, ok := self.sessions.Load(deviceID); ok {
		return true
	}
	return false
}

// getRunningSessionBySessionID finds a running session by its session ID.
// Returns the running session, the client device, and whether it was found.
func (self *SessionsMgr) getRunningSessionBySessionID(sessionID int64) (*RunningSession, sdkapi.IClientDevice, bool) {
	var foundRs *RunningSession
	var foundClnt sdkapi.IClientDevice

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		session := rs.GetSession()
		if session != nil && session.ID() == sessionID {
			foundRs = rs
			foundClnt = rs.clnt
			return false // Stop iteration
		}
		return true // Continue iteration
	})

	if foundRs != nil {
		return foundRs, foundClnt, true
	}
	return nil, nil, false
}

// handleSessionSaved applies side effects after a session is saved.
// For running sessions: resets timer (if time changed), updates TC rules (if bandwidth changed).
// For all sessions: emits EventSessionChanged (unless opts.IgnoreCallbacks is true).
func (self *SessionsMgr) handleSessionSaved(ctx context.Context, session sdkapi.IClientSession, changed sdkapi.SessionChangedFields, opts *sdkapi.SessionSaveOpts) error {
	// Get all session data in a single atomic snapshot to avoid multiple getter calls
	sessionData := session.Data()

	// Check if this is a running session
	rs, _, isRunning := self.getRunningSessionBySessionID(sessionData.ID)

	if isRunning {
		// Apply side effects to running session
		// Time changed: timeSecs or timeCons
		if changed.TimeSecs || changed.TimeCons {
			if err := rs.ApplyTimeUpdate(ApplyTimeUpdateParams{
				RemainingSecs: sessionData.RemainingTime,
			}); err != nil {
				return err
			}
		}
		// Data changed: dataMb or dataCons
		if changed.DataMb || changed.DataCons {
			// Check if session is now consumed after data update
			if err := rs.ApplyDataUpdate(); err != nil {
				return err
			}
		}
		// Bandwidth changed: downMbits, upMbits, or useGlobalSpeed
		if changed.DownMbits || changed.UpMbits || changed.UseGlobalSpeed {
			if err := rs.ApplyBandwidthUpdate(ApplyBandwidthUpdateParams{
				DownMbits: sessionData.DownMbits,
				UpMbits:   sessionData.UpMbits,
				UseGlobal: sessionData.UseGlobalSpeed,
			}); err != nil {
				return err
			}
		}
	}

	// Emit event (unless IgnoreCallbacks is set)
	if opts == nil || !opts.IgnoreCallbacks {
		self.emitSessionEvent(sdkapi.EventSessionChanged, session, changed)
	}

	return nil
}
