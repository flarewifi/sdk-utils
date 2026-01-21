package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	machineuid "core/internal/modules/machine-uid"
	"core/utils/config"

	"github.com/golang-jwt/jwt/v5"
)

// WebhookClaims represents JWT claims for internal webhook authentication
type WebhookClaims struct {
	DeviceID    int64  `json:"device_id"`
	PurchaseUID string `json:"purchase_uid"`
	jwt.RegisteredClaims
}

// CallbackClaims represents JWT claims for purchase callback authentication
type CallbackClaims struct {
	PurchaseUID string `json:"purchase_uid"`
	jwt.RegisteredClaims
}

// WebhookAuth verifies JWT token for internal webhook requests.
// Verifies the JWT token signed with application secret.
// Returns nil if authentication is successful, or an error if it fails.
func WebhookAuth(r *http.Request) error {
	// Get Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return errors.New("missing Authorization header")
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return errors.New("invalid Authorization header format")
	}

	tokenString := parts[1]

	// Get application secret
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		fmt.Println("WebhookAuth: Failed to read application config:", err)
		return errors.New("server configuration error")
	}

	// Parse and verify token
	token, err := jwt.ParseWithClaims(tokenString, &WebhookClaims{}, func(token *jwt.Token) (any, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(appCfg.Secret), nil
	})

	if err != nil {
		fmt.Println("WebhookAuth: Token verification failed:", err)
		return errors.New("invalid token")
	}

	// Extract claims
	if claims, ok := token.Claims.(*WebhookClaims); ok && token.Valid {
		fmt.Printf("WebhookAuth: Valid token for device %d, purchase %s\n", claims.DeviceID, claims.PurchaseUID)
		return nil
	}

	fmt.Println("WebhookAuth: Invalid token claims")
	return errors.New("invalid token claims")
}

// ExtractWebhookClaims extracts and returns the claims from a webhook JWT token.
// Returns the claims if valid, or an error if verification fails.
func ExtractWebhookClaims(r *http.Request) (*WebhookClaims, error) {
	// Get Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing Authorization header")
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, errors.New("invalid Authorization header format")
	}

	tokenString := parts[1]

	// Get application secret
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		fmt.Println("ExtractWebhookClaims: Failed to read application config:", err)
		return nil, errors.New("server configuration error")
	}

	// Parse and verify token
	token, err := jwt.ParseWithClaims(tokenString, &WebhookClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(appCfg.Secret), nil
	})

	if err != nil {
		fmt.Println("ExtractWebhookClaims: Token verification failed:", err)
		return nil, errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*WebhookClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// getCallbackSecret returns the secret used for signing callback tokens.
// It combines machine_id + application secret for enhanced security.
func getCallbackSecret() (string, error) {
	_, machineID := machineuid.GetMachineUID()
	appCfg, err := config.ReadApplicationConfig()
	if err != nil {
		return "", fmt.Errorf("failed to read application config: %w", err)
	}
	return machineID + appCfg.Secret, nil
}

// CreateCallbackToken creates a JWT token for purchase callback authentication.
// The token contains the purchase UUID and is signed with machine_id + application secret.
func CreateCallbackToken(purchaseUUID string) (string, error) {
	secret, err := getCallbackSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := CallbackClaims{
		PurchaseUID: purchaseUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)), // 10 minute expiry for callbacks
			Issuer:    "flarehotspot-core",
			Subject:   "callback-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign callback token: %w", err)
	}

	return tokenString, nil
}

// VerifyCallbackToken verifies a callback JWT token and returns the purchase UUID.
// Returns the purchase UUID if valid, or an error if verification fails.
func VerifyCallbackToken(tokenString string) (string, error) {
	secret, err := getCallbackSecret()
	if err != nil {
		fmt.Println("VerifyCallbackToken: Failed to get secret:", err)
		return "", errors.New("server configuration error")
	}

	token, err := jwt.ParseWithClaims(tokenString, &CallbackClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		fmt.Println("VerifyCallbackToken: Token verification failed:", err)
		return "", errors.New("invalid callback token")
	}

	if claims, ok := token.Claims.(*CallbackClaims); ok && token.Valid {
		fmt.Printf("VerifyCallbackToken: Valid token for purchase %s\n", claims.PurchaseUID)
		return claims.PurchaseUID, nil
	}

	fmt.Println("VerifyCallbackToken: Invalid token claims")
	return "", errors.New("invalid callback token claims")
}
