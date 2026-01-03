package jsonwebtoken

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(clms map[string]string, secret string) (string, error) {
	claims := jwt.MapClaims{}
	for k, v := range clms {
		claims[k] = v
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// secret is a []byte containing your secret, e.g. []byte("my_secret_key")
	return token.SignedString([]byte(secret))
}

func VerifyToken(clms string, secret string) (*jwt.Token, error) {
	token, err := jwt.Parse(clms, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// secret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
