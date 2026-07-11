package activation

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/config"
	"core/utils/crypt"
	"core/utils/product"
	"core/utils/tags"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	sdkutils "github.com/flarewifi/sdk-utils"
	"github.com/golang-jwt/jwt/v5"
)

const (
	pendingMachineUpdateFile = "/etc/.pmu"
	maxUpdateRetries         = 5
	retryBaseDelay           = 5 * time.Second
)

var (
	ErrNetworkIssue = errors.New("Activation error: network issue")
	ErrProcessFail  = errors.New("Activation error: process failure")
	ErrNotActivated = errors.New("Activation error: not activated")

	osReleaseFile  = "/etc/os_release.json"
	activationFile = "/etc/.tkn"
	randSeed       = "9562641867"

	IsValidating    atomic.Bool
	IsActivated     atomic.Bool
	ActivationError atomic.Value
)

// pendingMachineUpdate represents a machine ID update that needs to be synced to the cloud
type pendingMachineUpdate struct {
	OldID string `json:"old_id"`
	NewID string `json:"new_id"`
}

// savePendingMachineUpdate saves the machine ID update for retry on next boot
func savePendingMachineUpdate(oldID, newID string) {
	data, _ := json.Marshal(pendingMachineUpdate{OldID: oldID, NewID: newID})
	_ = os.WriteFile(pendingMachineUpdateFile, data, 0644)
}

// loadPendingMachineUpdate loads any pending machine ID update from previous boot
func loadPendingMachineUpdate() *pendingMachineUpdate {
	data, err := os.ReadFile(pendingMachineUpdateFile)
	if err != nil {
		return nil
	}
	var p pendingMachineUpdate
	if json.Unmarshal(data, &p) != nil {
		return nil
	}
	return &p
}

// clearPendingMachineUpdate removes the pending update file after successful sync
func clearPendingMachineUpdate() {
	_ = os.Remove(pendingMachineUpdateFile)
}

// buildMachineInfo builds a MachineInfo struct from system information
func buildMachineInfo(machineID string) (*rpc_flarewifi_v3.MachineInfo, error) {
	release, err := sdkutils.ReadOsRelease(osReleaseFile)
	if err != nil {
		return nil, err
	}

	info, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return nil, err
	}

	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return nil, err
	}

	return &rpc_flarewifi_v3.MachineInfo{
		// DeviceModel is read from the frozen os_release.json (stable for the
		// device's physical lifetime); DeviceConfig/BrandId are sourced from the
		// restamped, encrypted core/product.json (via the product package) — see
		// that package's doc comment.
		DeviceModel:  release.DeviceModel,
		DeviceConfig: product.DeviceConfig(),
		MachineId:    machineID,
		// CoreVersion is the ABI identity (core/plugin.json); ProductVersion is the
		// per-partner update lineage (core/product.json, falling back to the core
		// version on unstamped builds). The cloud registers CoreVersion as the
		// machine's version and decides update-eligibility from ProductVersion.
		CoreVersion:    info.Version,
		ProductVersion: product.Version(),
		BrandId:        product.BrandId(),
		Os:             strings.ToLower(release.Os),
		OsVersion:      release.OsVersion,
		OsTarget:       release.OsTarget,
		OsArch:         release.OsArch,
		OsProfile:      release.OsProfile,
		GoVersion:      sdkutils.GO_VERSION,
		GoArch:         sdkutils.GOARCH,
		Monolythic:     tags.HasGoTag("mono"),
		Channel:        strings.ToLower(cfg.Channel),
	}, nil
}

// updateMachineIDOnCloud updates the machine ID on the cloud server
// Returns true if successful, false otherwise
func updateMachineIDOnCloud(oldID, newID string) bool {
	srv, ctx := rpc.GetTwirpServiceAndCtx()

	machineInfo, err := buildMachineInfo(newID)
	if err != nil {
		return false
	}

	req := &rpc_flarewifi_v3.UpdateMachineInfoRequest{
		MachineId:   oldID,
		MachineInfo: machineInfo,
	}

	// Retry with exponential backoff
	for attempt := 1; attempt <= maxUpdateRetries; attempt++ {
		_, err = srv.UpdateMachineInfo(ctx, req)
		if err == nil {
			return true
		}

		if attempt < maxUpdateRetries {
			delay := retryBaseDelay * time.Duration(attempt)
			time.Sleep(delay)
		}
	}

	return false
}

// processMachineIDChange handles machine ID changes before activation
// Returns true if machine ID is ready for use (either unchanged, updated, or using cached)
func processMachineIDChange() {
	// First check for pending update from previous boot
	if pending := loadPendingMachineUpdate(); pending != nil {
		if updateMachineIDOnCloud(pending.OldID, pending.NewID) {
			// Success - update local cache and clear pending
			machineuid.WriteCachedMachineID(pending.NewID)
			clearPendingMachineUpdate()
			// Remove activation token since machine ID changed
			_ = os.Remove(activationFile)
		}
		return
	}

	// Check for new machine ID change
	oldID, newID := machineuid.GetMachineUIDWithChange()

	// No cached ID (new machine) or no change
	if oldID == "" {
		return
	}

	// Machine ID has changed - update cloud first
	if updateMachineIDOnCloud(oldID, newID) {
		// Success - update local cache
		machineuid.WriteCachedMachineID(newID)
		// Remove activation token since machine ID changed
		_ = os.Remove(activationFile)
	} else {
		// Failed - save for retry on next boot
		savePendingMachineUpdate(oldID, newID)
	}
}

