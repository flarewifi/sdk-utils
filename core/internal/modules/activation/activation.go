package activation

import (
	machineuid "core/internal/modules/machine-uid"
	rpc_flarewifi_v1 "core/internal/rpc"
	"core/utils/config"
	"core/utils/crypt"
	"core/utils/tags"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/golang-jwt/jwt/v5"
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

// buildMachineInfo builds a MachineInfo struct from system information
func buildMachineInfo(machineID string) (*rpc_flarewifi_v1.MachineInfo, error) {
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

	return &rpc_flarewifi_v1.MachineInfo{
		DeviceModel:    release.DeviceModel,
		DeviceConfig:   release.DeviceConfig,
		MachineId:      machineID,
		CurrentVersion: info.Version,
		BrandId:        release.BrandId,
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

// OnMachineIDChanged is called when the machine ID changes
// It updates the server with the new machine ID
func OnMachineIDChanged(oldID, newID string) {
	log.Printf("Machine ID changed: %s -> %s. Updating server...", oldID, newID)

	srv, ctx := rpc_flarewifi_v1.GetTwirpServiceAndCtx()

	machineInfo, err := buildMachineInfo(newID)
	if err != nil {
		log.Printf("Failed to build machine info: %v", err)
		return
	}

	req := &rpc_flarewifi_v1.UpdateMachineInfoRequest{
		MachineId:   oldID,
		MachineInfo: machineInfo,
	}

	_, err = srv.UpdateMachineInfo(ctx, req)
	if err != nil {
		log.Printf("Failed to update machine info on server: %v", err)
		return
	}

	log.Printf("Successfully updated machine info on server")
}

// CheckActivationFileExists performs an optimistic check for activation file presence
// If the file exists, it assumes the machine is activated without validating the token
// This provides immediate activation state on boot, while Validate() runs in background
func CheckActivationFileExists() bool {
	if _, err := os.Stat(activationFile); err == nil {
		// File exists, optimistically assume activated
		IsActivated.Store(true)
		log.Printf("Activation file exists, optimistically setting activated=true")
		return true
	}
	return false
}

func Validate() {
	IsValidating.Store(true)
	defer IsValidating.Store(false)

	// Check if machine ID changed and update server if needed
	oldID, newID := machineuid.GetMachineUID()
	if oldID != "" && newID != "" && oldID != newID {
		OnMachineIDChanged(oldID, newID)
		// Machine ID changed - token is no longer valid
		// Remove activation file and update state
		if _, err := os.Stat(activationFile); err == nil {
			_ = os.Remove(activationFile)
			log.Printf("Machine ID changed, removing activation file")
		}
		IsActivated.Store(false)
		ActivationError.Store(fmt.Errorf("machine ID changed from %s to %s", oldID, newID))
		return
	}

	// 1. Try online activation first
	okOnline, errOnline := checkActivationOnline()

	// 2. If server is reachable (no network error)
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
			log.Printf("Server validation failed, removing activation file and correcting optimistic assumption")
		}
		ActivationError.Store(ErrNotActivated)
		IsActivated.Store(false)
		return
	}

	// 3. Server unreachable - fall back to offline validation
	ok, errOffline := offlineActivation()
	if ok {
		IsActivated.Store(true)
		log.Printf("Offline activation validation succeeded")
		return
	}

	// Offline validation failed - token is invalid or expired
	// Do NOT remove the activation file when server is unreachable
	// Only remove when server explicitly says "not activated"
	log.Printf("Offline activation validation failed (server unreachable): %v", errOffline)
	ActivationError.Store(errOffline)
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
	srv, ctx := rpc_flarewifi_v1.GetTwirpServiceAndCtx()

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
	params := rpc_flarewifi_v1.MachineActivationRequest{
		MachineInfo: &rpc_flarewifi_v1.MachineInfo{
			DeviceModel:    release.DeviceModel,
			DeviceConfig:   release.DeviceConfig,
			MachineId:      machineID,
			CurrentVersion: info.Version,
			BrandId:        release.BrandId,
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

	var act *rpc_flarewifi_v1.MachineActivationResponse
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

		logMsg := fmt.Sprintf("Activation attempt %d failed: %v", attempt+1, err)
		log.Println(logMsg)
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
