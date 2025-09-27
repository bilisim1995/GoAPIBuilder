package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// GetLinks returns service links for the specified institution
func GetLinks(w http.ResponseWriter, r *http.Request) {
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        // Get query parameters
        kurumID := r.URL.Query().Get("kurum_id")
        if kurumID == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "kurum_id parameter is required")
                return
        }

        // Convert kurum_id to ObjectID
        kurumObjectID, err := primitive.ObjectIDFromHex(kurumID)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid kurum_id format")
                return
        }

        // Get links collection
        collection := config.GetLinksCollection(mongoClient)

        // Find all links for the specified kurum_id
        filter := bson.M{"kurum_id": kurumObjectID}
        cursor, err := collection.Find(ctx, filter)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch links: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        // Decode results
        var links []models.Link
        if err := cursor.All(ctx, &links); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode links: "+err.Error())
                return
        }

        // Prepare response
        response := models.APIResponse{
                Success: true,
                Data:    links,
                Count:   len(links),
                Message: "Kurum linkleri başarıyla çekildi",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}