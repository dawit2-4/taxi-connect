package user

import (
	"time"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

// RegisterHandler handles user registration
func RegisterHandler(userService *UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}

		id, err := userService.RegisterUser(r.Context(), req.Email, req.Password)
		if err != nil {
			// Check if it's a duplicate email error
			if err.Error() == "failed to create user: user with this email already exists" {
				http.Error(w, "Email already registered", http.StatusConflict)
				return
			}
			http.Error(w, "Registration failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id": id.String(),
		})
	}
}

// LoginHandler handles user login
func LoginHandler(userService *UserService, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}

		userID, err := userService.AuthenticateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Generate JWT token
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["sub"] = userID.String()
		claims["iat"] = time.Now().Unix()
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // 24 hour expiration

		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			http.Error(w, "Failed to generate token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": tokenString,
		})
	}
}

// MeHandler returns current user's profile
func MeHandler(userService *UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userIDValue := r.Context().Value("userID")
		if userIDValue == nil {
			http.Error(w, "User ID not found in context", http.StatusUnauthorized)
			return
		}

		userID, ok := userIDValue.(uuid.UUID)
		if !ok {
			http.Error(w, "Invalid user ID in context", http.StatusUnauthorized)
			return
		}

		user, err := userService.GetUserByID(r.Context(), userID)
		if err != nil {
			if err == ErrUserNotFound {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Return user data without password
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"created_at": user.CreatedAt,
			"is_active": user.IsActive,
			"reputation_score": user.ReputationScore,
			"report_count": user.ReportCount,
			"accuracy_rate": user.AccuracyRate,
		})
	}
}