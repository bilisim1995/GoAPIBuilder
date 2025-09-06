package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
)

// BasicAuth middleware for root endpoint authentication
func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get credentials from environment variables
		username := os.Getenv("API_USERNAME")
		password := os.Getenv("API_PASSWORD")
		
		// Default credentials if not set in environment
		if username == "" {
			username = "admin"
		}
		if password == "" {
			password = "mevzuat2025"
		}

		// Get credentials from request
		user, pass, ok := r.BasicAuth()
		
		// Check if credentials are valid
		if !ok || 
		   subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
		   subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			
			w.Header().Set("WWW-Authenticate", `Basic realm="Legal Documents API"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized: Please provide valid credentials"))
			return
		}
		
		// Call the next handler
		next(w, r)
	}
}