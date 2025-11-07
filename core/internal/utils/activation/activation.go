package activation

import (
	rpc_flarewifi_v1 "core/internal/rpc"
	"core/internal/utils/crypt"
	machineuid "core/internal/utils/machine-uid"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"tools/config"
	"tools/tags"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrNetworkIssue = errors.New("Activation error: network issue")
	ErrNotActivated = errors.New("Activation error: not activated")

	osReleaseFile  = "/etc/os_release.json"
	activationFile = "/etc/.tkn"

	IsValidating    atomic.Bool
	IsActivated     atomic.Bool
	ActivationError atomic.Value
)

func ValidateActivation() {
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
	ActivationError.Store(nil)
}

func offlineActivation() (ok bool, err error) {
	// Read the encrypted token from the activation file
	encToken, err := sdkutils.FsReadFile(activationFile)
	if err != nil {

		return false, fmt.Errorf("Activation error: failed to read activation file: %w", err)
	}

	// Read the application config to get the secret
	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return false, fmt.Errorf("Activation error: failed to read config: %w", err)
	}
	secret := cfg.Secret

	// Decrypt the token
	decryptedToken, err := crypt.DecryptToken(encToken, secret)
	if err != nil {

		return false, fmt.Errorf("Activation error: failed to decrypt token: %w", err)
	}

	if decryptedToken == "" {
		return false, ErrNotActivated
	}

	// The token is a jwt token, we can do basic validation
	// This is how it was created in the backend using github.com/golang-jwt/jwt/v5:
	// claims := jwt.MapClaims{
	// 	"machine_id": machine.MachineID,
	// 	"exp":        time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days expiration
	// 	"iat":        time.Now().Unix(),
	// }

	machineID := machineuid.GetMachineUID()

	// Parse and validate the token
	token, err := jwt.Parse(decryptedToken, func(token *jwt.Token) (any, error) {
		// Only allow HS256
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(machineID), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return false, ErrNotActivated
		}
		return false, errors.New("Activation error: invalid token")
	}
	if !token.Valid {
		return false, ErrNotActivated
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, fmt.Errorf("Activation error: invalid claims type")
	}
	// Check machine_id claim exists and matches
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

		return false, err
	}

	info, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {

		return false, fmt.Errorf("Activation error: failed to determine core info")
	}

	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return false, err
	}

	params := rpc_flarewifi_v1.MachineActivationRequest{
		MachineId:      machineuid.GetMachineUID(),
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

	act, err := srv.MachineActivation(ctx, &params)
	if err != nil {

		return false, ErrNetworkIssue
	}

	if act.Activated {
		token := act.Token
		secret := cfg.Secret
		encrypted, err := crypt.EncryptToken(token, secret)
		if err != nil {
			return false, ErrNotActivated
		}
		if err := sdkutils.FsWriteFile(activationFile, []byte(encrypted)); err != nil {
			return false, ErrNotActivated
		}
		return true, nil
	}

	return false, ErrNotActivated
}
