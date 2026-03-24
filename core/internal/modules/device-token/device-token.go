package devicetoken

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	machineuid "core/internal/modules/machine-uid"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DeviceTokenKey is the shared identifier for device tokens across cookies, localStorage, and headers
	// This ensures consistency between frontend (localStorage) and backend (cookies/headers)
	DeviceTokenKey = "flare_device_token"

	// HeaderName is the name of the HTTP header that carries the device JWT token
	HeaderName = "X-Device-Token"

	// LocalStorageKey is the key used in browser localStorage (documented for frontend reference)
	// JavaScript should use: localStorage.getItem('flare_device_token')
	LocalStorageKey = DeviceTokenKey
)

// getCookieName returns the dynamic cookie name based on machine ID
// Format: flare_device_token_{last4chars_lowercase}
func getCookieName() string {
	suffix := GetCookieNameSuffix()
	if suffix == "" {
		return DeviceTokenKey
	}
	return DeviceTokenKey + "_" + suffix
}

// GetCookieNameSuffix returns the last 4 chars of machine ID (lowercase)
// Used by templates to sync localStorage key with cookie name
// Returns empty string if machine ID is not available
func GetCookieNameSuffix() string {
	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		return ""
	}

	// Get last 4 characters and convert to lowercase
	if len(machineID) >= 4 {
		return strings.ToLower(machineID[len(machineID)-4:])
	}
	return strings.ToLower(machineID)
}

// GetDeviceTokenKey returns the complete device token key for use in both cookies and localStorage
// Format: flare_device_token_{last4chars_lowercase}
// Returns base key if machine ID is not available
func GetDeviceTokenKey() string {
	return getCookieName()
}

// GenerateDeviceToken creates a JWT token containing the device ID and cookie token
// The token is signed with the machine ID and expires in 10 years
func GenerateDeviceToken(deviceID int64, cookieToken string, machineID string) (string, error) {
	claims := jwt.MapClaims{
		"device_id":    deviceID,
		"cookie_token": cookieToken,
		"exp":          time.Now().UTC().Add(10 * 365 * 24 * time.Hour).Unix(), // 10 years
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(machineID))
}

// DeviceTokenClaims holds the parsed claims from a device token
type DeviceTokenClaims struct {
	DeviceID    int64
	CookieToken string
}

// VerifyDeviceToken verifies the JWT token and returns the device ID and cookie token
// Returns error if token is invalid or expired
func VerifyDeviceToken(tokenString string, machineID string) (DeviceTokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(machineID), nil
	})

	if err != nil {
		return DeviceTokenClaims{}, err
	}

	if !token.Valid {
		return DeviceTokenClaims{}, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return DeviceTokenClaims{}, fmt.Errorf("invalid token claims")
	}

	// Extract device_id from claims
	deviceIDClaim, ok := claims["device_id"]
	if !ok {
		return DeviceTokenClaims{}, fmt.Errorf("device_id not found in token")
	}

	// Handle both float64 (JSON number) and string representations
	var deviceID int64
	switch v := deviceIDClaim.(type) {
	case float64:
		deviceID = int64(v)
	case string:
		deviceID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return DeviceTokenClaims{}, fmt.Errorf("invalid device_id format: %w", err)
		}
	default:
		return DeviceTokenClaims{}, fmt.Errorf("device_id has unexpected type: %T", v)
	}

	// Extract cookie_token (optional - old tokens may not have it)
	cookieToken, _ := claims["cookie_token"].(string)

	return DeviceTokenClaims{
		DeviceID:    deviceID,
		CookieToken: cookieToken,
	}, nil
}

// SetDeviceCookie sets the JWT device cookie
func SetDeviceCookie(w http.ResponseWriter, deviceID int64, cookieToken string) error {
	_, machineID := machineuid.GetMachineUID()
	token, err := GenerateDeviceToken(deviceID, cookieToken, machineID)
	if err != nil {
		return fmt.Errorf("failed to generate device token: %w", err)
	}

	cookie := &http.Cookie{
		Name:     getCookieName(),
		Value:    token,
		Path:     "/",
		MaxAge:   315360000,                                       // 10 years in seconds
		Expires:  time.Now().UTC().Add(10 * 365 * 24 * time.Hour), // 10 years
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

// GetDeviceCookie retrieves and verifies the device cookie, returning the parsed claims
func GetDeviceCookie(r *http.Request) (DeviceTokenClaims, error) {
	cookie, err := r.Cookie(getCookieName())
	if err != nil {
		return DeviceTokenClaims{}, err
	}

	_, machineID := machineuid.GetMachineUID()
	claims, err := VerifyDeviceToken(cookie.Value, machineID)
	if err != nil {
		return DeviceTokenClaims{}, err
	}

	return claims, nil
}

// GetDeviceFromHeader retrieves and verifies the device token from X-Device-Token header
// Returns the parsed claims or error if token is invalid/missing
func GetDeviceFromHeader(r *http.Request) (DeviceTokenClaims, error) {
	tokenString := r.Header.Get(HeaderName)
	if tokenString == "" {
		return DeviceTokenClaims{}, fmt.Errorf("no device token in header")
	}

	_, machineID := machineuid.GetMachineUID()
	claims, err := VerifyDeviceToken(tokenString, machineID)
	if err != nil {
		return DeviceTokenClaims{}, fmt.Errorf("invalid device token: %w", err)
	}

	return claims, nil
}
