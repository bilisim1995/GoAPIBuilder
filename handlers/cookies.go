package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"legal-documents-api/models"
)

// ClearCookies clears all cookies from the client
func ClearCookies(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Common cookie names to clear
	cookieNames := []string{
		"sessionToken",
		"authToken", 
		"refreshToken",
		"userSession",
		"jwt",
		"token",
		"JSESSIONID",
		"PHPSESSID",
		"connect.sid",
	}

	// Clear each cookie by setting it to empty with past expiration
	for _, cookieName := range cookieNames {
		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0), // January 1, 1970 UTC
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false, // Set to true if using HTTPS in production
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}

	// Also clear with different path variations
	pathVariations := []string{"/", "/api", "/api/v1"}
	for _, path := range pathVariations {
		for _, cookieName := range cookieNames {
			cookie := &http.Cookie{
				Name:     cookieName,
				Value:    "",
				Path:     path,
				Expires:  time.Unix(0, 0),
				MaxAge:   -1,
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, cookie)
		}
	}

	// Prepare response
	response := models.APIResponse{
		Success: true,
		Message: "Tüm çerezler başarıyla temizlendi",
		Data:    map[string]interface{}{
			"cleared_cookies": cookieNames,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ClearSpecificCookie clears a specific cookie by name
func ClearSpecificCookie(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get cookie name from query parameter
	cookieName := r.URL.Query().Get("name")
	if cookieName == "" {
		response := models.APIResponse{
			Success: false,
			Error:   "Cookie name parameter is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Clear the specific cookie with different path variations
	pathVariations := []string{"/", "/api", "/api/v1"}
	for _, path := range pathVariations {
		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     path,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false, // Set to true if using HTTPS in production
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}

	// Prepare response
	response := models.APIResponse{
		Success: true,
		Message: "Çerez başarıyla temizlendi: " + cookieName,
		Data:    map[string]interface{}{
			"cleared_cookie": cookieName,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}