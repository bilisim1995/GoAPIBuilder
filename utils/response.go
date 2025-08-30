package utils

import (
        "encoding/json"
        "log"
        "net/http"

        "legal-documents-api/models"
)

// SendErrorResponse sends a standardized error response
func SendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
        response := models.APIResponse{
                Success: false,
                Error:   message,
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(statusCode)
        
        if err := json.NewEncoder(w).Encode(response); err != nil {
                log.Printf("Error encoding error response: %v", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
        }
}

// SendSuccessResponse sends a standardized success response
func SendSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
        response := models.APIResponse{
                Success: true,
                Data:    data,
                Message: message,
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        
        if err := json.NewEncoder(w).Encode(response); err != nil {
                log.Printf("Error encoding success response: %v", err)
                SendErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
        }
}
