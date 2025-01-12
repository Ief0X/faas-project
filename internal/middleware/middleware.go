package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

var JwtSecret = []byte("kohi")

// JWTMiddleware checks the token in the Authorization header
func JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			JSONResponse(w, http.StatusUnauthorized, "Authorization header missing")
			return
		}

		// Token should be in the format "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			JSONResponse(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			JSONResponse(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Call the next handler if the token is valid
		next(w, r)
	}
}

// Send JSON responses
func JSONResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
