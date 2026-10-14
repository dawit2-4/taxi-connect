package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access.
// This is defined here to avoid import cycles with the user package.
// We use interface{} for the user type to avoid needing to define the actual user struct.
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (interface{}, error)
}

// JWTMiddleware validates JWT tokens and sets user ID in context
func JWTMiddleware(userRepo UserRepository, jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// 2. Check if it's a Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			// 3. Parse and validate token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			})
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// 4. Extract claims and validate
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// 5. Extract user ID from claims (using "sub" standard claim)
				userIDStr, ok := claims["sub"].(string)
				if !ok || userIDStr == "" {
					http.Error(w, "Invalid token: missing user ID", http.StatusUnauthorized)
					return
				}

				// 6. Convert user ID string to UUID
				userID, err := uuid.Parse(userIDStr)
				if err != nil {
					http.Error(w, "Invalid token: invalid user ID format", http.StatusUnauthorized)
					return
				}

				// 7. Verify user exists
				if _, err := userRepo.GetUserByID(r.Context(), userID); err != nil {
					http.Error(w, "Invalid token: user not found", http.StatusUnauthorized)
					return
				}

				// 8. Set user ID in context
				ctx := context.WithValue(r.Context(), "userID", userID)
				r = r.WithContext(ctx)

				// 9. Call next handler
				next.ServeHTTP(w, r)
				return
			}

			// 10. Handle invalid token
			http.Error(w, "Invalid token", http.StatusUnauthorized)
		})
	}
}