// CheckActivationFileExists performs an optimistic check for activation file presence
// If the file exists, it assumes the machine is activated without validating the token
// This provides immediate activation state on boot, while Validate() runs in background
func CheckActivationFileExists() bool {
	if _, err := os.Stat(activationFile); err == nil {
		// File exists, optimistically assume activated
		IsActivated.Store(true)
		return true
	}
	return false
}

func Validate() {
	// Devkit builds never contact the cloud: treat the machine as activated and
	// skip all online/offline token validation. This also protects the manual
	// re-check endpoint from flipping IsActivated to false via a failed dial.
	if devkitBypass() {
		IsActivated.Store(true)
		return
	}

	IsValidating.Store(true)
	defer IsValidating.Store(false)

	// 1. Process machine ID changes FIRST (before activation)
	processMachineIDChange()

	// 2. Try online activation
	okOnline, errOnline := checkActivationOnline()

	// 3. If server is reachable (no network error)
	if errOnline == nil || errors.Is(errOnline, ErrNotActivated) {
		if okOnline {
			// Server says activated
			IsActivated.Store(true)
			return
		}

		// Server says NOT activated
		// If previously activated (token file exists), remove it
		if _, err := os.Stat(activationFile); err == nil {
			_ = os.Remove(activationFile)
		}
		ActivationError.Store(ErrNotActivated)
		IsActivated.Store(false)
		return
	}

	// 4. Server unreachable - fall back to offline validation
	ok, _ := offlineActivation()
	if ok {
		IsActivated.Store(true)
		return
	}

	// Offline validation failed - token is invalid or expired
	// Do NOT remove the activation file when server is unreachable
	// Only remove when server explicitly says "not activated"
	ActivationError.Store(ErrNetworkIssue)
	IsActivated.Store(false)
}

func offlineActivation() (ok bool, err error) {
	encToken, err := sdkutils.FsReadFile(activationFile)
	if err != nil {
		return false, err
	}

	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return false, err
	}
	secret := cfg.Secret + randSeed

	decryptedToken, err := crypt.DecryptToken(encToken, secret)
	if err != nil {
		return false, err
	}

	if decryptedToken == "" {
		return false, err
	}

	_, machineID := machineuid.GetMachineUID()

	token, err := jwt.Parse(decryptedToken, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrProcessFail
		}
		return []byte(machineID), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return false, err
	}
	if !token.Valid {
		return false, errors.New("Activation error: invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, errors.New("Activation error: failed claim")
	}

	claimID, ok := claims["machine_id"]
	if !ok {
		return false, fmt.Errorf("Activation error: failed claim")
	}

	if claimID != machineID {
		return false, fmt.Errorf("Activation error: machine ID mismatch")
	}

	return true, nil
}

func checkActivationOnline() (ok bool, err error) {
	srv, ctx := rpc.GetTwirpServiceAndCtx()

	release, err := sdkutils.ReadOsRelease(osReleaseFile)
	if err != nil {
		return false, ErrProcessFail
	}

	info, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return false, ErrProcessFail
	}

	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return false, ErrProcessFail
	}

	_, machineID := machineuid.GetMachineUID()
	params := rpc_flarewifi_v3.MachineActivationRequest{
		MachineInfo: &rpc_flarewifi_v3.MachineInfo{
			DeviceModel:    release.DeviceModel,
			DeviceConfig:   product.DeviceConfig(),
			MachineId:      machineID,
			CoreVersion:    info.Version,
			ProductVersion: product.Version(),
			BrandId:        product.BrandId(),
			Os:             strings.ToLower(release.Os),
			OsVersion:      release.OsVersion,
			OsTarget:       release.OsTarget,
			OsArch:         release.OsArch,
			OsProfile:      release.OsProfile,
			GoVersion:      sdkutils.GO_VERSION,
			GoArch:         sdkutils.GOARCH,
			Monolythic:     tags.HasGoTag("mono"),
			Channel:        strings.ToLower(cfg.Channel),
		},
	}

	var act *rpc_flarewifi_v3.MachineActivationResponse
	maxAttempts := 3
	retryDelays := []time.Duration{0, 5 * time.Second, 10 * time.Second}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelays[attempt])
		}

		act, err = srv.MachineActivation(ctx, &params)
		if err == nil {
			break
		}

		if attempt == maxAttempts-1 {
			return false, ErrNetworkIssue
		}
	}

	if err != nil {
		return false, ErrNetworkIssue
	}

	if act.Activated {
		token := act.Token
		secret := cfg.Secret + randSeed
		encrypted, err := crypt.EncryptToken(token, secret)
		if err != nil {
			return false, ErrProcessFail
		}
		if err := sdkutils.FsWriteFile(activationFile, []byte(encrypted)); err != nil {
			return false, ErrProcessFail
		}
		return true, nil
	}

	return false, ErrNotActivated
}
