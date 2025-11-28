package devicetoken

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	machineuid "core/internal/utils/machine-uid"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// DeviceTokenKey is the shared identifier for device tokens across cookies, localStorage, and headers
	// This ensures consistency between frontend (localStorage) and backend (cookies/headers)
	DeviceTokenKey = "flare_device_token"

	// CookieName is the name of the HTTP cookie that stores the device JWT token
	CookieName = DeviceTokenKey

	// HeaderName is the name of the HTTP header that carries the device JWT token
	HeaderName = "X-Device-Token"

	// LocalStorageKey is the key used in browser localStorage (documented for frontend reference)
	// JavaScript should use: localStorage.getItem('flare_device_token')
	LocalStorageKey = DeviceTokenKey
)

// GenerateDeviceToken creates a JWT token containing the device ID
// The token is signed with the machine ID and expires in 10 years
func GenerateDeviceToken(deviceID int64, machineID string) (string, error) {
	claims := jwt.MapClaims{
		"device_id": deviceID,
		"exp":       time.Now().Add(10 * 365 * 24 * time.Hour).Unix(), // 10 years
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(machineID))
}

// VerifyDeviceToken verifies the JWT token and returns the device ID
// Returns error if token is invalid or expired
func VerifyDeviceToken(tokenString string, machineID string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(machineID), nil
	})

	if err != nil {
		return 0, err
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	// Extract device_id from claims
	deviceIDClaim, ok := claims["device_id"]
	if !ok {
		return 0, fmt.Errorf("device_id not found in token")
	}

	// Handle both float64 (JSON number) and string representations
	var deviceID int64
	switch v := deviceIDClaim.(type) {
	case float64:
		deviceID = int64(v)
	case string:
		deviceID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid device_id format: %w", err)
		}
	default:
		return 0, fmt.Errorf("device_id has unexpected type: %T", v)
	}

	return deviceID, nil
}

// SetDeviceCookie sets the JWT device cookie
func SetDeviceCookie(w http.ResponseWriter, deviceID int64) error {
	machineID := machineuid.GetMachineUID()
	token, err := GenerateDeviceToken(deviceID, machineID)
	if err != nil {
		return fmt.Errorf("failed to generate device token: %w", err)
	}

	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   315360000,                                 // 10 years in seconds
		Expires:  time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

// GetDeviceCookie retrieves and verifies the device cookie, returning the device ID
func GetDeviceCookie(r *http.Request) (int64, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return 0, err
	}

	machineID := machineuid.GetMachineUID()
	deviceID, err := VerifyDeviceToken(cookie.Value, machineID)
	if err != nil {
		return 0, err
	}

	return deviceID, nil
}

// GetDeviceFromHeader retrieves and verifies the device token from X-Device-Token header
// Returns the device ID or error if token is invalid/missing
func GetDeviceFromHeader(r *http.Request) (int64, error) {
	tokenString := r.Header.Get(HeaderName)
	if tokenString == "" {
		return 0, fmt.Errorf("no device token in header")
	}

	machineID := machineuid.GetMachineUID()
	deviceID, err := VerifyDeviceToken(tokenString, machineID)
	if err != nil {
		return 0, fmt.Errorf("invalid device token: %w", err)
	}

	return deviceID, nil
}
