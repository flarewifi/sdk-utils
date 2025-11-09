package activation

import (
	rpc_flarewifi_v1 "core/internal/rpc"
	"core/internal/utils/crypt"
	machineuid "core/internal/utils/machine-uid"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"tools/config"
	"tools/tags"

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

func Validate() {
	IsValidating.Store(true)
	defer IsValidating.Store(false)

	ok, _ := offlineActivation()
	if ok {
		IsActivated.Store(true)
		return
	}

	okOnline, errOnline := checkActivationOnline()
	if !okOnline || errOnline != nil {
		ActivationError.Store(errOnline)
		IsActivated.Store(false)
		return
	}

	IsActivated.Store(true)
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

	machineID := machineuid.GetMachineUID()

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

	machineID := machineuid.GetMachineUID()
	params := rpc_flarewifi_v1.MachineActivationRequest{
		MachineId:      machineID,
		CurrentVersion: info.Version,
		BrandId:        release.BrandId,
		Os:             strings.ToLower(release.Os),
		OsVersion:      release.OsVersion,
		OsTarget:       release.OsTarget,
		OsArch:         release.OsArch,
		OsProfile:      release.OsProfile,
		OsConfig:       release.OsConfig,
		GoVersion:      sdkutils.GO_VERSION,
		GoArch:         sdkutils.GOARCH,
		IsMono:         tags.HasGoTag("mono"),
		Channel:        strings.ToLower(cfg.Channel),
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
