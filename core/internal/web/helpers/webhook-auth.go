package helpers

import (
	"errors"
	"fmt"
	"time"

	"core/utils/config"

	"github.com/golang-jwt/jwt/v5"
)

// PurchaseClaims represents JWT claims for purchase authentication
type PurchaseClaims struct {
	DeviceID    int64  `json:"device_id"`
	PurchaseUID string `json:"purchase_uid"`
	jwt.RegisteredClaims
}

// CreatePurchaseToken creates a JWT token for purchase authentication.
// The token contains the device ID and purchase UUID, signed with application secret.
// Token expires in 5 minutes.
func CreatePurchaseToken(deviceID int64, purchaseUUID string) (string, error) {
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		return "", fmt.Errorf("failed to read application config: %w", err)
	}

	now := time.Now()
	claims := PurchaseClaims{
		DeviceID:    deviceID,
		PurchaseUID: purchaseUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			Issuer:    "flarehotspot-core",
			Subject:   "purchase-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(appCfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign purchase token: %w", err)
	}

	return tokenString, nil
}

// VerifyPurchaseToken verifies a purchase JWT token and returns the claims.
// Returns the claims if valid, or an error if verification fails.
func VerifyPurchaseToken(tokenString string) (*PurchaseClaims, error) {
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		return nil, errors.New("server configuration error")
	}

	token, err := jwt.ParseWithClaims(tokenString, &PurchaseClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(appCfg.Secret), nil
	})

	if err != nil {
		return nil, errors.New("invalid purchase token")
	}

	if claims, ok := token.Claims.(*PurchaseClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}
