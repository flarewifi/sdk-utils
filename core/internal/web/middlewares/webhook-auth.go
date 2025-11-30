package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	sdkapi "sdk/api"
	"tools/config"

	"github.com/golang-jwt/jwt/v5"
)

// WebhookClaims represents JWT claims for internal webhook authentication
type WebhookClaims struct {
	DeviceID    int64  `json:"device_id"`
	PurchaseUID string `json:"purchase_uid"`
	jwt.RegisteredClaims
}

// WebhookAuth verifies JWT token for internal webhook requests.
// Verifies the JWT token signed with application secret and adds device/purchase info to context.
func WebhookAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Missing Authorization header"}`))
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Invalid Authorization header format"}`))
				return
			}

			tokenString := parts[1]

			// Get application secret
			appCfg, err := config.ReadApplicationConfig()
			if err != nil {
				fmt.Println("WebhookAuth: Failed to read application config:", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Server configuration error"}`))
				return
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
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Invalid token"}`))
				return
			}

			// Extract claims
			if claims, ok := token.Claims.(*WebhookClaims); ok && token.Valid {
				fmt.Printf("WebhookAuth: Valid token for device %d, purchase %s\n", claims.DeviceID, claims.PurchaseUID)

				// Add claims to context
				ctx := context.WithValue(r.Context(), sdkapi.WebhookDeviceIDKey, claims.DeviceID)
				ctx = context.WithValue(ctx, sdkapi.WebhookPurchaseUIDKey, claims.PurchaseUID)

				// Continue with the authenticated request
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				fmt.Println("WebhookAuth: Invalid token claims")
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error": "Invalid token claims"}`))
				return
			}
		})
	}
}
