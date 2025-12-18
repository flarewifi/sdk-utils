package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"core/tools/config"

	"github.com/golang-jwt/jwt/v5"
)

// WebhookClaims represents JWT claims for internal webhook authentication
type WebhookClaims struct {
	DeviceID    int64  `json:"device_id"`
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
